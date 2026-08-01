package httpserver

import (
	"context"
	"net/http"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/go-chi/chi/v5"
)

func mountExecutionRoutes(router chi.Router, service executionapp.Service) {
	router.Get("/executions", listExecutions(service))
	router.Get("/executions/{executionID}", getExecution(service))
	router.Route("/executions/{executionID}", func(route chi.Router) {
		route.Post("/start", startExecution(service))
		route.Post("/prerequisite-verifications", verifyPrerequisite(service))
		route.Post("/steps/{stepID}/complete", completeExecutionStep(service))
		route.Post("/measurements", recordMeasurement(service))
		route.Post("/observations", recordObservation(service))
		route.Post("/actions", recordAction(service))
		route.Post("/decisions", recordDecision(service))
		route.Post("/complete", completeExecution(service))
	})
}

func listExecutions(service executionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		result, err := service.List(r.Context(), principal, executionapp.Filter{
			SiteID:     r.URL.Query().Get("site_id"),
			State:      r.URL.Query().Get("state"),
			AssignedTo: r.URL.Query().Get("assigned_to"),
		})
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": result})
	}
}

func getExecution(service executionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, executionID, ok := executionRequest(w, r)
		if !ok {
			return
		}
		result, err := service.Get(r.Context(), principal, executionID)
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func startExecution(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.Start, http.StatusOK)
}

func verifyPrerequisite(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.VerifyPrerequisite, http.StatusCreated)
}

func recordMeasurement(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.RecordMeasurement, http.StatusCreated)
}

func recordObservation(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.RecordObservation, http.StatusCreated)
}

func recordAction(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.RecordAction, http.StatusCreated)
}

func recordDecision(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.RecordDecision, http.StatusCreated)
}

func completeExecution(service executionapp.Service) http.HandlerFunc {
	return executionCommand(service.Complete, http.StatusOK)
}

func executionCommand[C any, R any](
	action func(
		ctx context.Context,
		principal identitydomain.Principal,
		key, executionID string,
		command C,
	) (R, bool, error),
	status int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, executionID, ok := executionRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[C](w, r)
		if !ok {
			return
		}
		result, replay, err := action(
			r.Context(), principal, idempotencyKey(r), executionID, command,
		)
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		writeMutation(w, status, result, replay)
	}
}

func completeExecutionStep(service executionapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, executionID, ok := executionRequest(w, r)
		if !ok {
			return
		}
		stepID := chi.URLParam(r, "stepID")
		if !validUUIDParam(w, stepID, "step ID") {
			return
		}
		command, ok := decodeCommand[executionapp.CompleteStep](w, r)
		if !ok {
			return
		}
		result, replay, err := service.CompleteStep(
			r.Context(), principal, idempotencyKey(r), executionID, stepID, command,
		)
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		if result.State == "BLOCKED" {
			if replay {
				w.Header().Set("Idempotent-Replay", "true")
			}
			writeJSON(w, http.StatusConflict, result)
			return
		}
		writeMutation(w, http.StatusOK, result, replay)
	}
}

func executionRequest(
	w http.ResponseWriter,
	r *http.Request,
) (identitydomain.Principal, string, bool) {
	principal, ok := principalFromRequest(w, r)
	if !ok {
		return identitydomain.Principal{}, "", false
	}
	executionID := chi.URLParam(r, "executionID")
	if !validUUIDParam(w, executionID, "execution ID") {
		return identitydomain.Principal{}, "", false
	}
	return principal, executionID, true
}

func writeExecutionError(w http.ResponseWriter, err error) {
	writeDomainError(w, err, executionapp.ErrForbidden, executionapp.ErrNotFound,
		executionapp.ErrInvalid, executionapp.ErrVersionConflict)
}
