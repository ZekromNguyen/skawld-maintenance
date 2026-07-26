package httpserver

import (
	"net/http"

	assetapp "github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	"github.com/go-chi/chi/v5"
)

func mountAssetRoutes(router chi.Router, service assetapp.Service) {
	router.Get("/assets", listAssets(service))
	router.Post("/assets", createAsset(service))
	router.Get("/assets/{assetID}", getAsset(service))
	router.Post("/assets/{assetID}/criticality-approvals", approveAssetCriticality(service))
}

func listAssets(service assetapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		result, err := service.List(r.Context(), principal, assetapp.Filter{
			SiteID: r.URL.Query().Get("site_id"),
			Query:  r.URL.Query().Get("q"),
		})
		if err != nil {
			writeDomainError(w, err, assetapp.ErrForbidden, assetapp.ErrNotFound,
				assetapp.ErrInvalid, assetapp.ErrVersionConflict)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": result})
	}
}

func createAsset(service assetapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[assetapp.CreateAsset](w, r)
		if !ok {
			return
		}
		result, replay, err := service.Create(r.Context(), principal, idempotencyKey(r), command)
		if err != nil {
			writeDomainError(w, err, assetapp.ErrForbidden, assetapp.ErrNotFound,
				assetapp.ErrInvalid, assetapp.ErrVersionConflict)
			return
		}
		writeMutation(w, http.StatusCreated, result, replay)
	}
}

func getAsset(service assetapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "assetID")
		if !validUUIDParam(w, id, "asset ID") {
			return
		}
		result, err := service.Get(r.Context(), principal, id)
		if err != nil {
			writeDomainError(w, err, assetapp.ErrForbidden, assetapp.ErrNotFound,
				assetapp.ErrInvalid, assetapp.ErrVersionConflict)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func approveAssetCriticality(service assetapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "assetID")
		if !validUUIDParam(w, id, "asset ID") {
			return
		}
		command, ok := decodeCommand[assetapp.ApproveCriticality](w, r)
		if !ok {
			return
		}
		result, replay, err := service.ApproveCriticality(
			r.Context(), principal, idempotencyKey(r), id, command,
		)
		if err != nil {
			writeDomainError(w, err, assetapp.ErrForbidden, assetapp.ErrNotFound,
				assetapp.ErrInvalid, assetapp.ErrVersionConflict)
			return
		}
		writeMutation(w, http.StatusCreated, result, replay)
	}
}
