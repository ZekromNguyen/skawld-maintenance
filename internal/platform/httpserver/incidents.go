package httpserver

import (
	"net/http"
	"strings"

	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
	customfieldapp "github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	"github.com/go-chi/chi/v5"
)

func mountIncidentRoutes(
	router chi.Router,
	incidents incidentapp.Service,
	executions executionapp.Service,
	attachments attachmentapp.Service,
	customFields customfieldapp.Service,
) {
	router.Get("/incidents", listIncidents(incidents, customFields))
	router.Post("/incidents", createIncident(incidents))
	router.Get("/incidents/{incidentID}", getIncident(incidents, attachments, customFields))
	router.Post("/incidents/{incidentID}/resolution", resolveIncident(incidents))
	router.Post("/incidents/{incidentID}/close", closeIncident(incidents))
	router.Post("/incidents/{incidentID}/reopen", reopenIncident(incidents))
	router.Post("/incidents/{incidentID}/executions", createExecution(executions))
}

func listIncidents(service incidentapp.Service, customFields customfieldapp.Service) http.HandlerFunc {
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
			Status:   r.URL.Query().Get("status"),
			PageSize: pageSize,
			Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		}
		for key, values := range r.URL.Query() {
			if strings.HasPrefix(key, "custom_field.") && len(values) > 0 {
				if filter.CustomFields == nil {
					filter.CustomFields = map[string]string{}
				}
				filter.CustomFields[strings.TrimPrefix(key, "custom_field.")] = values[0]
			}
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
		definitions, err := customFields.Definitions(r.Context(), principal.OrganizationID, "incident")
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":         items,
			"custom_fields": definitions,
			"next_cursor":   nullableString(next),
			"has_more":      next != "",
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

func getIncident(incidents incidentapp.Service, attachments attachmentapp.Service, customFields customfieldapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, id, "incident ID") {
			return
		}
		result, err := incidents.Get(r.Context(), principal, id)
		if err != nil {
			writeDomainError(w, err, incidentapp.ErrForbidden, incidentapp.ErrNotFound,
				incidentapp.ErrInvalid, incidentapp.ErrVersionConflict)
			return
		}
		items, err := attachments.ListByEntity(r.Context(), principal, "INCIDENT", id)
		if err != nil {
			writeDomainError(w, err, attachmentapp.ErrForbidden, attachmentapp.ErrNotFound, attachmentapp.ErrInvalid, attachmentapp.ErrConflict)
			return
		}
		definitions, err := customFields.Definitions(r.Context(), principal.OrganizationID, "incident")
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			incidentapp.Incident
			Attachments  []attachmentapp.Attachment  `json:"attachments"`
			CustomFields []customfieldapp.Definition `json:"custom_fields"`
		}{result, items, definitions})
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

func closeIncident(service incidentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, id, "incident ID") {
			return
		}
		command, ok := decodeCommand[incidentapp.CloseIncident](w, r)
		if !ok {
			return
		}
		result, replay, err := service.Close(
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

func reopenIncident(service incidentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, id, "incident ID") {
			return
		}
		command, ok := decodeCommand[incidentapp.ReopenIncident](w, r)
		if !ok {
			return
		}
		result, replay, err := service.Reopen(
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
