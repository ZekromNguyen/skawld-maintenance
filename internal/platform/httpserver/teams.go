package httpserver

import (
	"net/http"

	teamapp "github.com/ZekromNguyen/skawld-maintenance/internal/team/application"
	"github.com/go-chi/chi/v5"
)

func mountTeamRoutes(router chi.Router, service teamapp.Service) {
	router.Get("/teams", listTeams(service))
	router.Get("/people", listPeople(service))
}

func listTeams(service teamapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		items, err := service.ListTeams(r.Context(), principal)
		if err != nil {
			writeDomainError(w, err, teamapp.ErrForbidden, teamapp.ErrInvalid, teamapp.ErrInvalid, nil)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func listPeople(service teamapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		items, err := service.ListPeople(r.Context(), principal)
		if err != nil {
			writeDomainError(w, err, teamapp.ErrForbidden, teamapp.ErrInvalid, teamapp.ErrInvalid, nil)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}
