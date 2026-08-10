package httpserver

import (
	"net/http"
	"strings"

	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
	"github.com/go-chi/chi/v5"
)

func mountAttachmentRoutes(router chi.Router, service attachmentapp.Service) {
	router.Post("/attachments", createAttachment(service))
	router.Post("/attachments/{attachmentID}/complete", completeAttachment(service))
	router.Get("/attachments", listAttachments(service))
}

func listAttachments(service attachmentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		entityKind := strings.TrimSpace(r.URL.Query().Get("entity_kind"))
		entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
		if !validUUIDParam(w, entityID, "entity ID") {
			return
		}
		items, err := service.ListByEntity(r.Context(), principal, entityKind, entityID)
		if err != nil {
			writeAttachmentError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createAttachment(service attachmentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[attachmentapp.CreateManifest](w, r)
		if !ok {
			return
		}
		result, replay, err := service.Create(
			r.Context(), principal, idempotencyKey(r), command,
		)
		if err != nil {
			writeAttachmentError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, result, replay)
	}
}

func completeAttachment(service attachmentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		attachmentID := chi.URLParam(r, "attachmentID")
		if !validUUIDParam(w, attachmentID, "attachment ID") {
			return
		}
		result, replay, err := service.Complete(
			r.Context(), principal, idempotencyKey(r), attachmentID,
		)
		if err != nil {
			writeAttachmentError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, result, replay)
	}
}

func writeAttachmentError(w http.ResponseWriter, err error) {
	writeDomainError(w, err, attachmentapp.ErrForbidden, attachmentapp.ErrNotFound,
		attachmentapp.ErrInvalid, attachmentapp.ErrConflict)
}
