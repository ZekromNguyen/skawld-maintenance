-- +goose Up
ALTER TABLE transcriptions
    ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0);

-- +goose Down
ALTER TABLE transcriptions DROP COLUMN IF EXISTS version;
