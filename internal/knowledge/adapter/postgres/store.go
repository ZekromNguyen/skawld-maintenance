package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IngestionEnqueuer interface {
	EnqueueDocumentIngestionTx(context.Context, pgx.Tx, string, string) error
}

type Store struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Idempotency idempotency.Store
	Audit       audit.Sink
	Enqueuer    IngestionEnqueuer
	Embeddings  skawld.EmbeddingProvider
}

type documentResult struct {
	Value  knowledgedomain.Document
	Replay bool
}

type revisionResult struct {
	Value  knowledgedomain.Revision
	Replay bool
}

func (s Store) CreateDocument(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command knowledgeapp.CreateDocument,
) (knowledgedomain.Document, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return knowledgedomain.Document{}, false, err
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (documentResult, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(
			ctx, tx, principal.ID, "knowledge.document.create.v1", key, hash, now,
		)
		if err != nil {
			return documentResult{}, err
		}
		if record.Replay {
			var replay knowledgedomain.Document
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return documentResult{}, err
			}
			return documentResult{Value: replay, Replay: true}, nil
		}
		value := knowledgedomain.Document{
			ID: s.IDs.New(), OrganizationID: principal.OrganizationID,
			SiteID: command.SiteID, Type: command.DocumentType, Title: command.Title,
			Authority: command.Authority, SourceReference: command.SourceReference,
			Revisions: []knowledgedomain.Revision{}, CreatedAt: now, UpdatedAt: now,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO documents (
				id, organization_id, site_id, document_type, title, authority,
				source_reference, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, nullif($7, ''),
				$8::uuid, $9, $9
			)
		`, value.ID, value.OrganizationID, value.SiteID, value.Type, value.Title,
			value.Authority, value.SourceReference, principal.ID, now)
		if err != nil {
			return documentResult{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			ActorID: principal.ID, Action: "knowledge.document.created",
			EntityKind: "document", EntityID: value.ID, After: value, OccurredAt: now,
		}); err != nil {
			return documentResult{}, err
		}
		body, _ := json.Marshal(value)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, "knowledge.document.create.v1", key,
			http.StatusCreated, body, now,
		); err != nil {
			return documentResult{}, err
		}
		return documentResult{Value: value}, nil
	})
	if err != nil {
		return knowledgedomain.Document{}, false, err
	}
	return result.Value, result.Replay, nil
}

func (s Store) CreateRevision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, documentID string,
	command knowledgeapp.CreateRevision,
) (knowledgedomain.Revision, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	scope := "knowledge.revision.create.v1:" + documentID
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (revisionResult, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return revisionResult{}, err
		}
		if record.Replay {
			var replay knowledgedomain.Revision
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return revisionResult{}, err
			}
			return revisionResult{Value: replay, Replay: true}, nil
		}
		var siteID string
		err = tx.QueryRow(ctx, `
			SELECT site_id::text FROM documents
			WHERE id = $1::uuid AND organization_id = $2::uuid
			FOR UPDATE
		`, documentID, principal.OrganizationID).Scan(&siteID)
		if errors.Is(err, pgx.ErrNoRows) {
			return revisionResult{}, knowledgeapp.ErrNotFound
		}
		if err != nil {
			return revisionResult{}, err
		}
		if !principal.CanAccessSite(principal.OrganizationID, siteID) {
			return revisionResult{}, knowledgeapp.ErrForbidden
		}
		if err := validateApplicability(ctx, tx, principal, siteID, command.Applicability); err != nil {
			return revisionResult{}, err
		}
		value := knowledgedomain.Revision{
			ID: s.IDs.New(), DocumentID: documentID, Revision: command.Revision,
			ApprovalStatus: knowledgedomain.ApprovalDraft,
			EffectiveAt:    command.EffectiveAt, ExpiresAt: command.ExpiresAt,
			IngestionState: knowledgedomain.IngestionAwaitingUpload,
			Language:       command.Language, Version: 1,
			Applicability: command.Applicability,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO document_revisions (
				id, organization_id, document_id, revision, effective_at,
				expires_at, language, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8::uuid, $9, $9
			)
		`, value.ID, principal.OrganizationID, documentID, value.Revision,
			value.EffectiveAt, value.ExpiresAt, value.Language, principal.ID, now)
		if err != nil {
			return revisionResult{}, err
		}
		for index := range value.Applicability {
			value.Applicability[index].ID = s.IDs.New()
			item := value.Applicability[index]
			_, err = tx.Exec(ctx, `
				INSERT INTO document_applicability (
					id, organization_id, revision_id, site_id, asset_id,
					asset_class, manufacturer, model, process_service, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, nullif($4, '')::uuid,
					nullif($5, '')::uuid, nullif($6, ''), nullif($7, ''),
					nullif($8, ''), nullif($9, ''), $10
				)
			`, item.ID, principal.OrganizationID, value.ID, item.SiteID, item.AssetID,
				item.AssetClass, item.Manufacturer, item.Model, item.ProcessService, now)
			if err != nil {
				return revisionResult{}, err
			}
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: principal.OrganizationID, SiteID: siteID,
			ActorID: principal.ID, Action: "knowledge.revision.created",
			EntityKind: "document_revision", EntityID: value.ID,
			After: value, OccurredAt: now,
		}); err != nil {
			return revisionResult{}, err
		}
		body, _ := json.Marshal(value)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusCreated, body, now,
		); err != nil {
			return revisionResult{}, err
		}
		return revisionResult{Value: value}, nil
	})
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	return result.Value, result.Replay, nil
}

func validateApplicability(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	documentSiteID string,
	items []knowledgedomain.Applicability,
) error {
	for _, item := range items {
		if item.SiteID != "" && item.SiteID != documentSiteID {
			return knowledgeapp.ErrForbidden
		}
		if item.AssetID == "" {
			continue
		}
		var siteID string
		err := tx.QueryRow(ctx, `
			SELECT site_id::text FROM assets
			WHERE id = $1::uuid AND organization_id = $2::uuid
		`, item.AssetID, principal.OrganizationID).Scan(&siteID)
		if errors.Is(err, pgx.ErrNoRows) {
			return knowledgeapp.ErrNotFound
		}
		if err != nil {
			return err
		}
		if siteID != documentSiteID {
			return knowledgeapp.ErrForbidden
		}
	}
	return nil
}

func encodeReplay(
	ctx context.Context,
	store idempotency.Store,
	tx pgx.Tx,
	principalID, scope, key string,
	status int,
	value any,
	now time.Time,
) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return store.Complete(ctx, tx, principalID, scope, key, status, body, now)
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return knowledgeapp.ErrNotFound
	}
	return fmt.Errorf("knowledge store: %w", err)
}
