package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/go-chi/chi/v5"
)

func mountHandoverRoutes(router chi.Router, service handoverapp.Service) {
	router.Get("/handovers", listHandovers(service))
	router.Post("/handovers/prepare-draft", prepareHandover(service))
	router.Get("/handovers/{handoverID}", getHandover(service))
	router.Post("/handovers/{handoverID}/edit", editHandover(service))
	router.Post("/handovers/{handoverID}/submit", handoverTransition(service.Submit))
	router.Post("/handovers/{handoverID}/accept", handoverTransition(service.Accept))
	router.Post("/handovers/{handoverID}/acknowledge", handoverTransition(service.Acknowledge))
}

func listHandovers(service handoverapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		pageSize, ok := parsePageSize(w, r)
		if !ok {
			return
		}
		filter := handoverapp.HandoverFilter{PageSize: pageSize}
		if siteID := r.URL.Query().Get("site_id"); siteID != "" {
			if !validUUIDParam(w, siteID, "site ID") {
				return
			}
			filter.SiteID = siteID
		}
		filter.States = r.URL.Query()["state"]
		filter.Cursor = strings.TrimSpace(r.URL.Query().Get("cursor"))
		result, err := service.List(r.Context(), principal, filter)
		if err != nil {
			writeHandoverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":       result.Items,
			"next_cursor": nullableString(result.NextCursor),
			"has_more":    result.HasMore,
		})
	}
}

func prepareHandover(service handoverapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[handoverapp.PrepareDraft](w, r)
		if !ok {
			return
		}
		value, replay, err := service.Prepare(
			r.Context(), principal, idempotencyKey(r), command,
		)
		if err != nil {
			writeHandoverError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, value, replay)
	}
}

func getHandover(service handoverapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "handoverID")
		if !validUUIDParam(w, id, "handover ID") {
			return
		}
		value, err := service.Get(r.Context(), principal, id)
		if err != nil {
			writeHandoverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func editHandover(service handoverapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "handoverID")
		if !validUUIDParam(w, id, "handover ID") {
			return
		}
		command, ok := decodeCommand[handoverapp.Edit](w, r)
		if !ok {
			return
		}
		value, replay, err := service.Edit(
			r.Context(), principal, idempotencyKey(r), id, command,
		)
		if err != nil {
			writeHandoverError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, value, replay)
	}
}

func handoverTransition(
	mutate func(
		context.Context, identitydomain.Principal, string, string,
		handoverapp.Transition,
	) (handoverapp.Handover, bool, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "handoverID")
		if !validUUIDParam(w, id, "handover ID") {
			return
		}
		command, ok := decodeCommand[handoverapp.Transition](w, r)
		if !ok {
			return
		}
		value, replay, err := mutate(
			r.Context(), principal, idempotencyKey(r), id, command,
		)
		if err != nil {
			writeHandoverError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, value, replay)
	}
}

func writeHandoverError(w http.ResponseWriter, err error) {
	if errors.Is(err, skawld.ErrProviderUnavailable) {
		writeProblem(w, http.StatusServiceUnavailable, "Capability Unavailable", err.Error())
		return
	}
	writeDomainError(w, err, handoverapp.ErrForbidden, handoverapp.ErrNotFound,
		handoverapp.ErrInvalid, handoverapp.ErrConflict)
}
