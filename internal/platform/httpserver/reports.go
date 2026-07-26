package httpserver

import (
	"context"
	"errors"
	"net/http"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/go-chi/chi/v5"
)

func mountReportRoutes(router chi.Router, service reportapp.Service) {
	router.Post("/executions/{executionID}/reports/draft", draftReport(service))
	router.Get("/reports/{reportID}", getReport(service))
	router.Post("/reports/{reportID}/edit", editReport(service))
	router.Post("/reports/{reportID}/submit", submitReport(service))
	router.Post("/reports/{reportID}/approve", approveReport(service))
}

func draftReport(service reportapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		executionID := chi.URLParam(r, "executionID")
		if !validUUIDParam(w, executionID, "execution ID") {
			return
		}
		value, replay, err := service.Draft(
			r.Context(), principal, idempotencyKey(r), executionID,
		)
		if err != nil {
			writeReportError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, value, replay)
	}
}

func getReport(service reportapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		reportID := chi.URLParam(r, "reportID")
		if !validUUIDParam(w, reportID, "report ID") {
			return
		}
		value, err := service.Get(r.Context(), principal, reportID)
		if err != nil {
			writeReportError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func editReport(service reportapp.Service) http.HandlerFunc {
	return reportMutation(service.Edit, http.StatusOK)
}

func submitReport(service reportapp.Service) http.HandlerFunc {
	return reportTransition(service.Submit)
}

func approveReport(service reportapp.Service) http.HandlerFunc {
	return reportTransition(service.Approve)
}

func reportMutation(
	mutate func(
		ctx context.Context, principal identitydomain.Principal,
		key, reportID string, command reportapp.Edit,
	) (reportapp.Report, bool, error),
	status int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		reportID := chi.URLParam(r, "reportID")
		if !validUUIDParam(w, reportID, "report ID") {
			return
		}
		command, ok := decodeCommand[reportapp.Edit](w, r)
		if !ok {
			return
		}
		value, replay, err := mutate(
			r.Context(), principal, idempotencyKey(r), reportID, command,
		)
		if err != nil {
			writeReportError(w, err)
			return
		}
		writeMutation(w, status, value, replay)
	}
}

func reportTransition(
	mutate func(
		ctx context.Context, principal identitydomain.Principal,
		key, reportID string, command reportapp.Transition,
	) (reportapp.Report, bool, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		reportID := chi.URLParam(r, "reportID")
		if !validUUIDParam(w, reportID, "report ID") {
			return
		}
		command, ok := decodeCommand[reportapp.Transition](w, r)
		if !ok {
			return
		}
		value, replay, err := mutate(
			r.Context(), principal, idempotencyKey(r), reportID, command,
		)
		if err != nil {
			writeReportError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, value, replay)
	}
}

func writeReportError(w http.ResponseWriter, err error) {
	if errors.Is(err, skawld.ErrProviderUnavailable) {
		writeProblem(w, http.StatusServiceUnavailable, "Capability Unavailable", err.Error())
		return
	}
	writeDomainError(w, err, reportapp.ErrForbidden, reportapp.ErrNotFound,
		reportapp.ErrInvalid, reportapp.ErrConflict)
}
