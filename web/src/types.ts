export type Principal = {
  id: string;
  display_name: string;
  organization_id: string;
  site_ids: string[];
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

export type Incident = {
  id: string;
  site_id: string;
  asset_id: string;
  asset_tag?: string;
  number: string;
  summary: string;
  severity: "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";
  state: "OPEN" | "IN_PROGRESS" | "RESOLVED";
  detected_at: string;
  version: number;
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
  incident_number?: string;
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
  document_id?: string;
  revision_id?: string;
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

export type MaintenanceReport = {
  id: string;
  execution_id: string;
  revision: number;
  version: number;
  state: "DRAFT" | "SUBMITTED" | "APPROVED";
  structured_content: {
    summary: string;
    measurements: string[];
    observations: string[];
    actions: string[];
    outcome: string;
    evidence_ids: string[];
    unknowns: string[];
    requires_human_review: boolean;
  };
  evidence: Evidence[];
  provider?: string;
  model?: string;
  prompt_version?: string;
  asset_tag?: string;
  incident_number?: string;
  created_at: string;
  updated_at: string;
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

export type ListPage<T> = ListResponse<T> & {
  next_cursor: string | null;
  has_more: boolean;
};

export type Problem = {
  title: string;
  status: number;
  detail?: string;
};

export type MonitoringMetric = {
  day?: string;
  value: number;
  sample_count: number;
  dimensions: Record<string, unknown>;
  site_id?: string;
};

export type MonitoringSummaryEntry = {
  metric_key: string;
  status: "PASS" | "WARN" | "CRIT";
  value: number;
  unit: string;
  group: string;
  site_id?: string;
  collected_at?: string;
  open_alerts: number;
};

export type MonitoringAlert = {
  id: number;
  metric_key: string;
  site_id?: string;
  state: "WARN" | "CRIT";
  message: string;
  value: number;
  opened_at: string;
  resolved_at?: string;
};

export type MonitorThreshold = {
  metric_key: string;
  site_id?: string;
  comparator: "GT" | "GTE" | "LT" | "LTE";
  warn_value: number;
  crit_value: number;
  enabled: boolean;
};
