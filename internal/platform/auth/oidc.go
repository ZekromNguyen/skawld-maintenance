package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Service struct {
	config            config.Auth
	oauth             oauth2.Config
	webVerifier       *oidc.IDTokenVerifier
	bearerVerifier    *oidc.IDTokenVerifier
	repository        Repository
	clock             clock.Clock
	logger            *slog.Logger
	bootstrapSubjects map[string]struct{}
}

func New(
	ctx context.Context,
	cfg config.Auth,
	repository Repository,
	systemClock clock.Clock,
	logger *slog.Logger,
) (*Service, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	audience := cfg.Audience
	if audience == "" {
		audience = cfg.ClientID
	}
	bootstrap := make(map[string]struct{}, len(cfg.BootstrapSubjects))
	for _, subject := range cfg.BootstrapSubjects {
		bootstrap[subject] = struct{}{}
	}
	return &Service{
		config: cfg,
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		webVerifier:       provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		bearerVerifier:    provider.Verifier(&oidc.Config{ClientID: audience}),
		repository:        repository,
		clock:             systemClock,
		logger:            logger,
		bootstrapSubjects: bootstrap,
	}, nil
}

func (s *Service) Begin(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "cannot start authentication", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "cannot start authentication", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	returnTo := safeReturnTo(r.URL.Query().Get("return_to"))
	if err := s.repository.SaveFlow(r.Context(), state, Flow{
		Nonce:        nonce,
		PKCEVerifier: verifier,
		ReturnTo:     returnTo,
		ExpiresAt:    s.clock.Now().Add(10 * time.Minute),
	}); err != nil {
		s.logger.Error("save OIDC flow", "error", err)
		http.Error(w, "cannot start authentication", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.CookieName + "_oidc_state",
		Value:    state,
		Path:     "/auth/callback",
		HttpOnly: true,
		Secure:   s.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	target := s.oauth.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		http.Error(w, "identity provider rejected authentication", http.StatusUnauthorized)
		return
	}
	state := r.URL.Query().Get("state")
	stateCookie, cookieErr := r.Cookie(s.config.CookieName + "_oidc_state")
	if cookieErr != nil || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(state)) != 1 {
		http.Error(w, "invalid authentication browser state", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.CookieName + "_oidc_state",
		Value:    "",
		Path:     "/auth/callback",
		HttpOnly: true,
		Secure:   s.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	flow, err := s.repository.ConsumeFlow(r.Context(), state)
	if err != nil {
		http.Error(w, "invalid or expired authentication flow", http.StatusUnauthorized)
		return
	}
	token, err := s.oauth.Exchange(
		r.Context(),
		r.URL.Query().Get("code"),
		oauth2.VerifierOption(flow.PKCEVerifier),
	)
	if err != nil {
		s.logger.Warn("OIDC code exchange failed", "error", err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "identity token missing", http.StatusUnauthorized)
		return
	}
	idToken, err := s.webVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "identity token invalid", http.StatusUnauthorized)
		return
	}
	claims, err := claimsFromToken(idToken)
	if err != nil || claims.Nonce != flow.Nonce {
		http.Error(w, "identity token claims invalid", http.StatusUnauthorized)
		return
	}
	principal, err := s.repository.ResolvePrincipal(r.Context(), claims.IdentityClaims, s.bootstrapSubjects)
	if err != nil {
		s.logger.Error("resolve OIDC principal", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	sessionToken, err := randomToken(32)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	expiresAt := s.clock.Now().Add(s.config.SessionTTL)
	if err := s.repository.CreateSession(r.Context(), sessionToken, principal.ID, expiresAt); err != nil {
		s.logger.Error("create web session", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.CookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
	http.Redirect(w, r, flow.ReturnTo, http.StatusSeeOther)
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(s.config.CookieName); err == nil {
		_ = s.repository.RevokeSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := s.authenticate(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401}`))
			return
		}
		next.ServeHTTP(w, r.WithContext(domain.WithPrincipal(r.Context(), principal)))
	})
}

func (s *Service) authenticate(r *http.Request) (domain.Principal, error) {
	if cookie, err := r.Cookie(s.config.CookieName); err == nil && cookie.Value != "" {
		return s.repository.PrincipalForSession(r.Context(), cookie.Value, s.bootstrapSubjects)
	}
	const bearerPrefix = "Bearer "
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, bearerPrefix) {
		return domain.Principal{}, fmt.Errorf("authentication required")
	}
	rawToken := strings.TrimSpace(strings.TrimPrefix(authorization, bearerPrefix))
	token, err := s.bearerVerifier.Verify(r.Context(), rawToken)
	if err != nil {
		return domain.Principal{}, fmt.Errorf("verify bearer token: %w", err)
	}
	claims, err := claimsFromToken(token)
	if err != nil {
		return domain.Principal{}, err
	}
	return s.repository.ResolvePrincipal(r.Context(), claims.IdentityClaims, s.bootstrapSubjects)
}

type tokenClaims struct {
	IdentityClaims
	Nonce string `json:"nonce"`
}

func claimsFromToken(token *oidc.IDToken) (tokenClaims, error) {
	var raw struct {
		Subject           string `json:"sub"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Nonce             string `json:"nonce"`
	}
	if err := token.Claims(&raw); err != nil {
		return tokenClaims{}, fmt.Errorf("decode identity claims: %w", err)
	}
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = strings.TrimSpace(raw.PreferredUsername)
	}
	if raw.Subject == "" || name == "" {
		return tokenClaims{}, fmt.Errorf("required identity claims missing")
	}
	return tokenClaims{
		IdentityClaims: IdentityClaims{
			Subject:     raw.Subject,
			DisplayName: name,
			Email:       raw.Email,
		},
		Nonce: raw.Nonce,
	}, nil
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func safeReturnTo(raw string) string {
	if raw == "" {
		return "/"
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") ||
		strings.HasPrefix(parsed.Path, "//") {
		return "/"
	}
	return parsed.RequestURI()
}
