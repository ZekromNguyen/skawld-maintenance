export type Principal = {
  id: string;
  display_name: string;
  organization_id: string;
  site_ids: string[];
  roles?: string[];
  permissions: string[];
};

export type Criticality = {
  rating: "A" | "B" | "C";
  safety_impact: number;
  production_impact: number;
  rationale: string;
};

export type Asset = {
  id: string;
  site_id: string;
  parent_asset_id?: string;
  tag: string;
  name: string;
  class: string;
  manufacturer?: string;
  model?: string;
  status: string;
  source_of_truth: "OWNED_BY_SKAWLD" | "EXTERNAL_REFERENCE";
  criticality?: Criticality;
};

export type IncidentStatus = "OPEN" | "IN_PROGRESS" | "RESOLVED" | "CLOSED" | "REOPENED";
export type Priority = "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";

export type Incident = {
  id: string;
  site_id: string;
  asset_id: string;
  asset_tag?: string;
  number: string;
  summary: string;
  details?: string;
  priority: Priority;
  status: IncidentStatus;
  assignee_id?: string;
  assignee_name?: string;
  reporter_id?: string;
  reporter_name?: string;
  team_id?: string;
  team_name?: string;
  occurred_at?: string;
  detected_at: string;
  resolved_at?: string;
  time_to_complete_seconds?: number;
  version: number;
  custom_values?: Record<string, unknown>;
  attachments?: Attachment[];
};

export type Team = { id: string; name: string };
export type Person = { id: string; display_name: string };

export type CustomFieldType = "TEXT" | "NUMBER" | "DATE" | "SELECT" | "MULTI_SELECT";
export type CustomFieldStatus = "ACTIVE" | "RETIRED";

export type CustomFieldOption = { label: string; value: string };

export type CustomFieldConfig = {
  required?: boolean;
  max_length?: number;
  regex?: string;
  min?: number;
  max?: number;
  options?: CustomFieldOption[];
};

export type CustomFieldDefinition = {
  id: string;
  entity_type: string;
  key: string;
  label: string;
  description?: string;
  field_type: CustomFieldType;
  config: CustomFieldConfig;
  status: CustomFieldStatus;
  sort_order: number;
  version: number;
  created_at: string;
  updated_at: string;
  retired_at?: string;
  has_values?: boolean;
};

export type HistoryEntry = {
  incident_id: string;
  principal_id: string;
  value_before?: unknown;
  value_after: unknown;
  changed_at: string;
};

export type Attachment = {
  id: string;
  organization_id: string;
  site_id: string;
  entity_kind: "EXECUTION" | "OBSERVATION" | "INCIDENT" | "DOCUMENT_REVISION";
  entity_id: string;
  original_filename: string;
  declared_mime: string;
  verified_mime?: string;
  size_bytes: number;
  checksum_sha256: string;
  state: string;
  upload_url?: string;
  upload_headers?: Record<string, string>;
  download_url?: string;
};

export type Step = {
  id: string;
  key: string;
  sequence: number;
  title: string;
  state: "PENDING" | "COMPLETED" | "BLOCKED";
  risk_level: string;
  required_prerequisite?: string;
  blocked_reason?: string;
  version: number;
};

export type Measurement = {
  id: string;
  measurement_type: string;
  value: string;
  unit: string;
  observed_at: string;
};

export type Execution = {
  id: string;
  incident_id: string;
  asset_id: string;
  asset_tag: string;
  purpose: string;
  state: "ASSIGNED" | "IN_PROGRESS" | "COMPLETED";
  version: number;
  steps: Step[];
  measurements: Measurement[];
  observations: unknown[];
  actions: unknown[];
};

export type Applicability = {
  id?: string;
  site_id?: string;
  asset_id?: string;
  asset_class?: string;
  manufacturer?: string;
  model?: string;
};

export type DocumentRevision = {
  id: string;
  document_id: string;
  revision: string;
  approval_status: "DRAFT" | "APPROVED" | "REVIEW_REQUIRED" | "SUPERSEDED" | "RETIRED";
  ingestion_state: "AWAITING_UPLOAD" | "QUEUED" | "PROCESSING" | "READY" | "FAILED";
  ingestion_error?: string;
  language: string;
  version: number;
  applicability: Applicability[];
};

export type KnowledgeDocument = {
  id: string;
  site_id: string;
  document_type: string;
  title: string;
  authority: string;
  source_reference?: string;
  revisions: DocumentRevision[];
};

export type Evidence = {
  id: string;
  kind: string;
  source_id: string;
  revision?: string;
  title: string;
  locator: string;
  authority: string;
  content: string;
  content_sha256: string;
  score: { rrf_score: number };
};

export type Recommendation = {
  id: string;
  incident_id: string;
  execution_id?: string;
  output: {
    status: "RECOMMENDATION" | "INSUFFICIENT_EVIDENCE";
    recommendation: string;
    evidence_ids: string[];
    assumptions: string[];
    unknowns: string[];
    confidence: number;
    risk_level: "INFORMATIONAL" | "ADVISORY";
    requires_human_confirmation: true;
  };
  evidence: Evidence[];
  provider: string;
  model: string;
  model_version: string;
  prompt_version: string;
};

export type EvaluationSummary = {
  recommendations: number;
  reviewed: number;
  review_coverage: number;
  recommendation_acceptance: number;
  human_override_rate: number;
  unsafe_recommendation_rate: number;
  unsupported_recommendation_rate: number;
  incorrect_next_step_rate: number;
  evidence_coverage: number;
  retrieval_precision: number;
  llm_calls: number;
  average_latency_ms: number;
  tokens_in: number;
  tokens_out: number;
  estimated_cost_micros: number;
  workflow_evaluations: number;
  workflow_gate_pass_rate: number;
  generated_at: string;
};

export type DashboardSummary = {
  open_incidents: number;
  in_progress_incidents: number;
  resolved_incidents: number;
  total_incidents: number;
  by_severity: { LOW: number; MEDIUM: number; HIGH: number; CRITICAL: number };
  active_executions: number;
  critical_assets: number;
  pending_handovers: number;
};

export type ReportContentItem = string | Record<string, unknown>;

export type MaintenanceReport = {
  id: string;
  execution_id: string;
  revision: number;
  version: number;
  state: "DRAFT" | "SUBMITTED" | "APPROVED";
  structured_content: {
    summary: string;
    measurements: ReportContentItem[];
    observations: ReportContentItem[];
    actions: ReportContentItem[];
    outcome: string;
    evidence_ids: string[];
    unknowns: ReportContentItem[];
    requires_human_review: boolean;
  };
  evidence: Evidence[];
  provider?: string;
  model?: string;
  prompt_version?: string;
};

export type HandoverListItem = {
  title: string;
  detail?: string;
  severity?: string;
};

export type ShiftHandover = {
  id: string;
  site_id: string;
  shift_start: string;
  shift_end: string;
  state: "DRAFT" | "SUBMITTED" | "ACCEPTED" | "ACKNOWLEDGED";
  version: number;
  structured_content: {
    summary: string;
    open_incidents: HandoverListItem[];
    active_executions: HandoverListItem[];
    safety_concerns: HandoverListItem[];
    follow_up: HandoverListItem[];
    evidence_ids: string[];
    unknowns: HandoverListItem[];
    requires_human_review: boolean;
  };
  evidence: Evidence[];
  provider: string;
  model: string;
  prompt_version: string;
};

export type DemonstrationEvent = {
  id: string;
  ordinal: number;
  schema_version: string;
  timestamp: string;
  actor_id: string;
  roles: string[];
  source: string;
  trust: string;
  sensitivity: string;
  action: string;
  intent?: string;
  entity?: { type: string; id?: string };
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  context?: Record<string, unknown>;
  decision?: Record<string, unknown>;
  result?: Record<string, unknown>;
  correction_of?: string;
  domain_event_id?: string;
  provenance: Record<string, unknown>;
  redactions?: Array<{
    id: string;
    json_path: string;
    action: "MASK" | "DROP";
    reason: string;
  }>;
};

export type Demonstration = {
  id: string;
  site_id: string;
  subject_kind: "EXECUTION" | "HANDOVER";
  subject_id: string;
  workflow_key: string;
  schema_version: string;
  session_id: string;
  status: "recording" | "completed" | "rejected";
  review_status: "PENDING" | "APPROVED" | "REJECTED" | "REDACTION_REQUIRED";
  initial_context: Record<string, unknown>;
  final_result?: Record<string, unknown>;
  events: DemonstrationEvent[];
  capture: {
    pending: number;
    processing: number;
    failed: number;
    applied: number;
    last_error?: string;
  };
  started_at: string;
  completed_at?: string;
  created_by: string;
};

export type WorkflowApplicability = {
  id?: string;
  site_id?: string;
  asset_id?: string;
  asset_class?: string;
  manufacturer?: string;
  model?: string;
  process_service?: string;
  operating_condition?: string;
  validation_status:
    | "VALIDATED"
    | "LIKELY_APPLICABLE"
    | "NOT_VALIDATED"
    | "NOT_APPLICABLE";
  approved_by?: string;
  approved_at?: string;
};

export type WorkflowVersion = {
  workflow_id: string;
  workflow_key: string;
  name: string;
  description?: string;
  version: number;
  status:
    | "CANDIDATE"
    | "APPROVED"
    | "REVIEW_REQUIRED"
    | "PUBLISHED"
    | "RETIRED"
    | "REJECTED";
  site_id: string;
  asset_class: string;
  candidate_digest: string;
  tool_catalog_digest: string;
  source_demonstration_ids: string[];
  steps: Array<{
    id: string;
    name?: string;
    kind: string;
    tool_name?: string;
    arguments?: Record<string, unknown>;
    evidence: Array<{
      demonstration_id: string;
      event_ids: string[];
    }>;
  }>;
  analysis: {
    sequence_consistency?: number;
    conflicts?: unknown[];
    sequence_variants?: unknown[];
    findings?: unknown[];
  };
  behavioral_changes: {
    base_version?: number;
    input_schema_changed?: boolean;
    context_schema_changed?: boolean;
    steps?: unknown[];
  };
  learning: Record<string, unknown>;
  applicability: WorkflowApplicability[];
  prerequisites: string[];
  required_competencies: string[];
  effective_at?: string;
  review_at?: string;
  reviews: Array<{
    id: string;
    candidate_digest: string;
    decision: string;
    reason: string;
    reviewed_by: string;
    reviewed_at: string;
  }>;
  evaluations: Array<{
    id: string;
    suite_name: string;
    gates_passed: boolean;
    metrics: Record<string, unknown>;
    completed_at: string;
  }>;
  improvement_candidates: Array<{
    id: string;
    demonstration_id: string;
    correction_event_id: string;
    corrected_event_id: string;
    corrected_action: string;
    reason?: string;
    status: string;
  }>;
  created_at: string;
  published_at?: string;
  published_by?: string;
};

export type ListResponse<T> = { items: T[] };

export type IncidentListResponse = ListPage<Incident> & {
  custom_fields: CustomFieldDefinition[];
};

export type IncidentDetail = Incident & {
  custom_fields: CustomFieldDefinition[];
};

export type ListPage<T> = ListResponse<T> & {
  next_cursor: string | null;
  has_more: boolean;
};

export type Problem = {
  title: string;
  status: number;
  detail?: string;
};
