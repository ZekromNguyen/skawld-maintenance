package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/buildinfo"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Authenticator interface {
	Begin(http.ResponseWriter, *http.Request)
	Callback(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	Middleware(http.Handler) http.Handler
}

type Dependencies struct {
	Logger        *slog.Logger
	Database      *pgxpool.Pool
	Auth          Authenticator
	Organizations application.OrganizationService
	Sites         application.SiteService
}

func New(dependencies Dependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(securityHeaders)
	router.Use(accessLog(dependencies.Logger))

	router.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "live",
			"build":  buildinfo.Current(),
		})
	})
	router.Get("/health/ready", readiness(dependencies.Database))

	router.Get("/auth/login", dependencies.Auth.Begin)
	router.Get("/auth/callback", dependencies.Auth.Callback)
	router.Post("/auth/logout", dependencies.Auth.Logout)

	router.Route("/api/v1", func(api chi.Router) {
		api.Use(dependencies.Auth.Middleware)
		api.Get("/me", currentPrincipal)
		api.Post("/organizations", createOrganization(dependencies.Organizations))
		api.Get("/sites/{siteID}", getSite(dependencies.Sites))
	})
	return router
}

func getSite(service application.SiteService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := domain.PrincipalFromContext(r.Context())
		if !ok {
			writeProblem(w, http.StatusUnauthorized, "Unauthorized", "principal is missing")
			return
		}
		siteID := chi.URLParam(r, "siteID")
		if _, err := uuid.Parse(siteID); err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid Request", "site ID must be a UUID")
			return
		}
		site, err := service.Get(r.Context(), principal, siteID)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, site)
		case errors.Is(err, application.ErrSiteNotFound):
			writeProblem(w, http.StatusNotFound, "Not Found", "site was not found")
		default:
			writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "request could not be completed")
		}
	}
}

func readiness(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if pool == nil || pool.Ping(ctx) != nil {
			writeProblem(w, http.StatusServiceUnavailable, "Not Ready", "database is unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func currentPrincipal(w http.ResponseWriter, r *http.Request) {
	principal, ok := domain.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "principal is missing")
		return
	}
	permissions := make([]string, 0, len(principal.Permissions))
	for permission := range principal.Permissions {
		permissions = append(permissions, string(permission))
	}
	sort.Strings(permissions)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":               principal.ID,
		"external_subject": principal.ExternalSubject,
		"display_name":     principal.DisplayName,
		"organization_id":  principal.OrganizationID,
		"site_ids":         principal.SiteIDs,
		"permissions":      permissions,
	})
}

func createOrganization(service application.OrganizationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := domain.PrincipalFromContext(r.Context())
		if !ok {
			writeProblem(w, http.StatusUnauthorized, "Unauthorized", "principal is missing")
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		var command application.CreateOrganization
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&command); err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid Request", "request body must match the schema")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeProblem(w, http.StatusBadRequest, "Invalid Request", "request body must contain one JSON object")
			return
		}
		result, replay, err := service.Create(r.Context(), principal, key, command)
		switch {
		case err == nil:
			if replay {
				w.Header().Set("Idempotent-Replayed", "true")
				writeJSON(w, http.StatusOK, result)
				return
			}
			writeJSON(w, http.StatusCreated, result)
		case errors.Is(err, application.ErrPermissionDenied):
			writeProblem(w, http.StatusForbidden, "Forbidden", "permission denied")
		case errors.Is(err, application.ErrIdempotencyKey),
			errors.Is(err, application.ErrInvalidOrganization):
			writeProblem(w, http.StatusBadRequest, "Invalid Request", err.Error())
		case errors.Is(err, idempotency.ErrKeyConflict):
			writeProblem(w, http.StatusConflict, "Idempotency Conflict", err.Error())
		default:
			writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "request could not be completed")
		}
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapped, r)
			logger.InfoContext(r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"bytes", wrapped.BytesWritten(),
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
