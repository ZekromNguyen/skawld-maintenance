package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type fakeAuth struct {
	principal domain.Principal
}

func (fakeAuth) Begin(w http.ResponseWriter, _ *http.Request)    { w.WriteHeader(http.StatusFound) }
func (fakeAuth) Callback(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusSeeOther) }
func (fakeAuth) Logout(w http.ResponseWriter, _ *http.Request)   { w.WriteHeader(http.StatusNoContent) }
func (a fakeAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := a.principal
		if principal.ID == "" {
			principal = domain.Principal{
				ID:              "00000000-0000-0000-0000-000000000001",
				ExternalSubject: "test",
				DisplayName:     "Test User",
				Permissions:     map[domain.Permission]struct{}{},
			}
		}
		next.ServeHTTP(w, r.WithContext(domain.WithPrincipal(r.Context(), principal)))
	})
}

type fakeSiteReader struct {
	site application.Site
}

func (r fakeSiteReader) Get(
	_ context.Context,
	principal domain.Principal,
	siteID string,
) (application.Site, error) {
	if siteID != r.site.ID || !principal.CanAccessSite(r.site.OrganizationID, r.site.ID) {
		return application.Site{}, application.ErrSiteNotFound
	}
	return r.site, nil
}

func TestLiveness(t *testing.T) {
	t.Parallel()
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{},
	})
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestCurrentPrincipalIsProtectedAndMapped(t *testing.T) {
	t.Parallel()
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestSiteEndpointRefusesWrongSite(t *testing.T) {
	t.Parallel()
	allowedSiteID := "00000000-0000-4000-8000-000000000010"
	wrongSiteID := "00000000-0000-4000-8000-000000000011"
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth: fakeAuth{principal: domain.Principal{
			ID:             "00000000-0000-4000-8000-000000000001",
			OrganizationID: "00000000-0000-4000-8000-000000000100",
			SiteIDs:        []string{allowedSiteID},
			Permissions:    map[domain.Permission]struct{}{},
		}},
		Sites: application.SiteService{Reader: fakeSiteReader{site: application.Site{
			ID:             wrongSiteID,
			OrganizationID: "00000000-0000-4000-8000-000000000100",
		}}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+wrongSiteID, nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
