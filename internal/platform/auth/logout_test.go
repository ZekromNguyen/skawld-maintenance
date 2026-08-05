package auth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
)

// fakeRepository is an in-memory sessionRepository the Logout handler test
// substitutes for the Postgres-backed Repository. Only the methods the handler
// exercises are non-trivial; the rest are stubs to satisfy the interface.
type fakeRepository struct {
	idToken     string
	sessionRaw  string
	token       string // the raw token the handler will send via cookie
	revokeErr   error
	revokeCalls int
	idTokenErr  error
}

func (f *fakeRepository) SaveFlow(context.Context, string, Flow) error { return nil }

func (f *fakeRepository) ConsumeFlow(context.Context, string) (Flow, error) {
	return Flow{}, nil
}

func (f *fakeRepository) ResolvePrincipal(
	context.Context,
	IdentityClaims,
	map[string]struct{},
) (identitydomain.Principal, error) {
	return identitydomain.Principal{}, nil
}

func (f *fakeRepository) CreateSession(
	_ context.Context, _ string, _ string, _ time.Time, _ string,
) error {
	return nil
}

func (f *fakeRepository) PrincipalForSession(
	context.Context, string, map[string]struct{},
) (identitydomain.Principal, error) {
	return identitydomain.Principal{}, nil
}

func (f *fakeRepository) SessionIDToken(context.Context, string) (string, error) {
	if f.idTokenErr != nil {
		return "", f.idTokenErr
	}
	return f.idToken, nil
}

func (f *fakeRepository) RevokeSession(context.Context, string) error {
	f.revokeCalls++
	return f.revokeErr
}

// newLogoutService builds a Service whose oidc verifiers are nil (Logout does
// not touch them) and whose endSessionURL/config are explicit so the redirect
// logic is exercised deterministically without a real identity provider.
func newLogoutService(t *testing.T, endSessionURL string, repo sessionRepository, cfg config.Auth) *Service {
	t.Helper()
	if cfg.CookieName == "" {
		cfg.CookieName = "skawld_session"
	}
	return &Service{
		config:            cfg,
		endSessionURL:     endSessionURL,
		repository:        repo,
		clock:             clock.System{},
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		bootstrapSubjects: map[string]struct{}{},
	}
}

func TestLogoutReturnsNoContentWhenNoSessionCookie(t *testing.T) {
	repo := &fakeRepository{}
	svc := newLogoutService(t, "https://idp.test/end", repo, config.Auth{
		RedirectURL: "http://localhost:5173/auth/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	svc.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if repo.revokeCalls != 0 {
		t.Fatalf("RevokeSession called %d times, want 0", repo.revokeCalls)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("Location = %q, want empty", loc)
	}
}

func TestLogoutClearsCookieAndReturnsNoContentWhenNoEndSessionEndpoint(t *testing.T) {
	repo := &fakeRepository{idToken: "id-token-X"}
	svc := newLogoutService(t, "", repo, config.Auth{
		RedirectURL: "http://localhost:5173/auth/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "skawld_session", Value: "the-session"})
	svc.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if repo.revokeCalls != 1 {
		t.Fatalf("RevokeSession called %d times, want 1", repo.revokeCalls)
	}
	assertCookieCleared(t, rec, "skawld_session")
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("Location = %q, want empty (no IdP logout)", loc)
	}
}

func TestLogoutReturnsNoContentWhenSessionHasNoIDToken(t *testing.T) {
	repo := &fakeRepository{idToken: ""}
	svc := newLogoutService(t, "https://idp.test/end", repo, config.Auth{
		RedirectURL: "http://localhost:5173/auth/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "skawld_session", Value: "the-session"})
	svc.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (no id_token → no IdP logout)", rec.Code, http.StatusNoContent)
	}
	if repo.revokeCalls != 1 {
		t.Fatalf("RevokeSession called %d times, want 1", repo.revokeCalls)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("Location = %q, want empty", loc)
	}
}

func TestLogoutRedirectsToIdPEndSessionWithHint(t *testing.T) {
	repo := &fakeRepository{idToken: "id-token-for-hint"}
	svc := newLogoutService(t, "https://idp.test/protocol/openid-connect/logout", repo, config.Auth{
		ClientID:    "skawld-web",
		RedirectURL: "http://localhost:5173/auth/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "skawld_session", Value: "the-session"})
	svc.Logout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (303 See Other)", rec.Code, http.StatusSeeOther)
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("Location header missing")
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if parsed.Host != "idp.test" {
		t.Fatalf("Location host = %q, want idp.test", parsed.Host)
	}
	if parsed.Path != "/protocol/openid-connect/logout" {
		t.Fatalf("Location path = %q, want /protocol/openid-connect/logout", parsed.Path)
	}
	q := parsed.Query()
	if got := q.Get("id_token_hint"); got != "id-token-for-hint" {
		t.Fatalf("id_token_hint = %q, want id-token-for-hint", got)
	}
	if got := q.Get("client_id"); got != "skawld-web" {
		t.Fatalf("client_id = %q, want skawld-web", got)
	}
	if got := q.Get("post_logout_redirect_uri"); got != "http://localhost:5173/" {
		t.Fatalf("post_logout_redirect_uri = %q, want http://localhost:5173/", got)
	}
	if repo.revokeCalls != 1 {
		t.Fatalf("RevokeSession called %d times, want 1", repo.revokeCalls)
	}
	assertCookieCleared(t, rec, "skawld_session")
}

func TestLogoutStillRedirectsWhenRevokeErrors(t *testing.T) {
	// Issue #2 regression: a DB failure during revocation must not strand the
	// user. The cookie is still cleared and the browser still follows the IdP
	// logout redirect; the failure is logged but not surfaced to the user.
	repo := &fakeRepository{
		idToken:   "id-token-for-hint",
		revokeErr: errDBTransient,
	}
	svc := newLogoutService(t, "https://idp.test/end", repo, config.Auth{
		ClientID:    "skawld-web",
		RedirectURL: "http://localhost:5173/auth/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "skawld_session", Value: "the-session"})
	svc.Logout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (must still redirect)", rec.Code, http.StatusSeeOther)
	}
	if repo.revokeCalls != 1 {
		t.Fatalf("RevokeSession called %d times, want 1", repo.revokeCalls)
	}
	assertCookieCleared(t, rec, "skawld_session")
}

func TestLogoutReturnsNoContentWhenRedirectURLIsRelative(t *testing.T) {
	// Defense-in-depth: if OIDC_REDIRECT_URL were ever set to a bare path the
	// handler must not synthesize a post_logout_redirect_uri it cannot vouch
	// for; it falls back to the local-session-only 204 path.
	repo := &fakeRepository{idToken: "id-token-for-hint"}
	svc := newLogoutService(t, "https://idp.test/end", repo, config.Auth{
		RedirectURL: "/auth/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "skawld_session", Value: "the-session"})
	svc.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("Location = %q, want empty (relative RedirectURL must not yield a redirect)", loc)
	}
}

// errDBTransient is a stand-in for a transient database error so the test
// above can assert the logout behavior without a real database connection.
var errDBTransient = &transientError{"transient database failure"}

type transientError struct{ msg string }

func (e *transientError) Error() string { return e.msg }

// assertCookieCleared verifies the handler set a cookie with MaxAge=-1 that
// overrides and expires any prior session cookie the browser is holding.
func assertCookieCleared(t *testing.T, rec *httptest.ResponseRecorder, name string) {
	t.Helper()
	setCookies := rec.Header().Values("Set-Cookie")
	for _, raw := range setCookies {
		if strings.Contains(raw, name+"=") && strings.Contains(raw, "Max-Age=0") {
			return
		}
		// Go's net/http emits "Max-Age=0" for cookies with MaxAge=-1 when
		// serialized via http.SetCookie. Some versions emit "expires=...1970".
		if strings.Contains(raw, name+"=") && strings.Contains(strings.ToLower(raw), "expires=thu, 01 jan 1970") {
			return
		}
	}
	t.Fatalf("expected Set-Cookie header to clear %q; got %v", name, setCookies)
}
