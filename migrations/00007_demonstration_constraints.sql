-- +goose Up
CREATE UNIQUE INDEX demonstrations_one_recording_subject_idx
    ON demonstrations(organization_id, subject_kind, subject_id)
    WHERE status = 'recording';

-- +goose Down
DROP INDEX IF EXISTS demonstrations_one_recording_subject_idx;
