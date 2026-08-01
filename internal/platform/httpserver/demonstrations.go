package httpserver

import (
	"errors"
	"net/http"

	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/go-chi/chi/v5"
)

func mountDemonstrationRoutes(
	router chi.Router,
	service demonstrationapp.Service,
) {
	router.Post("/demonstrations", startDemonstration(service))
	router.Get("/demonstrations", listDemonstrations(service))
	router.Get("/demonstrations/{demonstrationID}", getDemonstration(service))
	router.Post("/demonstrations/{demonstrationID}/complete", completeDemonstration(service))
	router.Post(
		"/demonstrations/{demonstrationID}/evidence-views",
		recordDemonstrationEvidenceView(service),
	)
	router.Post("/demonstrations/{demonstrationID}/reviews", reviewDemonstration(service))
	router.Post(
		"/demonstrations/{demonstrationID}/events/{eventID}/redactions",
		redactDemonstrationEvent(service),
	)
}

func startDemonstration(service demonstrationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[demonstrationapp.Start](w, r)
		if !ok {
			return
		}
		if !validUUIDParam(w, command.SubjectID, "subject ID") {
			return
		}
		value, err := service.Start(r.Context(), principal, command)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func listDemonstrations(service demonstrationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		siteID := r.URL.Query().Get("site_id")
		if siteID != "" && !validUUIDParam(w, siteID, "site ID") {
			return
		}
		values, err := service.List(r.Context(), principal, siteID)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": values})
	}
}

func getDemonstration(service demonstrationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, demonstrationID, ok := demonstrationRequest(w, r)
		if !ok {
			return
		}
		value, err := service.Get(r.Context(), principal, demonstrationID)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func completeDemonstration(service demonstrationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, demonstrationID, ok := demonstrationRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[demonstrationapp.Complete](w, r)
		if !ok {
			return
		}
		value, err := service.Complete(r.Context(), principal, demonstrationID, command)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func recordDemonstrationEvidenceView(
	service demonstrationapp.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, demonstrationID, ok := demonstrationRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[demonstrationapp.RecordEvidenceView](w, r)
		if !ok {
			return
		}
		value, err := service.RecordEvidenceView(
			r.Context(), principal, demonstrationID, command,
		)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func reviewDemonstration(service demonstrationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, demonstrationID, ok := demonstrationRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[demonstrationapp.Review](w, r)
		if !ok {
			return
		}
		value, err := service.Review(r.Context(), principal, demonstrationID, command)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func redactDemonstrationEvent(service demonstrationapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, demonstrationID, ok := demonstrationRequest(w, r)
		if !ok {
			return
		}
		eventID := chi.URLParam(r, "eventID")
		if !validUUIDParam(w, eventID, "event ID") {
			return
		}
		command, ok := decodeCommand[demonstrationapp.RedactEvent](w, r)
		if !ok {
			return
		}
		value, err := service.RedactEvent(
			r.Context(), principal, demonstrationID, eventID, command,
		)
		if err != nil {
			writeDemonstrationError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	}
}

func demonstrationRequest(
	w http.ResponseWriter,
	r *http.Request,
) (identitydomain.Principal, string, bool) {
	principal, ok := principalFromRequest(w, r)
	if !ok {
		return identitydomain.Principal{}, "", false
	}
	id := chi.URLParam(r, "demonstrationID")
	if !validUUIDParam(w, id, "demonstration ID") {
		return identitydomain.Principal{}, "", false
	}
	return principal, id, true
}

func writeDemonstrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, demonstrationapp.ErrCapturePending):
		writeProblem(
			w, http.StatusConflict, "Capture Pending",
			"semantic capture has pending or failed deliveries; retry after worker recovery",
		)
	default:
		writeDomainError(
			w, err, demonstrationapp.ErrForbidden, demonstrationapp.ErrNotFound,
			demonstrationapp.ErrInvalid, demonstrationapp.ErrConflict,
		)
	}
}
