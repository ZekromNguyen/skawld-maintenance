-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        REVOKE UPDATE, DELETE, TRUNCATE ON
            demonstration_events,
            demonstration_reviews
        FROM skawld_app;
        REVOKE DELETE, TRUNCATE ON
            demonstrations,
            demonstration_event_redactions
        FROM skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- Runtime immutability is intentionally not relaxed on rollback.
