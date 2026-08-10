-- +goose Up
ALTER TABLE recommendation_feedback
    DROP CONSTRAINT recommendation_feedback_outcome_check,
    ADD CONSTRAINT recommendation_feedback_outcome_check CHECK (
        outcome IN (
            'ACCEPTED', 'REJECTED', 'CORRECTED', 'UNSAFE',
            'UNSUPPORTED', 'INCORRECT_NEXT_STEP'
        )
    ),
    ADD COLUMN material_claims integer CHECK (material_claims >= 0),
    ADD COLUMN supported_claims integer CHECK (
        supported_claims >= 0
        AND (material_claims IS NULL OR supported_claims <= material_claims)
    ),
    ADD COLUMN retrieved_evidence integer CHECK (retrieved_evidence >= 0),
    ADD COLUMN relevant_evidence integer CHECK (
        relevant_evidence >= 0
        AND (
            retrieved_evidence IS NULL
            OR relevant_evidence <= retrieved_evidence
        )
    );

CREATE INDEX recommendation_feedback_quality_idx
    ON recommendation_feedback(
        organization_id, outcome, created_at DESC
    );

-- Recommendation feedback is an accountable reviewer label. Corrections are
-- new rows; normal runtime operation must never rewrite or erase old labels.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        REVOKE UPDATE, DELETE, TRUNCATE
        ON recommendation_feedback
        FROM skawld_app;
        GRANT SELECT, INSERT
        ON recommendation_feedback
        TO skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- Runtime immutability is intentionally not relaxed on rollback.
DROP INDEX IF EXISTS recommendation_feedback_quality_idx;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM recommendation_feedback
        WHERE outcome IN ('UNSUPPORTED', 'INCORRECT_NEXT_STEP')
    ) THEN
        RAISE EXCEPTION
            'cannot roll back while Phase 5 recommendation labels exist';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE recommendation_feedback
    DROP COLUMN IF EXISTS relevant_evidence,
    DROP COLUMN IF EXISTS retrieved_evidence,
    DROP COLUMN IF EXISTS supported_claims,
    DROP COLUMN IF EXISTS material_claims,
    DROP CONSTRAINT recommendation_feedback_outcome_check,
    ADD CONSTRAINT recommendation_feedback_outcome_check CHECK (
        outcome IN ('ACCEPTED', 'REJECTED', 'CORRECTED', 'UNSAFE')
    );
