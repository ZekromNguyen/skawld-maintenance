package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

// rejectingAuth never injects a principal, exercising the 401 path.
type rejectingAuth struct{ fakeAuth }

func (rejectingAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "principal is missing")
	})
}

func TestSummaryRequiresAuthentication(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   rejectingAuth{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSummaryRequiresIncidentReadPermission(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs:     []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
