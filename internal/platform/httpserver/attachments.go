package httpserver

import (
	"net/http"

	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
	"github.com/go-chi/chi/v5"
)

func mountAttachmentRoutes(router chi.Router, service attachmentapp.Service) {
	router.Post("/attachments", createAttachment(service))
	router.Post("/attachments/{attachmentID}/complete", completeAttachment(service))
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
