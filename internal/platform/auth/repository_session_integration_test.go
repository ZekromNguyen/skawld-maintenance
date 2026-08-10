package auth

import (
	"context"
	"crypto/sha256"
	"os"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestRepository returns a Repository bound to the test database. The test
// is skipped when TEST_DATABASE_URL is unset so the suite remains runnable in
// environments without Postgres (the unit tests in oidc_logout_test.go then
// carry the coverage for the OIDC service handler logic).
func newTestRepository(t *testing.T, now time.Time) (Repository, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return Repository{Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now}}, pool
}

// seedTestPrincipal inserts a principal row the caller can attach a session to.
// Returns the principal id. Deletes the row at test end so the suite is
// re-runnable against a shared database.
func seedTestPrincipal(t *testing.T, pool *pgxpool.Pool, now time.Time) string {
	t.Helper()
	ctx := context.Background()
	subject := "session-test-" + uuid.NewString()
	principalID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, email, status, created_at, updated_at
		) VALUES ($1::uuid, $2, $3, $4, 'ACTIVE', $5, $5)
	`, principalID, subject, "Session Test", "session.test@example.invalid", now); err != nil {
		t.Fatalf("seed principal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE id = $1::uuid`, principalID)
	})
	return principalID
}

// createSessionWithClock is a small wrapper around CreateSession that materializes
// the Repository's clock-derived timestamps so a fixed-clock round-trip and an
// expiry that is already in the past can be expressed by the same code path.
func createSessionWithClock(t *testing.T, repo Repository, token, principalID string, expiresAt time.Time, idToken string) {
	t.Helper()
	if err := repo.CreateSession(context.Background(), token, principalID, expiresAt, idToken); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

// deleteSession deletes a session row by computing the same SHA-256 token hash
// the repository uses for storage. pgcrypto is not present in this project, so
// we compute the hash in Go rather than relying on a SQL digest function.
func deleteSession(pool *pgxpool.Pool, token string) {
	hash := sha256.Sum256([]byte(token))
	_, _ = pool.Exec(context.Background(),
		`DELETE FROM web_sessions WHERE token_hash = $1`, hash[:])
}

func TestCreateSessionPersistsIDTokenRoundTrippedBySessionIDToken(t *testing.T) {
	now := time.Now().UTC()
	repo, pool := newTestRepository(t, now)
	principalID := seedTestPrincipal(t, pool, now)
	token := "roundtrip-" + uuid.NewString()
	createSessionWithClock(t, repo, token, principalID, now.Add(8*time.Hour), "id-token-A")
	t.Cleanup(func() { deleteSession(pool, token) })

	got, err := repo.SessionIDToken(context.Background(), token)
	if err != nil {
		t.Fatalf("SessionIDToken: %v", err)
	}
	if got != "id-token-A" {
		t.Fatalf("id_token = %q, want %q", got, "id-token-A")
	}
}

func TestCreateSessionAcceptsEmptyIDToken(t *testing.T) {
	now := time.Now().UTC()
	repo, pool := newTestRepository(t, now)
	principalID := seedTestPrincipal(t, pool, now)
	token := "empty-" + uuid.NewString()
	createSessionWithClock(t, repo, token, principalID, now.Add(8*time.Hour), "")
	t.Cleanup(func() { deleteSession(pool, token) })

	got, err := repo.SessionIDToken(context.Background(), token)
	if err != nil {
		t.Fatalf("SessionIDToken: %v", err)
	}
	if got != "" {
		t.Fatalf("id_token = %q, want empty (NULL coalesced to '')", got)
	}
}

func TestSessionIDTokenReturnsErrorAfterRevoke(t *testing.T) {
	now := time.Now().UTC()
	repo, pool := newTestRepository(t, now)
	principalID := seedTestPrincipal(t, pool, now)
	token := "revoked-" + uuid.NewString()
	createSessionWithClock(t, repo, token, principalID, now.Add(8*time.Hour), "id-token-B")
	t.Cleanup(func() { deleteSession(pool, token) })

	if err := repo.RevokeSession(context.Background(), token); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	if _, err := repo.SessionIDToken(context.Background(), token); err == nil {
		t.Fatal("SessionIDToken after revoke returned nil error; expected rows.Err or no-rows")
	}
}

func TestSessionIDTokenReturnsErrorAfterExpiry(t *testing.T) {
	// Use a Fixed clock frozen in the past so the session is already expired
	// at lookup time (its expires_at is before the repository's now).
	frozenNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	repo := Repository{Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: frozenNow}}
	principalID := seedTestPrincipal(t, pool, frozenNow)
	token := "expired-" + uuid.NewString()
	// Session was created in the past and expires one second before lookup, so
	// the `expires_at > now` predicate excludes it.
	createSessionWithClock(t, repo, token, principalID, frozenNow.Add(-1*time.Second), "id-token-C")
	t.Cleanup(func() { deleteSession(pool, token) })

	if _, err := repo.SessionIDToken(ctx, token); err == nil {
		t.Fatal("SessionIDToken on expired session returned nil error; expected expiry predicate to exclude it")
	}
}

func TestSessionIDTokenUnknownTokenReturnsError(t *testing.T) {
	now := time.Now().UTC()
	repo, _ := newTestRepository(t, now)
	if _, err := repo.SessionIDToken(context.Background(), "not-a-real-session-token"); err == nil {
		t.Fatal("SessionIDToken on unknown token returned nil error; expected pgx.ErrNoRows")
	}
}

func TestRevokeSessionIsIdempotent(t *testing.T) {
	now := time.Now().UTC()
	repo, pool := newTestRepository(t, now)
	principalID := seedTestPrincipal(t, pool, now)
	token := "idempotent-" + uuid.NewString()
	createSessionWithClock(t, repo, token, principalID, now.Add(8*time.Hour), "id-token-D")
	t.Cleanup(func() { deleteSession(pool, token) })

	if err := repo.RevokeSession(context.Background(), token); err != nil {
		t.Fatalf("first RevokeSession: %v", err)
	}
	if err := repo.RevokeSession(context.Background(), token); err != nil {
		t.Fatalf("second (no-op) RevokeSession: %v", err)
	}
	if _, err := repo.SessionIDToken(context.Background(), token); err == nil {
		t.Fatal("SessionIDToken after revoke returned nil; expected revoked predicate to exclude it")
	}
}
