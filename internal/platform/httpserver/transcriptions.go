package httpserver

import (
	"errors"
	"net/http"

	transcriptionapp "github.com/ZekromNguyen/skawld-maintenance/internal/transcription/application"
	"github.com/go-chi/chi/v5"
)

func mountTranscriptionRoutes(router chi.Router, service transcriptionapp.Service) {
	router.Post("/attachments/{attachmentID}/transcriptions", requestTranscription(service))
	router.Get("/transcriptions/{transcriptionID}", getTranscription(service))
	router.Post("/transcriptions/{transcriptionID}/verify", verifyTranscription(service))
}

func requestTranscription(service transcriptionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		attachmentID := chi.URLParam(r, "attachmentID")
		if !validUUIDParam(w, attachmentID, "attachment ID") {
			return
		}
		value, replay, err := service.Request(
			r.Context(), principal, idempotencyKey(r), attachmentID,
		)
		if err != nil {
			writeTranscriptionError(w, err)
			return
		}
		writeMutation(w, http.StatusAccepted, value, replay)
	}
}

func getTranscription(service transcriptionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "transcriptionID")
		if !validUUIDParam(w, id, "transcription ID") {
			return
		}
		value, err := service.Get(r.Context(), principal, id)
		if err != nil {
			writeTranscriptionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func verifyTranscription(service transcriptionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "transcriptionID")
		if !validUUIDParam(w, id, "transcription ID") {
			return
		}
		command, ok := decodeCommand[transcriptionapp.Verify](w, r)
		if !ok {
			return
		}
		value, replay, err := service.Verify(
			r.Context(), principal, idempotencyKey(r), id, command,
		)
		if err != nil {
			writeTranscriptionError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, value, replay)
	}
}

func writeTranscriptionError(w http.ResponseWriter, err error) {
	if errors.Is(err, transcriptionapp.ErrUnavailable) {
		writeProblem(w, http.StatusServiceUnavailable, "Capability Unavailable", err.Error())
		return
	}
	writeDomainError(w, err, transcriptionapp.ErrForbidden, transcriptionapp.ErrNotFound,
		transcriptionapp.ErrInvalid, transcriptionapp.ErrConflict)
}
