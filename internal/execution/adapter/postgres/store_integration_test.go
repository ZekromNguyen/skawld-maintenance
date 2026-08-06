package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	assetpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/asset/adapter/postgres"
	assetapp "github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	executionpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/execution/adapter/postgres"
	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	incidentpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/incident/adapter/postgres"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPumpExecutionPersistsEvidenceAndBlocksIntrusiveStep(t *testing.T) {
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
	generator := id.UUID{}
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Integration Organization', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'TEST', 'Integration Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Integration Technician', 'ACTIVE', $2, $2)
	`, principalID, now)
	if err != nil {
		t.Fatal(err)
	}

	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetCreate:        {},
			identitydomain.PermissionAssetRead:          {},
			identitydomain.PermissionIncidentCreate:     {},
			identitydomain.PermissionIncidentRead:       {},
			identitydomain.PermissionExecutionWrite:     {},
			identitydomain.PermissionExecutionRead:      {},
			identitydomain.PermissionPrerequisiteVerify: {},
		},
	}
	commonClock := clock.Fixed{Time: now}
	assetStore := assetpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	incidentStore := incidentpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	executionStore := executionpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}

	asset, _, err := assetStore.Create(ctx, principal, uuid.NewString(), assetapp.CreateAsset{
		SiteID: siteID, Tag: "P-302", Name: "Process Pump P-302",
		Class: "CENTRIFUGAL_PUMP", SourceOfTruth: "OWNED_BY_SKAWLD",
		Components: []assetapp.ComponentInput{
			{Code: "BRG-M", Name: "Motor-side bearing", Type: "BEARING"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	incident, _, err := incidentStore.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
		SiteID: siteID, AssetID: asset.ID, Summary: "High vibration",
		Severity: "HIGH", SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, _, err := executionStore.Create(ctx, principal, uuid.NewString(), executionapp.CreateExecution{
		IncidentID: incident.ID, Purpose: "Inspect high vibration",
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, _, err = executionStore.Start(ctx, principal, uuid.NewString(), execution.ID, executionapp.StartExecution{
		ExpectedVersion: execution.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	measurement, _, err := executionStore.RecordMeasurement(
		ctx, principal, uuid.NewString(), execution.ID,
		executionapp.RecordMeasurement{
			ClientEventID: uuid.NewString(), ComponentID: asset.Components[0].ID,
			MeasurementType: "VIBRATION_VELOCITY", Value: "8.1", Unit: "MM_PER_S",
			Source: "MANUAL", DataQuality: "GOOD", VerificationStatus: "UNVERIFIED",
			ObservedAt: now,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Value != "8.1" || measurement.Unit != "MM_PER_S" {
		t.Fatalf("measurement = %s %s", measurement.Value, measurement.Unit)
	}

	for index := 0; index < 3; index++ {
		step := execution.Steps[index]
		_, _, err = executionStore.CompleteStep(
			ctx, principal, uuid.NewString(), execution.ID, step.ID,
			executionapp.CompleteStep{
				ExpectedExecutionVersion: execution.Version,
				ExpectedStepVersion:      step.Version,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		execution, err = executionStore.Get(ctx, principal, execution.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	intrusive := execution.Steps[3]
	blocked, _, err := executionStore.CompleteStep(
		ctx, principal, uuid.NewString(), execution.ID, intrusive.ID,
		executionapp.CompleteStep{
			ExpectedExecutionVersion: execution.Version,
			ExpectedStepVersion:      intrusive.Version,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.State != "BLOCKED" {
		t.Fatalf("intrusive step state = %s, want BLOCKED", blocked.State)
	}
	_, _, err = executionStore.VerifyPrerequisite(
		ctx, principal, uuid.NewString(), execution.ID,
		executionapp.VerifyPrerequisite{
			Type: "ENERGY_ISOLATION", Status: "VERIFIED",
			ExternalReference: "LOTO-TEST-001", VerifiedAt: now,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	execution, err = executionStore.Get(ctx, principal, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = executionStore.CompleteStep(
		ctx, principal, uuid.NewString(), execution.ID, intrusive.ID,
		executionapp.CompleteStep{
			ExpectedExecutionVersion: execution.Version,
			ExpectedStepVersion:      intrusive.Version,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

// TestOrgScopedPrincipalWithNilSiteIDsSeesExecution is a regression test for
// tenant queries treating a nil SiteIDs slice as "no site restriction". An
// org-level principal (for example the bootstrap administrator created with an
// organization) has SiteIDs == nil, which pgx encodes as a NULL array. Without
// NULL-safe cardinality checks every scoped query silently returned no rows or
// Create failed with "no rows in result set".
func TestOrgScopedPrincipalWithNilSiteIDsSeesExecution(t *testing.T) {
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
	generator := id.UUID{}
	organizationID, siteID, technicianID, orgAdminID :=
		uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Integration Organization', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'TEST', 'Integration Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, principalID := range []string{technicianID, orgAdminID} {
		_, err = pool.Exec(ctx, `
			INSERT INTO principals (
				id, external_subject, display_name, status, created_at, updated_at
			) VALUES ($1::uuid, $1, 'Integration Technician', 'ACTIVE', $2, $2)
		`, principalID, now)
		if err != nil {
			t.Fatal(err)
		}
	}

	technician := identitydomain.Principal{
		ID: technicianID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetCreate:    {},
			identitydomain.PermissionIncidentCreate: {},
			identitydomain.PermissionExecutionWrite: {},
			identitydomain.PermissionExecutionRead:  {},
		},
	}
	// The org-scoped principal has no site memberships; SiteIDs must stay nil to
	// reproduce the NULL-array encoding regression.
	orgAdmin := identitydomain.Principal{
		ID: orgAdminID, OrganizationID: organizationID, SiteIDs: nil,
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExecutionRead: {},
		},
	}
	commonClock := clock.Fixed{Time: now}
	assetStore := assetpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	incidentStore := incidentpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	executionStore := executionpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}

	asset, _, err := assetStore.Create(ctx, technician, uuid.NewString(), assetapp.CreateAsset{
		SiteID: siteID, Tag: "P-302", Name: "Process Pump P-302",
		Class: "CENTRIFUGAL_PUMP", SourceOfTruth: "OWNED_BY_SKAWLD",
	})
	if err != nil {
		t.Fatal(err)
	}
	incident, _, err := incidentStore.Create(ctx, technician, uuid.NewString(), incidentapp.CreateIncident{
		SiteID: siteID, AssetID: asset.ID, Summary: "High vibration",
		Severity: "HIGH", SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := executionStore.Create(ctx, technician, uuid.NewString(), executionapp.CreateExecution{
		IncidentID: incident.ID, Purpose: "Inspect high vibration",
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := executionStore.List(ctx, orgAdmin, executionapp.Filter{PageSize: 25})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("org-scoped principal listed %d executions, want 1", len(items))
	}
	if _, err := executionStore.Get(ctx, orgAdmin, items[0].ID); err != nil {
		t.Fatalf("org-scoped Get failed: %v", err)
	}
}

func TestExecutionListCursorPagination(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	now := time.Now().UTC().Truncate(time.Second)
	generator := id.UUID{}
	organizationID, siteID, technicianID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Execution Pagination', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'EPG', 'Execution Pagination Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Execution Pagination Technician', 'ACTIVE', $2, $2)
	`, technicianID, now); err != nil {
		t.Fatal(err)
	}

	technician := identitydomain.Principal{
		ID: technicianID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetCreate:    {},
			identitydomain.PermissionIncidentCreate: {},
			identitydomain.PermissionExecutionWrite: {},
			identitydomain.PermissionExecutionRead:  {},
		},
	}
	commonClock := clock.Fixed{Time: now}
	assetStore := assetpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	incidentStore := incidentpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	executionStore := executionpostgres.Store{
		Pool: pool, IDs: generator, Clock: commonClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	asset, _, err := assetStore.Create(ctx, technician, uuid.NewString(), assetapp.CreateAsset{
		SiteID: siteID, Tag: "P-302", Name: "Process Pump P-302",
		Class: "CENTRIFUGAL_PUMP", SourceOfTruth: "OWNED_BY_SKAWLD",
	})
	if err != nil {
		t.Fatal(err)
	}
	incident, _, err := incidentStore.Create(ctx, technician, uuid.NewString(), incidentapp.CreateIncident{
		SiteID: siteID, AssetID: asset.ID, Summary: "High vibration",
		Severity: "HIGH", SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 3; index++ {
		if _, _, err := executionStore.Create(ctx, technician, uuid.NewString(), executionapp.CreateExecution{
			IncidentID: incident.ID, Purpose: "Inspect high vibration " + string(rune('0'+index)),
		}); err != nil {
			t.Fatal(err)
		}
	}

	page1, hasMore, err := executionStore.List(ctx, technician, executionapp.Filter{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || !hasMore {
		t.Fatalf("page 1 = %d items, hasMore %v; want 2, true", len(page1), hasMore)
	}
	last := page1[len(page1)-1]
	cursor := last.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
	page2, hasMore2, err := executionStore.List(ctx, technician, executionapp.Filter{PageSize: 2, Cursor: cursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 1 || hasMore2 {
		t.Fatalf("page 2 = %d items, hasMore %v; want 1, false", len(page2), hasMore2)
	}
	seen := map[string]bool{}
	for _, item := range append(page1, page2...) {
		seen[item.ID] = true
	}
	if len(seen) != 3 {
		t.Fatalf("distinct executions across pages = %d, want 3", len(seen))
	}
}
