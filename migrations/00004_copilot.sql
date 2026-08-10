-- +goose Up
ALTER TABLE attachments DROP CONSTRAINT attachments_entity_kind_check;
ALTER TABLE attachments ADD CONSTRAINT attachments_entity_kind_check CHECK (
    entity_kind IN ('EXECUTION', 'OBSERVATION', 'INCIDENT', 'DOCUMENT_REVISION')
);

CREATE TABLE documents (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid REFERENCES sites(id),
    document_type text NOT NULL CHECK (
        document_type IN (
            'OEM_MANUAL', 'SOP', 'WORK_INSTRUCTION', 'DATASHEET',
            'INSPECTION_PROCEDURE', 'SAFETY_PROCEDURE', 'TECHNICAL_BULLETIN'
        )
    ),
    title text NOT NULL CHECK (btrim(title) <> ''),
    authority text NOT NULL CHECK (
        authority IN ('OEM', 'CORPORATE', 'SITE_APPROVED', 'REGULATORY', 'EXPERT_REFERENCE')
    ),
    source_reference text,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX documents_scope_idx ON documents(organization_id, site_id, document_type);

CREATE TABLE document_revisions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    document_id uuid NOT NULL REFERENCES documents(id),
    revision text NOT NULL CHECK (btrim(revision) <> ''),
    approval_status text NOT NULL DEFAULT 'DRAFT' CHECK (
        approval_status IN ('DRAFT', 'APPROVED', 'REVIEW_REQUIRED', 'SUPERSEDED', 'RETIRED')
    ),
    effective_at timestamptz,
    expires_at timestamptz,
    superseded_by_id uuid REFERENCES document_revisions(id),
    attachment_id uuid REFERENCES attachments(id),
    ingestion_state text NOT NULL DEFAULT 'AWAITING_UPLOAD' CHECK (
        ingestion_state IN ('AWAITING_UPLOAD', 'QUEUED', 'PROCESSING', 'READY', 'FAILED')
    ),
    ingestion_error text,
    content_sha256 text,
    language text NOT NULL DEFAULT 'en',
    approved_by uuid REFERENCES principals(id),
    approved_at timestamptz,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CHECK (expires_at IS NULL OR effective_at IS NULL OR expires_at > effective_at),
    CHECK (
        (approval_status = 'APPROVED' AND approved_by IS NOT NULL AND approved_at IS NOT NULL)
        OR approval_status <> 'APPROVED'
    ),
    UNIQUE(document_id, revision)
);

CREATE INDEX document_revisions_validity_idx
    ON document_revisions(organization_id, approval_status, ingestion_state, effective_at, expires_at);

CREATE TABLE document_applicability (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    revision_id uuid NOT NULL REFERENCES document_revisions(id) ON DELETE CASCADE,
    site_id uuid REFERENCES sites(id),
    asset_id uuid REFERENCES assets(id),
    asset_class text,
    manufacturer text,
    model text,
    process_service text,
    created_at timestamptz NOT NULL,
    CHECK (
        site_id IS NOT NULL OR asset_id IS NOT NULL OR asset_class IS NOT NULL OR
        manufacturer IS NOT NULL OR model IS NOT NULL OR process_service IS NOT NULL
    )
);

CREATE INDEX document_applicability_revision_idx ON document_applicability(revision_id);
CREATE INDEX document_applicability_asset_idx ON document_applicability(organization_id, site_id, asset_id);

CREATE TABLE document_chunks (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    revision_id uuid NOT NULL REFERENCES document_revisions(id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal > 0),
    locator text NOT NULL CHECK (btrim(locator) <> ''),
    content text NOT NULL CHECK (btrim(content) <> ''),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[a-f0-9]{64}$'),
    token_estimate integer NOT NULL CHECK (token_estimate > 0),
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(content, ''))
    ) STORED,
    created_at timestamptz NOT NULL,
    UNIQUE(revision_id, ordinal),
    UNIQUE(revision_id, content_sha256)
);

CREATE INDEX document_chunks_fts_idx ON document_chunks USING gin(search_vector);

CREATE TABLE embeddings (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    source_kind text NOT NULL CHECK (source_kind IN ('DOCUMENT_CHUNK', 'INCIDENT')),
    source_id uuid NOT NULL,
    source_revision text NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL,
    dimensions integer NOT NULL CHECK (dimensions > 0),
    distance_metric text NOT NULL CHECK (distance_metric = 'COSINE'),
    embedding vector NOT NULL,
    input_sha256 text NOT NULL CHECK (input_sha256 ~ '^[a-f0-9]{64}$'),
    state text NOT NULL CHECK (state IN ('ACTIVE', 'SUPERSEDED', 'FAILED')),
    created_at timestamptz NOT NULL,
    UNIQUE(source_kind, source_id, provider, model, model_version, input_sha256)
);

CREATE INDEX embeddings_source_idx
    ON embeddings(organization_id, source_kind, source_id, state);

CREATE TABLE retrieval_runs (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    actor_id uuid NOT NULL REFERENCES principals(id),
    query_sha256 text NOT NULL,
    context_sha256 text NOT NULL,
    filters jsonb NOT NULL,
    ranked_evidence jsonb NOT NULL,
    embedding_provider text,
    embedding_model text,
    embedding_version text,
    latency_ms bigint NOT NULL CHECK (latency_ms >= 0),
    created_at timestamptz NOT NULL
);

CREATE TABLE recommendations (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    incident_id uuid NOT NULL REFERENCES incidents(id),
    execution_id uuid REFERENCES maintenance_executions(id),
    status text NOT NULL CHECK (status IN ('RECOMMENDATION', 'INSUFFICIENT_EVIDENCE', 'FAILED_VALIDATION')),
    risk_level text NOT NULL CHECK (
        risk_level IN ('INFORMATIONAL', 'ADVISORY', 'OPERATIONAL_LOW', 'SAFETY_SIGNIFICANT')
    ),
    structured_output jsonb NOT NULL,
    evidence_snapshot jsonb NOT NULL,
    context_snapshot jsonb NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL,
    prompt_version text NOT NULL,
    input_sha256 text NOT NULL,
    output_sha256 text NOT NULL,
    latency_ms bigint NOT NULL CHECK (latency_ms >= 0),
    tokens_in integer,
    tokens_out integer,
    estimated_cost_micros bigint,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL
);

CREATE INDEX recommendations_incident_idx
    ON recommendations(organization_id, incident_id, created_at DESC);

CREATE TABLE recommendation_feedback (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    recommendation_id uuid NOT NULL REFERENCES recommendations(id),
    outcome text NOT NULL CHECK (outcome IN ('ACCEPTED', 'REJECTED', 'CORRECTED', 'UNSAFE')),
    correction text,
    reason text,
    actor_id uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL
);

ALTER TABLE maintenance_reports
    ADD COLUMN site_id uuid REFERENCES sites(id),
    ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN evidence_snapshot jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN provider text,
    ADD COLUMN model text,
    ADD COLUMN model_version text,
    ADD COLUMN prompt_version text,
    ADD COLUMN input_sha256 text,
    ADD COLUMN output_sha256 text,
    ADD COLUMN generated_by_kind text NOT NULL DEFAULT 'HUMAN' CHECK (
        generated_by_kind IN ('HUMAN', 'AI_DRAFT')
    ),
    ADD COLUMN created_by uuid REFERENCES principals(id),
    ADD COLUMN updated_at timestamptz;

UPDATE maintenance_reports r
SET site_id = e.site_id, updated_at = r.created_at
FROM maintenance_executions e
WHERE e.id = r.execution_id;

ALTER TABLE maintenance_reports
    ALTER COLUMN site_id SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL;

CREATE TABLE maintenance_report_edits (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    report_id uuid NOT NULL REFERENCES maintenance_reports(id),
    revision integer NOT NULL CHECK (revision > 0),
    previous_content jsonb NOT NULL,
    new_content jsonb NOT NULL,
    edited_by uuid NOT NULL REFERENCES principals(id),
    edited_at timestamptz NOT NULL,
    UNIQUE(report_id, revision)
);

CREATE TABLE shift_handovers (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    shift_start timestamptz NOT NULL,
    shift_end timestamptz NOT NULL,
    state text NOT NULL CHECK (state IN ('DRAFT', 'SUBMITTED', 'ACCEPTED', 'ACKNOWLEDGED')),
    structured_content jsonb NOT NULL,
    evidence_snapshot jsonb NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL,
    prompt_version text NOT NULL,
    input_sha256 text NOT NULL,
    output_sha256 text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES principals(id),
    submitted_by uuid REFERENCES principals(id),
    submitted_at timestamptz,
    accepted_by uuid REFERENCES principals(id),
    accepted_at timestamptz,
    acknowledged_by uuid REFERENCES principals(id),
    acknowledged_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (shift_end > shift_start)
);

CREATE INDEX shift_handovers_scope_idx
    ON shift_handovers(organization_id, site_id, shift_start DESC);

CREATE TABLE transcriptions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    attachment_id uuid NOT NULL REFERENCES attachments(id),
    state text NOT NULL CHECK (state IN ('QUEUED', 'PROCESSING', 'CANDIDATE', 'VERIFIED', 'REJECTED', 'FAILED')),
    transcript text,
    language text,
    provider text NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL,
    confidence numeric(5, 4),
    verification_status text NOT NULL CHECK (
        verification_status IN ('UNVERIFIED', 'VERIFIED', 'REJECTED')
    ),
    verified_by uuid REFERENCES principals(id),
    verified_at timestamptz,
    source_sha256 text NOT NULL,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE(attachment_id, provider, model, model_version)
);

CREATE TABLE ai_call_records (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid REFERENCES sites(id),
    capability text NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL,
    prompt_version text,
    input_sha256 text NOT NULL,
    output_sha256 text,
    outcome text NOT NULL CHECK (
        outcome IN ('SUCCEEDED', 'INSUFFICIENT_EVIDENCE', 'INVALID_OUTPUT', 'TIMEOUT', 'PROVIDER_ERROR')
    ),
    latency_ms bigint NOT NULL CHECK (latency_ms >= 0),
    tokens_in integer,
    tokens_out integer,
    estimated_cost_micros bigint,
    error_code text,
    created_at timestamptz NOT NULL
);

CREATE INDEX ai_call_records_scope_idx
    ON ai_call_records(organization_id, site_id, capability, created_at DESC);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            documents,
            document_revisions,
            document_applicability,
            document_chunks,
            embeddings,
            retrieval_runs,
            recommendations,
            recommendation_feedback,
            maintenance_report_edits,
            shift_handovers,
            transcriptions,
            ai_call_records
        TO skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS ai_call_records;
DROP TABLE IF EXISTS transcriptions;
DROP TABLE IF EXISTS shift_handovers;
DROP TABLE IF EXISTS maintenance_report_edits;
ALTER TABLE maintenance_reports
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS generated_by_kind,
    DROP COLUMN IF EXISTS output_sha256,
    DROP COLUMN IF EXISTS input_sha256,
    DROP COLUMN IF EXISTS prompt_version,
    DROP COLUMN IF EXISTS model_version,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS evidence_snapshot,
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS site_id;
DROP TABLE IF EXISTS recommendation_feedback;
DROP TABLE IF EXISTS recommendations;
DROP TABLE IF EXISTS retrieval_runs;
DROP TABLE IF EXISTS embeddings;
DROP TABLE IF EXISTS document_chunks;
DROP TABLE IF EXISTS document_applicability;
DROP TABLE IF EXISTS document_revisions;
DROP TABLE IF EXISTS documents;
ALTER TABLE attachments DROP CONSTRAINT attachments_entity_kind_check;
ALTER TABLE attachments ADD CONSTRAINT attachments_entity_kind_check CHECK (
    entity_kind IN ('EXECUTION', 'OBSERVATION', 'INCIDENT')
);
