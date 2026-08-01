package httpserver

import (
	"errors"
	"net/http"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/go-chi/chi/v5"
)

func mountKnowledgeRoutes(router chi.Router, service knowledgeapp.Service) {
	router.Get("/documents", listDocuments(service))
	router.Post("/documents", createDocument(service))
	router.Get("/documents/{documentID}", getDocument(service))
	router.Post("/documents/{documentID}/revisions", createDocumentRevision(service))
	router.Post("/document-revisions/{revisionID}/ingestion", requestDocumentIngestion(service))
	router.Post("/document-revisions/{revisionID}/approve", approveDocumentRevision(service))
	router.Post("/document-revisions/{revisionID}/retire", retireDocumentRevision(service))
	router.Post("/search", searchKnowledge(service))
}

func listDocuments(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		items, err := service.ListDocuments(r.Context(), principal, knowledgeapp.Filter{
			SiteID: r.URL.Query().Get("site_id"),
		})
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createDocument(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[knowledgeapp.CreateDocument](w, r)
		if !ok {
			return
		}
		value, replay, err := service.CreateDocument(
			r.Context(), principal, idempotencyKey(r), command,
		)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, value, replay)
	}
}

func getDocument(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		documentID := chi.URLParam(r, "documentID")
		if !validUUIDParam(w, documentID, "document ID") {
			return
		}
		value, err := service.GetDocument(r.Context(), principal, documentID)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func createDocumentRevision(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		documentID := chi.URLParam(r, "documentID")
		if !validUUIDParam(w, documentID, "document ID") {
			return
		}
		command, ok := decodeCommand[knowledgeapp.CreateRevision](w, r)
		if !ok {
			return
		}
		value, replay, err := service.CreateRevision(
			r.Context(), principal, idempotencyKey(r), documentID, command,
		)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeMutation(w, http.StatusCreated, value, replay)
	}
}

func requestDocumentIngestion(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		revisionID, principal, ok := revisionRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[knowledgeapp.RequestIngestion](w, r)
		if !ok {
			return
		}
		value, replay, err := service.RequestIngestion(
			r.Context(), principal, idempotencyKey(r), revisionID, command,
		)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeMutation(w, http.StatusAccepted, value, replay)
	}
}

func approveDocumentRevision(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		revisionID, principal, ok := revisionRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[knowledgeapp.ApproveRevision](w, r)
		if !ok {
			return
		}
		value, replay, err := service.ApproveRevision(
			r.Context(), principal, idempotencyKey(r), revisionID, command,
		)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, value, replay)
	}
}

func retireDocumentRevision(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		revisionID, principal, ok := revisionRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[knowledgeapp.RetireRevision](w, r)
		if !ok {
			return
		}
		value, replay, err := service.RetireRevision(
			r.Context(), principal, idempotencyKey(r), revisionID, command,
		)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeMutation(w, http.StatusOK, value, replay)
	}
}

func searchKnowledge(service knowledgeapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		query, ok := decodeCommand[knowledgedomain.SearchQuery](w, r)
		if !ok {
			return
		}
		value, err := service.Search(r.Context(), principal, query)
		if err != nil {
			writeKnowledgeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func revisionRequest(w http.ResponseWriter, r *http.Request) (string, identitydomain.Principal, bool) {
	principal, ok := principalFromRequest(w, r)
	if !ok {
		return "", identitydomain.Principal{}, false
	}
	revisionID := chi.URLParam(r, "revisionID")
	if !validUUIDParam(w, revisionID, "revision ID") {
		return "", identitydomain.Principal{}, false
	}
	return revisionID, principal, true
}

func writeKnowledgeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, knowledgeapp.ErrNoAttachment):
		writeProblem(w, http.StatusUnprocessableEntity, "Attachment Required", err.Error())
	default:
		writeDomainError(w, err, knowledgeapp.ErrForbidden, knowledgeapp.ErrNotFound,
			knowledgeapp.ErrInvalid, knowledgeapp.ErrConflict)
	}
}
