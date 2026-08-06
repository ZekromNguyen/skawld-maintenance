package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/postgres"
	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type recordSink struct {
	applied int
}

func (s *recordSink) Apply(
	_ context.Context,
	_ identitydomain.Principal,
	records []integrationdomain.ExternalRecord,
) error {
	s.applied += len(records)
	return nil
}

func integrationImportHandler(sink integrationapp.ProjectionSink, principal identitydomain.Principal) http.Handler {
	return New(Dependencies{
		Logger:          slog.New(slog.DiscardHandler),
		Auth:            fakeAuth{principal: principal},
		IntegrationSink: sink,
	})
}

func TestIntegrationImportEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExternalImport: {},
		},
	}
	sink := &recordSink{}
	handler := integrationImportHandler(sink, principal)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(
		`{"kind":"ASSET","organization_id":"o1","site_id":"11111111-1111-4111-8111-111111111111","external_system":"CMMS-X","external_id":"a-1","external_version":"v1","observed_at":"2026-08-01T09:00:00Z","attributes":{"tag":"P-302","name":"Pump P-302","asset_class":"PUMP"}}` + "\n",
	)); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("system", "CMMS-X"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("instance", "import-test"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("version", "1.0"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if sink.applied == 0 {
		t.Fatal("sink was not applied")
	}
	var envelope struct {
		Imported   int    `json:"imported"`
		Complete   bool   `json:"complete"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Imported != 1 || !envelope.Complete {
		t.Fatalf("envelope = %+v, want imported 1 complete true", envelope)
	}
}

func TestIntegrationImportForbidden(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetRead: {},
		},
	}
	handler := integrationImportHandler(&recordSink{}, principal)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("{}\n"))
	writer.WriteField("system", "CMMS-X")
	writer.WriteField("instance", "import-test")
	writer.WriteField("version", "1.0")
	writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestIntegrationImportRejectsOversizedBody(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExternalImport: {},
		},
	}
	handler := integrationImportHandler(&recordSink{}, principal)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	part.Write(make([]byte, maxImportBytes+1))
	writer.WriteField("system", "CMMS-X")
	writer.WriteField("instance", "import-test")
	writer.WriteField("version", "1.0")
	writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestIntegrationImportRejectsMissingIdentityFields(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExternalImport: {},
		},
	}
	handler := integrationImportHandler(&recordSink{}, principal)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	part.Write([]byte("{}\n"))
	writer.WriteField("system", "CMMS-X")
	writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestIntegrationImportRejectsMalformedSnapshot(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExternalImport: {},
		},
	}
	handler := integrationImportHandler(&recordSink{}, principal)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	part.Write([]byte("{not-json}\n"))
	writer.WriteField("system", "CMMS-X")
	writer.WriteField("instance", "import-test")
	writer.WriteField("version", "1.0")
	writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestIntegrationImportRejectsMissingAttribute(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Import Handler Organization', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'HANDLER', 'Import Handler Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Import Handler Admin', 'ACTIVE', $2, $2)
	`, principalID, now); err != nil {
		t.Fatal(err)
	}
	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID,
		SiteIDs: []string{siteID}, Permissions: permissions,
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		IntegrationSink: integrationpostgres.Sink{
			Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
		},
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	part.Write([]byte(
		`{"kind":"ASSET","organization_id":"` + organizationID + `","site_id":"` + siteID + `","external_system":"CMMS-X","external_id":"a-9","external_version":"v1","observed_at":"2026-08-01T09:00:00Z","attributes":{"tag":"P-302","asset_class":"PUMP"}}` + "\n",
	))
	writer.WriteField("system", "CMMS-X")
	writer.WriteField("instance", "import-test")
	writer.WriteField("version", "1.0")
	writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (missing required attribute)", recorder.Code)
	}
}
