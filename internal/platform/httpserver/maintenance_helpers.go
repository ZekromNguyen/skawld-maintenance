package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/google/uuid"
)

func principalFromRequest(w http.ResponseWriter, r *http.Request) (domain.Principal, bool) {
	principal, ok := domain.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "principal is missing")
	}
	return principal, ok
}

func decodeCommand[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var command T
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&command); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid Request", "request body must match the schema")
		return command, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeProblem(w, http.StatusBadRequest, "Invalid Request", "request body must contain one JSON object")
		return command, false
	}
	return command, true
}

func idempotencyKey(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

func validUUIDParam(w http.ResponseWriter, value, label string) bool {
	if _, err := uuid.Parse(value); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid Request", label+" must be a UUID")
		return false
	}
	return true
}

func writeMutation[T any](w http.ResponseWriter, status int, value T, replay bool) {
	if replay {
		w.Header().Set("Idempotent-Replayed", "true")
		writeJSON(w, http.StatusOK, value)
		return
	}
	writeJSON(w, status, value)
}

func writeDomainError(
	w http.ResponseWriter,
	err error,
	forbidden, notFound, invalid, versionConflict error,
) {
	switch {
	case errors.Is(err, forbidden):
		writeProblem(w, http.StatusForbidden, "Forbidden", "permission denied")
	case errors.Is(err, notFound):
		writeProblem(w, http.StatusNotFound, "Not Found", "resource was not found")
	case errors.Is(err, invalid):
		writeProblem(w, http.StatusBadRequest, "Invalid Request", err.Error())
	case errors.Is(err, versionConflict), errors.Is(err, idempotency.ErrKeyConflict),
		errors.Is(err, idempotency.ErrInProgress):
		writeProblem(w, http.StatusConflict, "Conflict", err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "request could not be completed")
	}
}

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

func parsePageSize(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("page_size"))
	if raw == "" {
		return defaultPageSize, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 || value > maxPageSize {
		writeProblem(w, http.StatusBadRequest, "Invalid Request", "page_size must be an integer between 1 and 100")
		return 0, false
	}
	return value, true
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
