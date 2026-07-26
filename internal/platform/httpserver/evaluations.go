package httpserver

import (
	"errors"
	"net/http"

	evaluationapp "github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/application"
	"github.com/go-chi/chi/v5"
)

func mountEvaluationRoutes(
	router chi.Router,
	service evaluationapp.Service,
) {
	router.Get("/evaluations/summary", evaluationSummary(service))
}

func evaluationSummary(service evaluationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		value, err := service.Summary(r.Context(), principal)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, value)
		case errors.Is(err, evaluationapp.ErrForbidden):
			writeProblem(
				w, http.StatusForbidden, "Forbidden",
				"evaluation review permission is required",
			)
		default:
			writeProblem(
				w, http.StatusInternalServerError, "Internal Server Error",
				"evaluation summary could not be loaded",
			)
		}
	}
}
