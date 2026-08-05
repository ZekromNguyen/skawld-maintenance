-- +goose Up
-- The id_token (JWT) is passed as id_token_hint to the provider's
-- end_session_endpoint so RP-initiated logout destroys the SSO session,
-- preventing the browser from silently re-authenticating after sign-out.
ALTER TABLE web_sessions ADD COLUMN id_token text;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, UPDATE ON web_sessions TO skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON web_sessions TO skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE web_sessions DROP COLUMN IF EXISTS id_token;
