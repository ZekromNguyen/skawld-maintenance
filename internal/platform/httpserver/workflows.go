package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
	"github.com/go-chi/chi/v5"
)

func mountWorkflowRoutes(router chi.Router, service workflowapp.Service) {
	router.Post("/workflow-candidates", compileWorkflow(service))
	router.Get("/workflows", listWorkflows(service))
	router.Get("/workflows/applicable", applicableWorkflows(service))
	router.Get(
		"/workflows/{workflowID}/versions/{version}",
		getWorkflowVersion(service),
	)
	router.Post(
		"/workflows/{workflowID}/versions/{version}/reviews",
		reviewWorkflowVersion(service),
	)
	router.Post(
		"/workflows/{workflowID}/versions/{version}/publish",
		publishWorkflowVersion(service),
	)
	router.Post(
		"/workflows/{workflowID}/versions/{version}/applicability",
		expandWorkflowApplicability(service),
	)
	router.Post(
		"/workflows/{workflowID}/versions/{version}/retire",
		retireWorkflowVersion(service),
	)
}

func compileWorkflow(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[workflowapp.Compile](w, r)
		if !ok {
			return
		}
		for _, demonstrationID := range command.DemonstrationIDs {
			if !validUUIDParam(w, demonstrationID, "demonstration ID") {
				return
			}
		}
		value, err := service.Compile(r.Context(), principal, command)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func listWorkflows(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		values, err := service.List(r.Context(), principal)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"items": values})
	}
}

func applicableWorkflows(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		assetID := r.URL.Query().Get("asset_id")
		if !validUUIDParam(w, assetID, "asset ID") {
			return
		}
		values, err := service.Applicable(r.Context(), principal, assetID)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"items": values})
	}
}

func getWorkflowVersion(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, workflowID, version, ok := workflowRequest(w, r)
		if !ok {
			return
		}
		value, err := service.Get(
			r.Context(), principal, workflowID, version,
		)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func reviewWorkflowVersion(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, workflowID, version, ok := workflowRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[workflowapp.Review](w, r)
		if !ok {
			return
		}
		for _, applicability := range command.Applicability {
			if applicability.SiteID != "" &&
				!validUUIDParam(w, applicability.SiteID, "site ID") {
				return
			}
			if applicability.AssetID != "" &&
				!validUUIDParam(w, applicability.AssetID, "asset ID") {
				return
			}
		}
		value, err := service.Review(
			r.Context(), principal, workflowID, version, command,
		)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func publishWorkflowVersion(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, workflowID, version, ok := workflowRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[workflowapp.Publish](w, r)
		if !ok {
			return
		}
		value, err := service.Publish(
			r.Context(), principal, workflowID, version, command,
		)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func expandWorkflowApplicability(
	service workflowapp.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, workflowID, version, ok := workflowRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[workflowapp.ExpandApplicability](w, r)
		if !ok {
			return
		}
		if command.Applicability.SiteID != "" &&
			!validUUIDParam(w, command.Applicability.SiteID, "site ID") {
			return
		}
		if command.Applicability.AssetID != "" &&
			!validUUIDParam(w, command.Applicability.AssetID, "asset ID") {
			return
		}
		value, err := service.ExpandApplicability(
			r.Context(), principal, workflowID, version, command,
		)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func retireWorkflowVersion(service workflowapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, workflowID, version, ok := workflowRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[workflowapp.Retire](w, r)
		if !ok {
			return
		}
		value, err := service.Retire(
			r.Context(), principal, workflowID, version, command,
		)
		if err != nil {
			writeWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func workflowRequest(
	w http.ResponseWriter,
	r *http.Request,
) (identitydomain.Principal, string, int, bool) {
	principal, ok := principalFromRequest(w, r)
	if !ok {
		return identitydomain.Principal{}, "", 0, false
	}
	workflowID := chi.URLParam(r, "workflowID")
	if !validUUIDParam(w, workflowID, "workflow ID") {
		return identitydomain.Principal{}, "", 0, false
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || version < 1 {
		writeProblem(
			w, http.StatusBadRequest, "Invalid Request",
			"workflow version must be a positive integer",
		)
		return identitydomain.Principal{}, "", 0, false
	}
	return principal, workflowID, version, true
}

func writeWorkflowError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workflowapp.ErrPolicy):
		writeProblem(
			w, http.StatusUnprocessableEntity,
			"Workflow Policy Failed",
			"candidate did not satisfy review, evaluation, or publication policy",
		)
	default:
		writeDomainError(
			w, err, workflowapp.ErrForbidden, workflowapp.ErrNotFound,
			workflowapp.ErrInvalid, workflowapp.ErrConflict,
		)
	}
}
