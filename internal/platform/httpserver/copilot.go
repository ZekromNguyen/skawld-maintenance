package httpserver

import (
	"errors"
	"net/http"

	copilotapp "github.com/ZekromNguyen/skawld-maintenance/internal/copilot/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/go-chi/chi/v5"
)

func mountCopilotRoutes(router chi.Router, service copilotapp.Service) {
	router.Post("/incidents/{incidentID}/recommendations", generateRecommendation(service))
	router.Get("/recommendations/{recommendationID}", getRecommendation(service))
	router.Post("/recommendations/{recommendationID}/feedback", recommendationFeedback(service))
}

func generateRecommendation(service copilotapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		incidentID := chi.URLParam(r, "incidentID")
		if !validUUIDParam(w, incidentID, "incident ID") {
			return
		}
		command, ok := decodeCommand[copilotapp.GenerateRecommendation](w, r)
		if !ok {
			return
		}
		value, replay, err := service.Generate(
			r.Context(), principal, idempotencyKey(r), incidentID, command,
		)
		if err != nil {
			writeCopilotError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, value, replay)
	}
}

func getRecommendation(service copilotapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		recommendationID := chi.URLParam(r, "recommendationID")
		if !validUUIDParam(w, recommendationID, "recommendation ID") {
			return
		}
		value, err := service.Get(r.Context(), principal, recommendationID)
		if err != nil {
			writeCopilotError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func recommendationFeedback(service copilotapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		recommendationID := chi.URLParam(r, "recommendationID")
		if !validUUIDParam(w, recommendationID, "recommendation ID") {
			return
		}
		command, ok := decodeCommand[copilotapp.Feedback](w, r)
		if !ok {
			return
		}
		err := service.Feedback(
			r.Context(), principal, idempotencyKey(r), recommendationID, command,
		)
		if err != nil {
			writeCopilotError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeCopilotError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, skawld.ErrProviderUnavailable):
		writeProblem(w, http.StatusServiceUnavailable, "Capability Unavailable", err.Error())
	case errors.Is(err, copilotapp.ErrEvidenceInvalid),
		errors.Is(err, copilotapp.ErrUnsafeOutput):
		writeProblem(w, http.StatusUnprocessableEntity, "Model Output Rejected", err.Error())
	case errors.Is(err, idempotency.ErrKeyConflict),
		errors.Is(err, idempotency.ErrInProgress):
		writeProblem(w, http.StatusConflict, "Conflict", err.Error())
	default:
		writeDomainError(w, err, copilotapp.ErrForbidden, copilotapp.ErrNotFound,
			copilotapp.ErrInvalid, idempotency.ErrKeyConflict)
	}
}
