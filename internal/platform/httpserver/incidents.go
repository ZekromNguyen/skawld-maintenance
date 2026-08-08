package httpserver

import (
	"net/http"
	"strings"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	"github.com/go-chi/chi/v5"
)

func mountIncidentRoutes(
	router chi.Router,
	incidents incidentapp.Service,
	executions executionapp.Service,
) {
	router.Get("/incidents", listIncidents(incidents))
	router.Post("/incidents", createIncident(incidents))
	router.Get("/incidents/{incidentID}", getIncident(incidents))
	router.Post("/incidents/{incidentID}/resolution", resolveIncident(incidents))
	router.Post("/incidents/{incidentID}/executions", createExecution(executions))
}

func listIncidents(service incidentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		pageSize, ok := parsePageSize(w, r)
		if !ok {
			return
		}
		filter := incidentapp.Filter{
			SiteID:   r.URL.Query().Get("site_id"),
			AssetID:  r.URL.Query().Get("asset_id"),
			State:    r.URL.Query().Get("state"),
			Severity: r.URL.Query().Get("severity"),
			PageSize: pageSize,
			Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		}
		if filter.SiteID != "" && !validUUIDParam(w, filter.SiteID, "site ID") {
			return
		}
		items, next, err := service.List(r.Context(), principal, filter)
		if err != nil {
			writeDomainError(w, err, incidentapp.ErrForbidden, incidentapp.ErrNotFound,
				incidentapp.ErrInvalid, incidentapp.ErrVersionConflict)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":       items,
			"next_cursor": nullableString(next),
			"has_more":    next != "",
		})
	}
}

func createIncident(service incidentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[incidentapp.CreateIncident](w, r)
		if !ok {
			return
		}
		result, replay, err := service.Create(
			r.Context(), principal, idempotencyKey(r), command,
		)
		if err != nil {
			writeDomainError(w, err, incidentapp.ErrForbidden, incidentapp.ErrNotFound,
				incidentapp.ErrInvalid, incidentapp.ErrVersionConflict)
			return
		}
		writeMutation(w, http.StatusCreated, result, replay)
	}
}

func getIncident(service incidentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, id, "incident ID") {
			return
		}
		result, err := service.Get(r.Context(), principal, id)
		if err != nil {
			writeDomainError(w, err, incidentapp.ErrForbidden, incidentapp.ErrNotFound,
				incidentapp.ErrInvalid, incidentapp.ErrVersionConflict)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func resolveIncident(service incidentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, id, "incident ID") {
			return
		}
		command, ok := decodeCommand[incidentapp.ResolveIncident](w, r)
		if !ok {
			return
		}
		result, replay, err := service.Resolve(
			r.Context(), principal, idempotencyKey(r), id, command,
		)
		if err != nil {
			writeDomainError(w, err, incidentapp.ErrForbidden, incidentapp.ErrNotFound,
				incidentapp.ErrInvalid, incidentapp.ErrVersionConflict)
			return
		}
		writeMutation(w, http.StatusOK, result, replay)
	}
}

func createExecution(service executionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		incidentID := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, incidentID, "incident ID") {
			return
		}
		command, ok := decodeCommand[executionapp.CreateExecution](w, r)
		if !ok {
			return
		}
		if command.IncidentID != "" && command.IncidentID != incidentID {
			writeProblem(w, http.StatusBadRequest, "Invalid Request", "incident ID does not match route")
			return
		}
		command.IncidentID = incidentID
		result, replay, err := service.Create(
			r.Context(), principal, idempotencyKey(r), command,
		)
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, result, replay)
	}
}
