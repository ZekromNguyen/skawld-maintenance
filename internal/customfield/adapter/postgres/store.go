package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

const selectDefinition = `
	SELECT id::text, entity_type, key, label, coalesce(description, ''),
	       field_type, config, status, sort_order, version,
	       created_at, updated_at, retired_at
	FROM field_definitions
`

func scanDefinition(row scanner) (domain.Definition, error) {
	var value domain.Definition
	var rawConfig []byte
	err := row.Scan(
		&value.ID, &value.EntityType, &value.Key, &value.Label, &value.Description,
		&value.FieldType, &rawConfig, &value.Status, &value.SortOrder, &value.Version,
		&value.CreatedAt, &value.UpdatedAt, &value.RetiredAt,
	)
	if err != nil {
		return domain.Definition{}, err
	}
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &value.Config); err != nil {
			return domain.Definition{}, err
		}
	}
	return value, nil
}

type scanner interface {
	Scan(...any) error
}

func (s Store) ListByEntity(ctx context.Context, organizationID, entityType string) ([]domain.Definition, error) {
	rows, err := s.Pool.Query(ctx, selectDefinition+`
		WHERE organization_id = $1::uuid AND entity_type = $2
		ORDER BY sort_order, key
	`, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Definition
	for rows.Next() {
		value, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (s Store) Get(ctx context.Context, organizationID, id string) (domain.Definition, error) {
	value, err := scanDefinition(s.Pool.QueryRow(ctx, selectDefinition+`
		WHERE id = $1::uuid AND organization_id = $2::uuid
	`, id, organizationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Definition{}, application.ErrNotFound
	}
	return value, err
}

func (s Store) Create(ctx context.Context, organizationID string, d domain.Definition, now time.Time) (domain.Definition, error) {
	rawConfig, err := json.Marshal(d.Config)
	if err != nil {
		return domain.Definition{}, err
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO field_definitions (
			organization_id, entity_type, key, label, description, field_type,
			config, status, sort_order, version, created_at, updated_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $11)
		RETURNING id::text, created_at, updated_at
	`, organizationID, d.EntityType, d.Key, d.Label, d.Description,
		string(d.FieldType), rawConfig, string(d.Status), d.SortOrder, d.Version, now)
	if err := row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// UNIQUE (organization_id, entity_type, key) violation: a tenant
			// reused a field key.
			return domain.Definition{}, application.ErrConflict
		}
		return domain.Definition{}, err
	}
	return d, nil
}

func (s Store) Update(ctx context.Context, organizationID, id string, d domain.Definition, now time.Time) (domain.Definition, error) {
	rawConfig, err := json.Marshal(d.Config)
	if err != nil {
		return domain.Definition{}, err
	}
	tag, err := s.Pool.Exec(ctx, `
		UPDATE field_definitions
		SET label = $3, description = $4, field_type = $5, config = $6::jsonb,
		    sort_order = $7, version = $8, updated_at = $9
		WHERE id = $1::uuid AND organization_id = $2::uuid AND version = $10
		  AND (
		    ($5 = field_type AND $6::jsonb = config)
		    OR NOT EXISTS (
		      SELECT 1 FROM incidents i
		      WHERE i.organization_id = $2::uuid AND i.custom_values ? $1::text
		    )
		  )
	`, id, organizationID, d.Label, d.Description, string(d.FieldType),
		rawConfig, d.SortOrder, d.Version, now, d.Version-1)
	if err != nil {
		return domain.Definition{}, err
	}
	if tag.RowsAffected() != 1 {
		return domain.Definition{}, application.ErrConflict
	}
	return s.Get(ctx, organizationID, id)
}

func (s Store) Retire(ctx context.Context, organizationID, id string, now time.Time) (domain.Definition, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE field_definitions
		SET status = 'RETIRED', retired_at = $3, version = version + 1, updated_at = $3
		WHERE id = $1::uuid AND organization_id = $2::uuid AND status = 'ACTIVE'
	`, id, organizationID, now)
	if err != nil {
		return domain.Definition{}, err
	}
	if tag.RowsAffected() != 1 {
		return domain.Definition{}, application.ErrConflict
	}
	return s.Get(ctx, organizationID, id)
}

func (s Store) History(ctx context.Context, organizationID, id string) ([]application.HistoryEntry, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT h.incident_id::text, h.principal_id::text,
		       h.value_before, h.value_after, h.changed_at
		FROM custom_value_history h
		JOIN field_definitions f ON f.id = h.field_definition_id
		WHERE f.id = $1::uuid AND f.organization_id = $2::uuid
		ORDER BY h.changed_at DESC
		LIMIT 100
	`, id, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []application.HistoryEntry
	for rows.Next() {
		var entry application.HistoryEntry
		var before, after []byte
		if err := rows.Scan(&entry.IncidentID, &entry.PrincipalID, &before, &after, &entry.ChangedAt); err != nil {
			return nil, err
		}
		if len(before) > 0 {
			if err := json.Unmarshal(before, &entry.ValueBefore); err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(after, &entry.ValueAfter); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

// Usage reports which definitions already hold at least one incident value,
// keyed by definition id, for the org and entity.
func (s Store) Usage(ctx context.Context, organizationID, entityType string) (map[string]bool, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT f.id::text,
		       EXISTS (
		         SELECT 1 FROM incidents i
		         WHERE i.organization_id = f.organization_id AND i.custom_values ? f.id::text
		       )
		FROM field_definitions f
		WHERE f.organization_id = $1::uuid AND f.entity_type = $2
	`, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var id string
		var used bool
		if err := rows.Scan(&id, &used); err != nil {
			return nil, err
		}
		out[id] = used
	}
	return out, rows.Err()
}

func (s Store) HasValues(ctx context.Context, organizationID, id string) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM incidents
			WHERE organization_id = $1::uuid AND custom_values ? $2::text
		)
	`, organizationID, id).Scan(&exists)
	return exists, err
}
