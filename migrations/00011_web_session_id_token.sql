-- +goose Up
-- The id_token (JWT) is passed as id_token_hint to the provider's
-- end_session_endpoint so RP-initiated logout destroys the SSO session,
-- preventing the browser from silently re-authenticating after sign-out.
ALTER TABLE web_sessions ADD COLUMN id_token text;

-- +goose Down
ALTER TABLE web_sessions DROP COLUMN IF EXISTS id_token;
