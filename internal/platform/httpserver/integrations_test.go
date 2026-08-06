package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
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
