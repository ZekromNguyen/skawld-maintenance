-- +goose Up
CREATE TABLE maintenance_workflows (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    workflow_key text NOT NULL CHECK (btrim(workflow_key) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    description text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, workflow_key)
);

CREATE TABLE workflow_versions (
    workflow_id uuid NOT NULL REFERENCES maintenance_workflows(id),
    version integer NOT NULL CHECK (version > 0),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    asset_class text NOT NULL CHECK (btrim(asset_class) <> ''),
    status text NOT NULL CHECK (
        status IN (
            'CANDIDATE', 'APPROVED', 'REVIEW_REQUIRED',
            'PUBLISHED', 'RETIRED', 'REJECTED'
        )
    ),
    sdk_schema_version text NOT NULL,
    sdk_payload jsonb NOT NULL,
    candidate_digest text NOT NULL CHECK (length(candidate_digest) = 64),
    analysis jsonb NOT NULL DEFAULT '{}'::jsonb,
    behavioral_changes jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_demonstration_ids uuid[] NOT NULL,
    prerequisites text[] NOT NULL DEFAULT '{}',
    required_competencies text[] NOT NULL DEFAULT '{}',
    effective_at timestamptz,
    review_at timestamptz,
    created_by uuid NOT NULL REFERENCES principals(id),
    approved_by uuid REFERENCES principals(id),
    approved_at timestamptz,
    published_by uuid REFERENCES principals(id),
    published_at timestamptz,
    retired_by uuid REFERENCES principals(id),
    retired_at timestamptz,
    superseded_by_version integer,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (workflow_id, version),
    CHECK (cardinality(source_demonstration_ids) >= 2),
    CHECK (review_at IS NULL OR effective_at IS NULL OR review_at > effective_at),
    CHECK (
        (status <> 'PUBLISHED' OR (
            approved_by IS NOT NULL AND approved_at IS NOT NULL
            AND published_by IS NOT NULL AND published_at IS NOT NULL
            AND effective_at IS NOT NULL AND review_at IS NOT NULL
        ))
    )
);

CREATE INDEX workflow_versions_scope_status_idx
    ON workflow_versions(organization_id, site_id, asset_class, status);

CREATE UNIQUE INDEX workflow_versions_one_published_idx
    ON workflow_versions(workflow_id)
    WHERE status = 'PUBLISHED';

CREATE TABLE workflow_applicability (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    workflow_id uuid NOT NULL,
    workflow_version integer NOT NULL,
    site_id uuid REFERENCES sites(id),
    asset_id uuid REFERENCES assets(id),
    asset_class text,
    manufacturer text,
    model text,
    process_service text,
    operating_condition text,
    validation_status text NOT NULL CHECK (
        validation_status IN (
            'VALIDATED', 'LIKELY_APPLICABLE',
            'NOT_VALIDATED', 'NOT_APPLICABLE'
        )
    ),
    approved_by uuid NOT NULL REFERENCES principals(id),
    approved_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    FOREIGN KEY (workflow_id, workflow_version)
        REFERENCES workflow_versions(workflow_id, version),
    CHECK (
        site_id IS NOT NULL OR asset_id IS NOT NULL
        OR nullif(btrim(asset_class), '') IS NOT NULL
        OR nullif(btrim(manufacturer), '') IS NOT NULL
        OR nullif(btrim(model), '') IS NOT NULL
        OR nullif(btrim(process_service), '') IS NOT NULL
    )
);

CREATE INDEX workflow_applicability_lookup_idx
    ON workflow_applicability(
        organization_id, site_id, asset_id, asset_class,
        manufacturer, model, validation_status
    );

CREATE TABLE workflow_reviews (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    workflow_id uuid NOT NULL,
    workflow_version integer NOT NULL,
    candidate_digest text NOT NULL CHECK (length(candidate_digest) = 64),
    decision text NOT NULL CHECK (
        decision IN ('APPROVED', 'REJECTED', 'REVIEW_REQUIRED')
    ),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    applicability_snapshot jsonb NOT NULL DEFAULT '[]'::jsonb,
    prerequisites text[] NOT NULL DEFAULT '{}',
    required_competencies text[] NOT NULL DEFAULT '{}',
    effective_at timestamptz,
    review_at timestamptz,
    reviewed_by uuid NOT NULL REFERENCES principals(id),
    reviewed_at timestamptz NOT NULL,
    FOREIGN KEY (workflow_id, workflow_version)
        REFERENCES workflow_versions(workflow_id, version)
);

CREATE INDEX workflow_reviews_timeline_idx
    ON workflow_reviews(workflow_id, workflow_version, reviewed_at);

CREATE TABLE workflow_evaluation_reports (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    workflow_id uuid NOT NULL,
    workflow_version integer NOT NULL,
    workflow_digest text NOT NULL CHECK (length(workflow_digest) = 64),
    suite_name text NOT NULL CHECK (btrim(suite_name) <> ''),
    gates_passed boolean NOT NULL,
    sdk_payload jsonb NOT NULL,
    started_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    FOREIGN KEY (workflow_id, workflow_version)
        REFERENCES workflow_versions(workflow_id, version)
);

CREATE INDEX workflow_evaluations_candidate_idx
    ON workflow_evaluation_reports(
        workflow_id, workflow_version, suite_name, completed_at DESC
    );

CREATE TABLE workflow_improvement_candidates (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    workflow_id uuid NOT NULL REFERENCES maintenance_workflows(id),
    workflow_version integer NOT NULL,
    demonstration_id uuid NOT NULL REFERENCES demonstrations(id),
    correction_event_id uuid NOT NULL REFERENCES demonstration_events(id),
    corrected_event_id uuid NOT NULL REFERENCES demonstration_events(id),
    corrected_action text NOT NULL CHECK (btrim(corrected_action) <> ''),
    reason text,
    context_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    outcome_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'ACCEPTED', 'REJECTED', 'INCORPORATED')),
    created_at timestamptz NOT NULL,
    UNIQUE (workflow_id, workflow_version, correction_event_id),
    FOREIGN KEY (workflow_id, workflow_version)
        REFERENCES workflow_versions(workflow_id, version)
);

CREATE INDEX workflow_improvements_status_idx
    ON workflow_improvement_candidates(
        organization_id, workflow_id, workflow_version, status, created_at
    );

-- SDK payload and compiler evidence are immutable once a version has entered
-- publication. Retirement changes lifecycle metadata only.
-- +goose StatementBegin
CREATE FUNCTION guard_published_workflow_version() RETURNS trigger AS $$
BEGIN
    IF OLD.status IN ('PUBLISHED', 'RETIRED') AND (
        NEW.sdk_payload IS DISTINCT FROM OLD.sdk_payload
        OR NEW.candidate_digest IS DISTINCT FROM OLD.candidate_digest
        OR NEW.analysis IS DISTINCT FROM OLD.analysis
        OR NEW.behavioral_changes IS DISTINCT FROM OLD.behavioral_changes
        OR NEW.source_demonstration_ids IS DISTINCT FROM OLD.source_demonstration_ids
        OR NEW.prerequisites IS DISTINCT FROM OLD.prerequisites
        OR NEW.required_competencies IS DISTINCT FROM OLD.required_competencies
        OR NEW.effective_at IS DISTINCT FROM OLD.effective_at
        OR NEW.review_at IS DISTINCT FROM OLD.review_at
    ) THEN
        RAISE EXCEPTION 'published workflow version is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER workflow_version_immutability
BEFORE UPDATE ON workflow_versions
FOR EACH ROW EXECUTE FUNCTION guard_published_workflow_version();

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE ON
            maintenance_workflows,
            workflow_versions,
            workflow_improvement_candidates
        TO skawld_app;
        GRANT SELECT, INSERT ON
            workflow_applicability,
            workflow_reviews,
            workflow_evaluation_reports
        TO skawld_app;
        REVOKE DELETE, TRUNCATE ON
            maintenance_workflows,
            workflow_versions,
            workflow_applicability,
            workflow_reviews,
            workflow_evaluation_reports,
            workflow_improvement_candidates
        FROM skawld_app;
        REVOKE UPDATE ON
            workflow_applicability,
            workflow_reviews,
            workflow_evaluation_reports
        FROM skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS workflow_version_immutability ON workflow_versions;
DROP FUNCTION IF EXISTS guard_published_workflow_version();
DROP TABLE IF EXISTS workflow_improvement_candidates;
DROP TABLE IF EXISTS workflow_evaluation_reports;
DROP TABLE IF EXISTS workflow_reviews;
DROP TABLE IF EXISTS workflow_applicability;
DROP TABLE IF EXISTS workflow_versions;
DROP TABLE IF EXISTS maintenance_workflows;
