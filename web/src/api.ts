import type {
  Asset,
  Demonstration,
  EvaluationSummary,
  Execution,
  KnowledgeDocument,
  ListPage,
  MaintenanceReport,
  Incident,
  ListResponse,
  Measurement,
  Principal,
  Problem,
  Recommendation,
  ShiftHandover,
  Step,
  WorkflowApplicability,
  WorkflowVersion
} from "./types";

export class ApiError extends Error {
  readonly status: number;

  constructor(problem: Problem) {
    super(problem.detail || problem.title);
    this.status = problem.status;
  }
}

/** Parse an error body defensively: non-JSON bodies (proxy/gateway HTML,
 * empty 5xx) must not crash the error path with a SyntaxError. */
async function parseProblem(response: Response): Promise<Problem> {
  try {
    const body = (await response.json()) as Partial<Problem>;
    if (body && typeof body === "object" && typeof body.status === "number") {
      return {
        status: body.status,
        title: body.title ?? "Request failed",
        detail: body.detail
      };
    }
  } catch {
    // non-JSON error body; fall through to a status-derived problem
  }
  return {
    status: response.status,
    title: "Request failed",
    detail: `Request failed with status ${response.status}`
  };
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`/api/v1${path}`, {
      ...init,
      credentials: "include",
      headers: {
        Accept: "application/json",
        ...(init?.body ? { "Content-Type": "application/json" } : {}),
        ...init?.headers
      }
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new ApiError({ status: 0, title: "Network error", detail: `Network error: ${message}` });
  }
  if (response.status === 401) {
    const login = new URL("/auth/login", window.location.origin);
    login.searchParams.set("return_to", window.location.pathname + window.location.search);
    window.location.assign(login.toString());
    throw new ApiError({ title: "Authentication required", status: 401 });
  }
  if (!response.ok) {
    throw new ApiError(await parseProblem(response));
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

function command<T>(path: string, value: unknown): Promise<T> {
  return request<T>(path, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify(value)
  });
}

export interface ListOptions {
  site_id?: string;
  state?: string[];
  cursor?: string;
  page_size?: number;
}

function listQuery(options?: ListOptions): string {
  const params = new URLSearchParams();
  if (options?.site_id) params.set("site_id", options.site_id);
  for (const state of options?.state ?? []) params.append("state", state);
  if (options?.cursor) params.set("cursor", options.cursor);
  if (options?.page_size) params.set("page_size", String(options.page_size));
  const query = params.toString();
  return query ? `?${query}` : "";
}

export async function fetchAll<T>(
  first: ListPage<T>,
  fetchPage: (cursor: string) => Promise<ListPage<T>>,
): Promise<T[]> {
  const items = [...first.items];
  let cursor = first.next_cursor;
  while (cursor) {
    const page = await fetchPage(cursor);
    items.push(...page.items);
    cursor = page.next_cursor;
  }
  return items;
}

export const api = {
  principal: () => request<Principal>("/me"),
  assets: (options?: ListOptions) =>
    request<ListPage<Asset>>(`/assets${listQuery(options)}`),
  createAsset: (value: {
    site_id: string;
    tag: string;
    name: string;
    class: string;
  }) =>
    command<Asset>("/assets", {
      ...value,
      source_of_truth: "OWNED_BY_SKAWLD",
      components: []
    }),
  incidents: (options?: ListOptions) =>
    request<ListPage<Incident>>(`/incidents${listQuery(options)}`),
  createIncident: (value: {
    site_id: string;
    asset_id: string;
    summary: string;
    severity: string;
  }) =>
    command<Incident>("/incidents", {
      ...value,
      source_of_truth: "OWNED_BY_SKAWLD",
      detected_at: new Date().toISOString()
    }),
  execution: (id: string) => request<Execution>(`/executions/${id}`),
  createExecution: (incidentID: string) =>
    command<Execution>(`/incidents/${incidentID}/executions`, {
      purpose: "High vibration pump inspection"
    }),
  startExecution: (execution: Execution) =>
    command<Execution>(`/executions/${execution.id}/start`, {
      expected_version: execution.version
    }),
  completeStep: (execution: Execution, step: Step) =>
    command<Step>(`/executions/${execution.id}/steps/${step.id}/complete`, {
      expected_execution_version: execution.version,
      expected_step_version: step.version
    }),
  recordMeasurement: (
    execution: Execution,
    measurementType: string,
    value: string,
    unit: string
  ) =>
    command<Measurement>(`/executions/${execution.id}/measurements`, {
      client_event_id: crypto.randomUUID(),
      measurement_type: measurementType,
      value,
      unit,
      source: "MANUAL",
      data_quality: "GOOD",
      verification_status: "UNVERIFIED",
      observed_at: new Date().toISOString()
    }),
  documents: (siteID?: string, options?: ListOptions) =>
    request<ListPage<KnowledgeDocument>>(
      `/documents${listQuery({ site_id: siteID, ...(options ?? {}) })}`
    ),
  createDocument: (siteID: string, title: string, documentType: string) =>
    command<KnowledgeDocument>("/documents", {
      site_id: siteID,
      document_type: documentType,
      title,
      authority: "SITE_APPROVED"
    }),
  createDocumentRevision: (
    documentID: string,
    siteID: string,
    assetClass: string
  ) =>
    command<KnowledgeDocument["revisions"][number]>(
      `/documents/${documentID}/revisions`,
      {
        revision: "R1",
        language: "en",
        applicability: [{ site_id: siteID, asset_class: assetClass }]
      }
    ),
  uploadDocumentRevision: async (
    revisionID: string,
    siteID: string,
    file: File
  ) => {
    const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
    const checksum = Array.from(new Uint8Array(digest))
      .map((value) => value.toString(16).padStart(2, "0"))
      .join("");
    const manifest = await command<{
      id: string;
      upload_url: string;
      upload_headers?: Record<string, string>;
    }>("/attachments", {
      site_id: siteID,
      entity_kind: "DOCUMENT_REVISION",
      entity_id: revisionID,
      client_event_id: crypto.randomUUID(),
      original_filename: file.name,
      declared_mime: file.type || "application/pdf",
      size_bytes: file.size,
      checksum_sha256: checksum
    });
    const upload = await fetch(manifest.upload_url, {
      method: "PUT",
      headers: {
        "Content-Type": file.type || "application/pdf",
        ...manifest.upload_headers
      },
      body: file
    });
    if (!upload.ok) throw new Error("Object upload failed");
    await command(`/attachments/${manifest.id}/complete`, {});
    return command(`/document-revisions/${revisionID}/ingestion`, {
      attachment_id: manifest.id
    });
  },
  searchKnowledge: (siteID: string, query: string, assetID?: string, limit?: number) =>
    request<{ retrieval_run_id: string; items: import("./types").Evidence[] }>(
      "/search",
      {
        method: "POST",
        body: JSON.stringify({
          site_id: siteID,
          asset_id: assetID || "",
          query,
          limit: limit ?? 8
        })
      }
    ),
  recommendation: (incidentID: string, executionID?: string) =>
    command<Recommendation>(`/incidents/${incidentID}/recommendations`, {
      execution_id: executionID || "",
      question: "What is the next safe non-intrusive inspection step?"
    }),
  reviewRecommendation: (
    recommendationID: string,
    value: {
      outcome:
        | "ACCEPTED"
        | "REJECTED"
        | "CORRECTED"
        | "UNSAFE"
        | "UNSUPPORTED"
        | "INCORRECT_NEXT_STEP";
      reason: string;
      material_claims?: number;
      supported_claims?: number;
      retrieved_evidence?: number;
      relevant_evidence?: number;
    }
  ) => command<void>(`/recommendations/${recommendationID}/feedback`, value),
  evaluationSummary: () =>
    request<EvaluationSummary>("/evaluations/summary"),
  draftReport: (executionID: string) =>
    command<MaintenanceReport>(`/executions/${executionID}/reports/draft`, {}),
  prepareHandover: (siteID: string) => {
    const end = new Date();
    const start = new Date(end.getTime() - 12 * 60 * 60 * 1000);
    return command<ShiftHandover>("/handovers/prepare-draft", {
      site_id: siteID,
      shift_start: start.toISOString(),
      shift_end: end.toISOString()
    });
  },
  demonstrations: (siteID?: string, options?: ListOptions) =>
    request<ListPage<Demonstration>>(
      `/demonstrations${listQuery({ site_id: siteID, ...(options ?? {}) })}`
    ),
  demonstration: (id: string) =>
    request<Demonstration>(`/demonstrations/${id}`),
  startDemonstration: (
    subjectKind: "EXECUTION" | "HANDOVER",
    subjectID: string
  ) =>
    command<Demonstration>("/demonstrations", {
      subject_kind: subjectKind,
      subject_id: subjectID
    }),
  completeDemonstration: (id: string, outcome: string) =>
    command<Demonstration>(`/demonstrations/${id}/complete`, { outcome }),
  recordEvidenceView: (id: string, evidenceID: string, intent: string) =>
    command(`/demonstrations/${id}/evidence-views`, {
      evidence_id: evidenceID,
      intent
    }),
  reviewDemonstration: (
    id: string,
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED",
    reason: string
  ) =>
    command(`/demonstrations/${id}/reviews`, { decision, reason }),
  redactDemonstrationEvent: (
    demonstrationID: string,
    eventID: string,
    jsonPath: string,
    reason: string
  ) =>
    command(`/demonstrations/${demonstrationID}/events/${eventID}/redactions`, {
      json_path: jsonPath,
      action: "MASK",
      reason
    }),
  workflows: (options?: ListOptions) =>
    request<ListPage<WorkflowVersion>>(`/workflows${listQuery(options)}`),
  workflow: (workflowID: string, version: number) =>
    request<WorkflowVersion>(
      `/workflows/${workflowID}/versions/${version}`
    ),
  compileWorkflow: (name: string, demonstrationIDs: string[]) =>
    command<WorkflowVersion>("/workflow-candidates", {
      name,
      description: "Compiled from reviewed semantic demonstrations",
      demonstration_ids: demonstrationIDs
    }),
  reviewWorkflow: (
    value: WorkflowVersion,
    decision: "APPROVED" | "REJECTED" | "REVIEW_REQUIRED",
    reason: string,
    applicability: WorkflowApplicability[]
  ) => {
    const effective = new Date();
    const review = new Date(effective);
    review.setUTCFullYear(review.getUTCFullYear() + 1);
    return command<WorkflowVersion>(
      `/workflows/${value.workflow_id}/versions/${value.version}/reviews`,
      {
        decision,
        reason,
        applicability: decision === "APPROVED" ? applicability : [],
        prerequisites:
          decision === "APPROVED"
            ? ["ENERGY_ISOLATION_WHEN_INTRUSIVE"]
            : [],
        required_competencies:
          decision === "APPROVED" ? [value.asset_class] : [],
        effective_at:
          decision === "APPROVED" ? effective.toISOString() : undefined,
        review_at:
          decision === "APPROVED" ? review.toISOString() : undefined
      }
    );
  },
  publishWorkflow: (value: WorkflowVersion, reason: string) =>
    command<WorkflowVersion>(
      `/workflows/${value.workflow_id}/versions/${value.version}/publish`,
      { reason }
    ),
  expandWorkflowApplicability: (
    value: WorkflowVersion,
    applicability: WorkflowApplicability,
    reason: string
  ) =>
    command<WorkflowVersion>(
      `/workflows/${value.workflow_id}/versions/${value.version}/applicability`,
      { reason, applicability }
    ),
  retireWorkflow: (value: WorkflowVersion, reason: string) =>
    command<WorkflowVersion>(
      `/workflows/${value.workflow_id}/versions/${value.version}/retire`,
      { reason }
    ),
  applicableWorkflows: (assetID: string) =>
    request<ListResponse<WorkflowVersion>>(
      `/workflows/applicable?asset_id=${encodeURIComponent(assetID)}`
    ),
  asset: (id: string) => request<Asset>(`/assets/${id}`),
  approveAssetCriticality: (assetID: string) =>
    command<unknown>(`/assets/${assetID}/criticality-approvals`, {}),
  incident: (id: string) => request<Incident>(`/incidents/${id}`),
  listExecutions: (options?: ListOptions) =>
    request<ListPage<Execution>>(`/executions${listQuery(options)}`),
  resolveIncident: (incidentID: string) =>
    command<Incident>(`/incidents/${incidentID}/resolution`, {}),
  generateRecommendation: (incidentID: string) =>
    command<Recommendation>(`/incidents/${incidentID}/recommendations`, {}),
  recommendationFeedback: (
    id: string,
    value: { accepted: boolean; correction?: string }
  ) => command<unknown>(`/recommendations/${id}/feedback`, value),
  reports: (options: ListOptions = {}) =>
    request<ListPage<MaintenanceReport>>(`/reports${listQuery(options)}`),
  report: (id: string) => request<MaintenanceReport>(`/reports/${id}`),
  submitReport: (id: string) => command<MaintenanceReport>(`/reports/${id}/submit`, {}),
  approveReport: (id: string) => command<MaintenanceReport>(`/reports/${id}/approve`, {}),
  editReport: (id: string, value: unknown) =>
    command<MaintenanceReport>(`/reports/${id}/edit`, value),
  document: (id: string) => request<KnowledgeDocument>(`/documents/${id}`),
  approveDocumentRevision: (revisionID: string) =>
    command<unknown>(`/document-revisions/${revisionID}/approve`, {}),
  retireDocumentRevision: (revisionID: string, reason: string) =>
    command<unknown>(`/document-revisions/${revisionID}/retire`, { reason }),
  requestDocumentIngestion: (revisionID: string) =>
    command<unknown>(`/document-revisions/${revisionID}/ingestion`, {}),
  handovers: (options: ListOptions = {}) =>
    request<ListPage<ShiftHandover>>(`/handovers${listQuery(options)}`),
  pendingHandovers: async () => {
    const options = { state: ["DRAFT", "SUBMITTED", "ACCEPTED"], page_size: 100 };
    const first = await request<ListPage<ShiftHandover>>(`/handovers${listQuery(options)}`);
    return fetchAll(first, (cursor) =>
      request<ListPage<ShiftHandover>>(`/handovers${listQuery({ ...options, cursor })}`)
    );
  },
  handover: (id: string) => request<ShiftHandover>(`/handovers/${id}`),
  submitHandover: (id: string) => command<ShiftHandover>(`/handovers/${id}/submit`, {}),
  acceptHandover: (id: string) => command<ShiftHandover>(`/handovers/${id}/accept`, {}),
  acknowledgeHandover: (id: string) =>
    command<ShiftHandover>(`/handovers/${id}/acknowledge`, {})
};
