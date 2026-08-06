-- +goose Up
CREATE INDEX maintenance_reports_list_idx
    ON maintenance_reports(organization_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS maintenance_reports_list_idx;
