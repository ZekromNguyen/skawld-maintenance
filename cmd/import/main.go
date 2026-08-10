package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/ndjson"
	integrationpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/postgres"
	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type importArgs struct {
	Snapshot        string
	SiteID          string
	DatabaseURL     string
	Cursor          string
	Limit           int
	ExternalSystem  string
	ExternalSubject string
}

func main() {
	args := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := runImport(ctx, logger, args); err != nil {
		logger.Error("import failed", "error", err)
		os.Exit(1)
	}
}

func parseFlags() importArgs {
	args := importArgs{}
	flag.StringVar(&args.Snapshot, "snapshot", "", "NDJSON snapshot path (required)")
	flag.StringVar(&args.SiteID, "site-id", "", "target site UUID (required)")
	flag.StringVar(&args.DatabaseURL, "database-url", os.Getenv("DATABASE_URL"), "PostgreSQL URL")
	flag.StringVar(&args.Cursor, "cursor", "", "resume cursor from a previous run")
	flag.IntVar(&args.Limit, "limit", 100, "records per page (1-500)")
	flag.StringVar(&args.ExternalSystem, "external-system", "CMMS", "external system identity")
	flag.StringVar(&args.ExternalSubject, "external-subject", "import-admin", "administrator principal external subject")
	flag.Parse()
	return args
}

func runImport(ctx context.Context, logger *slog.Logger, args importArgs) error {
	if strings.TrimSpace(args.Snapshot) == "" || strings.TrimSpace(args.SiteID) == "" {
		return fmt.Errorf("-snapshot and -site-id are required")
	}
	if args.Limit < 1 || args.Limit > 500 {
		return fmt.Errorf("-limit must be between 1 and 500")
	}
	pool, err := database.Open(ctx, config.Database{
		URL: args.DatabaseURL, APIMaxConns: 5, WorkerMaxConns: 5,
		MaxBudget: 10, StatementTimeout: 15 * time.Second, LockTimeout: 3 * time.Second,
	}, config.RoleAPI)
	if err != nil {
		return err
	}
	defer pool.Close()
	principal, err := loadImportPrincipal(ctx, pool, args.ExternalSubject, args.SiteID)
	if err != nil {
		return err
	}
	snapshot, err := os.ReadFile(args.Snapshot)
	if err != nil {
		return err
	}
	rewritten := strings.ReplaceAll(string(snapshot), "REPLACED_BY_PRINCIPAL", principal.OrganizationID)
	rewritten = strings.ReplaceAll(rewritten, "REPLACED_BY_SITE", principal.SiteIDs[0])
	tempFile, err := os.CreateTemp("", "import-*.ndjson")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.WriteString(rewritten); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	connector, err := ndjson.New(tempFile.Name(), os.TempDir(), integrationdomain.ConnectorIdentity{
		System: args.ExternalSystem, Instance: "import-cli", Version: "1.0",
		ReadOnly: true, Capabilities: []integrationdomain.Capability{
			integrationdomain.CapabilityReadAssets,
		},
	})
	if err != nil {
		return err
	}
	importer := integrationapp.Importer{
		Connector: connector,
		Sink: integrationpostgres.Sink{
			Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
		},
	}
	cursor := args.Cursor
	total := 0
	for {
		page, err := importer.Pull(ctx, principal, cursor, args.Limit)
		if err != nil {
			return err
		}
		total += len(page.Records)
		logger.Info("import page", "records", len(page.Records), "next_cursor", page.NextCursor)
		if page.Complete {
			break
		}
		cursor = page.NextCursor
	}
	logger.Info("import complete", "total_records", total)
	return nil
}

// loadImportPrincipal resolves an administrator principal for the target site
// so the import runs under real tenant/site scope and RBAC.
func loadImportPrincipal(
	ctx context.Context,
	pool *pgxpool.Pool,
	subject string,
	siteID string,
) (identitydomain.Principal, error) {
	var principalID, organizationID string
	err := pool.QueryRow(ctx, `
		SELECT p.id::text, m.organization_id::text
		FROM principals p
		JOIN memberships m ON m.principal_id = p.id
		WHERE p.external_subject = $1 AND m.role = 'Administrator' AND m.site_id = $2::uuid
		LIMIT 1
	`, subject, siteID).Scan(&principalID, &organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.Principal{}, fmt.Errorf(
			"no administrator principal with subject %q in site %s", subject, siteID)
	}
	if err != nil {
		return identitydomain.Principal{}, err
	}
	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	return identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID,
		SiteIDs: []string{siteID}, Permissions: permissions,
	}, nil
}
