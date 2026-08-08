package httpserver

import (
	"errors"
	"net/http"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/go-chi/chi/v5"
)

func mountCustomFieldRoutes(router chi.Router, fields application.Service) {
	router.Get("/admin/field-definitions", listFieldDefinitions(fields))
	router.Post("/admin/field-definitions", createFieldDefinition(fields))
	router.Get("/admin/field-definitions/{fieldID}", getFieldDefinition(fields))
	router.Patch("/admin/field-definitions/{fieldID}", updateFieldDefinition(fields))
	router.Post("/admin/field-definitions/{fieldID}/retire", retireFieldDefinition(fields))
	router.Get("/admin/field-definitions/{fieldID}/history", fieldDefinitionHistory(fields))
}

func listFieldDefinitions(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		entityType := r.URL.Query().Get("entity_type")
		if entityType == "" {
			entityType = "incident"
		}
		items, err := service.ListByEntity(r.Context(), principal, entityType)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[application.CreateDefinition](w, r)
		if !ok {
			return
		}
		result, err := service.Create(r.Context(), principal, command)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	}
}

func getFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		result, err := service.Get(r.Context(), principal, id)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func updateFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		command, ok := decodeCommand[application.UpdateDefinition](w, r)
		if !ok {
			return
		}
		result, err := service.Update(r.Context(), principal, id, command)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func retireFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		result, err := service.Retire(r.Context(), principal, id)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func fieldDefinitionHistory(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		items, err := service.History(r.Context(), principal, id)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func writeFieldError(w http.ResponseWriter, err error) {
	// ErrValidation is the one mapping the shared helper does not know;
	// everything else delegates to writeDomainError so the common
	// status/idempotency mapping cannot drift.
	if errors.Is(err, application.ErrValidation) {
		writeProblem(w, http.StatusUnprocessableEntity, "Unprocessable Entity", err.Error())
		return
	}
	writeDomainError(w, err,
		application.ErrForbidden, application.ErrNotFound,
		application.ErrInvalid, application.ErrConflict)
}
