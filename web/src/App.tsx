import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { api } from "./api";
import { relativeTime, severityTone } from "./presentation";
import { useI18n } from "./i18n/I18nProvider";
import type { Locale, MessageKey } from "./i18n/messages";
import { ConsoleLayout } from "./console/layout/ConsoleLayout";
import { PlaceholderPage } from "./console/pages/PlaceholderPage";
import type {
  Asset,
  Demonstration,
  Evidence,
  EvaluationSummary,
  Execution,
  Incident,
  KnowledgeDocument,
  MaintenanceReport,
  Principal,
  Recommendation,
  ShiftHandover,
  Step,
  WorkflowVersion
} from "./types";

type View =
  | "overview"
  | "assets"
  | "incidents"
  | "knowledge"
  | "handover"
  | "demonstrations"
  | "workflows"
  | "quality";

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<ConsoleLayout />}>
          <Route path="/" element={<PlaceholderPage titleKey="nav.overview" />} />
          <Route path="/assets" element={<PlaceholderPage titleKey="nav.assets" />} />
          <Route path="/assets/:assetId" element={<PlaceholderPage titleKey="nav.assets" />} />
          <Route path="/incidents" element={<PlaceholderPage titleKey="nav.incidents" />} />
          <Route path="/incidents/:incidentId" element={<PlaceholderPage titleKey="nav.incidents" />} />
          <Route path="/executions/:executionId" element={<PlaceholderPage titleKey="nav.executions" />} />
          <Route path="/reports" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/reports/:reportId" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/handovers" element={<PlaceholderPage titleKey="nav.handover" />} />
          <Route path="/knowledge" element={<PlaceholderPage titleKey="nav.knowledge" />} />
          <Route path="/knowledge/:documentId" element={<PlaceholderPage titleKey="nav.knowledge" />} />
          <Route path="/search" element={<PlaceholderPage titleKey="nav.search" />} />
          <Route path="/quality" element={<PlaceholderPage titleKey="nav.quality" />} />
          <Route path="/demonstrations" element={<PlaceholderPage titleKey="nav.demonstrations" />} />
          <Route path="/workflows" element={<PlaceholderPage titleKey="nav.workflows" />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function NavItem(props: { active: boolean; labelKey: MessageKey; onClick: () => void }) {
  const { t } = useI18n();
  return (
    <button className={props.active ? "nav-item active" : "nav-item"} onClick={props.onClick}>
      <span className="nav-indicator" />
      {t(props.labelKey)}
    </button>
  );
}

function signOut() {
  // Submit a POST form so the browser follows the full 303 → Keycloak → SPA
  // redirect chain natively. Using fetch here would force it through a
  // cross-origin redirect that Keycloak does not CORS-allow, leaving the page
  // frozen on the SPA for several seconds before the catch block could fall
  // back to /auth/login.
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/auth/logout";
  form.style.display = "none";
  document.body.appendChild(form);
  form.submit();
}

const viewTitle: Record<View, MessageKey> = {
  overview: "pageTitle.overview",
  assets: "pageTitle.assets",
  incidents: "pageTitle.incidents",
  knowledge: "pageTitle.knowledge",
  handover: "pageTitle.handover",
  demonstrations: "pageTitle.demonstrations",
  workflows: "pageTitle.workflows",
  quality: "pageTitle.quality"
};

function Overview(props: {
  assets: Asset[];
  incidents: Incident[];
  openCount: number;
  criticalAssetCount: number;
  onOpenIncidents: () => void;
}) {
  const { t } = useI18n();
  return (
    <>
      <section className="metrics" aria-label={t("overview.operationalStatus")}>
        <Metric label={t("overview.registeredAssets")} value={props.assets.length} detail={t("overview.nativeExternal")} />
        <Metric label={t("overview.openIncidents")} value={props.openCount} detail={t("overview.requireAttention")} accent />
        <Metric label={t("overview.criticalAssets")} value={props.criticalAssetCount} detail={t("overview.criticalityA")} />
        <Metric label={t("overview.unsafeAiActions")} value={0} detail={t("overview.safetyBoundaryEnforced")} safe />
      </section>
      <section className="panel">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">{t("overview.priorityQueue")}</span>
            <h2>{t("overview.activeIncidents")}</h2>
          </div>
          <button className="secondary-button" onClick={props.onOpenIncidents}>{t("overview.openWorkbench")}</button>
        </div>
        <IncidentTable incidents={props.incidents.slice(0, 8)} />
      </section>
    </>
  );
}

function Metric(props: { label: string; value: number; detail: string; accent?: boolean; safe?: boolean }) {
  return (
    <article className={`metric ${props.accent ? "metric-accent" : ""} ${props.safe ? "metric-safe" : ""}`}>
      <span>{props.label}</span>
      <strong>{props.value.toString().padStart(2, "0")}</strong>
      <small>{props.detail}</small>
    </article>
  );
}

function AssetsTable({ assets, onCreate }: { assets: Asset[]; onCreate: () => void }) {
  const { t } = useI18n();
  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("assets.sourceAwareRegistry")}</span>
          <h2>{t("assets.title")}</h2>
        </div>
        <div className="heading-actions"><span className="count">{t("assets.count", { count: assets.length })}</span><button className="secondary-button" onClick={onCreate}>{t("assets.add")}</button></div>
      </div>
      <div className="table-wrap">
        <table>
          <thead><tr><th>{t("assets.tag")}</th><th>{t("assets.asset")}</th><th>{t("assets.class")}</th><th>{t("assets.criticality")}</th><th>{t("assets.authority")}</th><th>{t("assets.status")}</th></tr></thead>
          <tbody>
            {assets.map((asset) => (
              <tr key={asset.id}>
                <td className="mono strong">{asset.tag}</td>
                <td><strong>{asset.name}</strong><small>{[asset.manufacturer, asset.model].filter(Boolean).join(" · ") || t("assets.noOem")}</small></td>
                <td>{asset.class}</td>
                <td><span className={`criticality rating-${asset.criticality?.rating ?? "none"}`}>{asset.criticality?.rating ?? "—"}</span></td>
                <td><span className="source-badge">{asset.source_of_truth === "EXTERNAL_REFERENCE" ? t("assets.externalProjection") : t("assets.skawldNative")}</span></td>
                <td><span className="status-dot" />{asset.status}</td>
              </tr>
            ))}
            {assets.length === 0 && <EmptyRow columns={6} labelKey="assets.empty" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function IncidentTable({ incidents }: { incidents: Incident[] }) {
  const { t, locale } = useI18n();
  return (
    <div className="table-wrap">
      <table>
        <thead><tr><th>{t("incidents.incident")}</th><th>{t("assets.asset")}</th><th>{t("incidents.summary")}</th><th>{t("incidents.severity")}</th><th>{t("incidents.state")}</th><th>{t("incidents.detected")}</th></tr></thead>
        <tbody>
          {incidents.map((incident) => (
            <tr key={incident.id}>
              <td className="mono strong">{incident.number}</td>
              <td className="mono">{incident.asset_tag || "—"}</td>
              <td className="summary-cell">{incident.summary}</td>
              <td><span className={`severity ${severityTone(incident.severity)}`}>{incident.severity}</span></td>
              <td>{incident.state.replace("_", " ")}</td>
              <td>{relativeTime(incident.detected_at, locale)}</td>
            </tr>
          ))}
          {incidents.length === 0 && <EmptyRow columns={6} labelKey="incidents.empty" />}
        </tbody>
      </table>
    </div>
  );
}

function IncidentQueue(props: {
  incidents: Incident[];
  selected?: Incident;
  onSelect: (incident: Incident) => void;
  onCreate: () => void;
}) {
  const { t, locale } = useI18n();
  return (
    <section className="queue panel">
      <div className="panel-heading">
        <div><span className="eyebrow">{t("incidents.authorizedScope")}</span><h2>{t("incidents.queue")}</h2></div>
        <div className="heading-actions"><span className="count">{props.incidents.length}</span><button className="secondary-button" onClick={props.onCreate}>{t("incidents.new")}</button></div>
      </div>
      <div className="queue-list">
        {props.incidents.map((incident) => (
          <button
            key={incident.id}
            className={props.selected?.id === incident.id ? "queue-item selected" : "queue-item"}
            onClick={() => props.onSelect(incident)}
          >
            <span className={`severity-bar ${severityTone(incident.severity)}`} />
            <span>
              <span className="queue-meta"><span className="mono">{incident.number}</span><span>{relativeTime(incident.detected_at, locale)}</span></span>
              <strong>{incident.asset_tag} · {incident.summary}</strong>
              <small>{incident.state.replace("_", " ")} · {incident.severity}</small>
            </span>
          </button>
        ))}
        {props.incidents.length === 0 && <div className="empty">{t("incidents.none")}</div>}
      </div>
    </section>
  );
}

function ExecutionPanel(props: {
  incident?: Incident;
  execution?: Execution;
  completedSteps: number;
  busy: boolean;
  onCreate: () => Promise<void>;
  onStart: () => Promise<void>;
  onCompleteStep: (step: Step) => Promise<void>;
  onMeasurement: (type: string, value: string, unit: string) => Promise<void>;
  recommendation?: Recommendation;
  report?: MaintenanceReport;
  onRecommend: () => Promise<void>;
  onReviewRecommendation: (
    outcome:
      | "ACCEPTED"
      | "REJECTED"
      | "CORRECTED"
      | "UNSAFE"
      | "UNSUPPORTED"
      | "INCORRECT_NEXT_STEP",
    reason: string,
    materialClaims: number,
    supportedClaims: number,
    retrievedEvidence: number,
    relevantEvidence: number
  ) => Promise<void>;
  onDraftReport: () => Promise<void>;
  onCapture: () => Promise<void>;
  onEvidenceView: (evidenceID: string) => Promise<void>;
}) {
  const { t } = useI18n();
  if (!props.incident) {
    return <section className="execution panel empty-state"><strong>{t("execution.selectIncident")}</strong><p>{t("execution.reviewContext")}</p></section>;
  }
  if (!props.execution) {
    return (
      <section className="execution panel empty-state">
        <span className={`severity ${severityTone(props.incident.severity)}`}>{props.incident.severity}</span>
        <strong>{props.incident.asset_tag} · {props.incident.summary}</strong>
        <p>{t("execution.cmmsDisclaimer")}</p>
        <button className="primary-button" disabled={props.busy || props.incident.state === "RESOLVED"} onClick={() => void props.onCreate()}>
          {t("execution.createPumpInspection")}
        </button>
      </section>
    );
  }
  const execution = props.execution;
  const progress = Math.round((props.completedSteps / execution.steps.length) * 100);
  return (
    <section className="execution panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("execution.deterministicExecution")}</span>
          <h2>{t("execution.inspection", { tag: props.execution.asset_tag })}</h2>
        </div>
        <span className="state-badge">{props.execution.state.replace("_", " ")}</span>
      </div>
      <div className="progress-row"><span style={{ width: `${progress}%` }} /><small>{t("execution.stepsProgress", { done: props.completedSteps, total: props.execution.steps.length })}</small></div>
      {props.execution.state === "ASSIGNED" && (
        <button className="primary-button full" disabled={props.busy} onClick={() => void props.onStart()}>{t("execution.startInspection")}</button>
      )}
      <div className="steps">
        {props.execution.steps.map((step) => (
          <article key={step.id} className={`step step-${step.state.toLowerCase()}`}>
            <span className="step-number">{step.sequence}</span>
            <div>
              <strong>{step.title}</strong>
              <small>{step.risk_level.replaceAll("_", " ")}</small>
              {step.required_prerequisite && <span className="prerequisite">{t("execution.requiresVerified", { prerequisite: step.required_prerequisite.replaceAll("_", " ") })}</span>}
              {step.blocked_reason && <p className="blocked-reason">{step.blocked_reason}</p>}
            </div>
            <button
              className="step-action"
              disabled={props.busy || execution.state !== "IN_PROGRESS" || step.state === "COMPLETED"}
              onClick={() => void props.onCompleteStep(step)}
            >
              {step.state === "COMPLETED" ? t("execution.done") : step.state === "BLOCKED" ? t("execution.retry") : t("execution.complete")}
            </button>
          </article>
        ))}
      </div>
      {props.execution.state === "IN_PROGRESS" && (
        <MeasurementForm onSubmit={props.onMeasurement} busy={props.busy} />
      )}
      {props.execution.measurements.length > 0 && (
        <div className="evidence">
          <span className="eyebrow">{t("execution.recordedEvidence")}</span>
          {props.execution.measurements.map((measurement) => (
            <span key={measurement.id}><strong>{measurement.value}</strong> {measurement.unit.replaceAll("_", "/")} · {measurement.measurement_type.replaceAll("_", " ")}</span>
          ))}
        </div>
      )}
      <div className="copilot-actions">
        <button className="primary-button" disabled={props.busy} onClick={() => void props.onCapture()}>
          {t("execution.startSemanticDemo")}
        </button>
        <button className="secondary-button" disabled={props.busy} onClick={() => void props.onRecommend()}>
          {t("execution.generateRecommendation")}
        </button>
        <button className="secondary-button" disabled={props.busy} onClick={() => void props.onDraftReport()}>
          {t("execution.prepareReportDraft")}
        </button>
      </div>
      {props.recommendation && (
        <RecommendationCard
          key={props.recommendation.id}
          value={props.recommendation}
          onEvidenceView={props.onEvidenceView}
          onReview={props.onReviewRecommendation}
          busy={props.busy}
        />
      )}
      {props.report && <ReportCard value={props.report} />}
    </section>
  );
}

function RecommendationCard(props: {
  value: Recommendation;
  onEvidenceView: (evidenceID: string) => Promise<void>;
  onReview: (
    outcome:
      | "ACCEPTED"
      | "REJECTED"
      | "CORRECTED"
      | "UNSAFE"
      | "UNSUPPORTED"
      | "INCORRECT_NEXT_STEP",
    reason: string,
    materialClaims: number,
    supportedClaims: number,
    retrievedEvidence: number,
    relevantEvidence: number
  ) => Promise<void>;
  busy: boolean;
}) {
  const { t } = useI18n();
  const value = props.value;
  const [outcome, setOutcome] = useState<
    "ACCEPTED" | "REJECTED" | "CORRECTED" | "UNSAFE" | "UNSUPPORTED" | "INCORRECT_NEXT_STEP"
  >("ACCEPTED");
  const [reason, setReason] = useState(t("recommendation.defaultReason"));
  const [materialClaims, setMaterialClaims] = useState(
    value.output.status === "RECOMMENDATION" ? 1 : 0
  );
  const [supportedClaims, setSupportedClaims] = useState(materialClaims);
  const [retrievedEvidence, setRetrievedEvidence] = useState(value.evidence.length);
  const [relevantEvidence, setRelevantEvidence] = useState(value.output.evidence_ids.length);
  const countsInvalid =
    materialClaims < 0 ||
    supportedClaims < 0 ||
    supportedClaims > materialClaims ||
    retrievedEvidence < 0 ||
    relevantEvidence < 0 ||
    relevantEvidence > retrievedEvidence;
  return (
    <article className="ai-result">
      <div className="result-header">
        <div><span className="eyebrow">{t("recommendation.advisoryProposal")}</span><h2>{value.output.status.replaceAll("_", " ")}</h2></div>
        <span className="state-badge">{Math.round(value.output.confidence * 100)}% · {value.output.risk_level}</span>
      </div>
      <p>{value.output.recommendation || t("recommendation.insufficient")}</p>
      <EvidenceLinks
        evidence={value.evidence}
        selected={value.output.evidence_ids}
        onView={props.onEvidenceView}
      />
      <small>{t("recommendation.unknowns", { value: value.output.unknowns.join(" · ") || t("recommendation.noneStated") })} · {t("recommendation.humanConfirmation")}</small>
      <small className="provenance">{value.provider}/{value.model} · {value.prompt_version}</small>
      <div className="quality-review">
        <span className="eyebrow">{t("recommendation.pilotQualityLabel")}</span>
        <select value={outcome} onChange={(event) => setOutcome(event.target.value as typeof outcome)}>
          <option value="ACCEPTED">{t("recommendation.accepted")}</option>
          <option value="REJECTED">{t("recommendation.rejected")}</option>
          <option value="UNSAFE">{t("recommendation.unsafe")}</option>
          <option value="UNSUPPORTED">{t("recommendation.unsupportedClaim")}</option>
          <option value="INCORRECT_NEXT_STEP">{t("recommendation.incorrectNextStep")}</option>
        </select>
        <input value={reason} onChange={(event) => setReason(event.target.value)} aria-label={t("recommendation.reason")} />
        <div className="quality-counts">
          <label>
            {t("recommendation.materialClaims")}
            <input
              type="number"
              min="0"
              value={materialClaims}
              onChange={(event) => setMaterialClaims(Number(event.target.value))}
            />
          </label>
          <label>
            {t("recommendation.supportedClaims")}
            <input
              type="number"
              min="0"
              max={materialClaims}
              value={supportedClaims}
              onChange={(event) => setSupportedClaims(Number(event.target.value))}
            />
          </label>
          <label>
            {t("recommendation.retrievedEvidence")}
            <input
              type="number"
              min="0"
              value={retrievedEvidence}
              onChange={(event) => setRetrievedEvidence(Number(event.target.value))}
            />
          </label>
          <label>
            {t("recommendation.relevantEvidence")}
            <input
              type="number"
              min="0"
              max={retrievedEvidence}
              value={relevantEvidence}
              onChange={(event) => setRelevantEvidence(Number(event.target.value))}
            />
          </label>
        </div>
        <button
          className="secondary-button"
          disabled={props.busy || reason.trim() === "" || countsInvalid}
          onClick={() =>
            void props.onReview(
              outcome,
              reason,
              materialClaims,
              supportedClaims,
              retrievedEvidence,
              relevantEvidence
            )
          }
        >
          {t("recommendation.recordReview")}
        </button>
      </div>
    </article>
  );
}

function QualityPanel(props: {
  value?: EvaluationSummary;
  busy: boolean;
  onRefresh: () => Promise<void>;
}) {
  const { t } = useI18n();
  if (!props.value) {
    return <section className="panel empty-state"><strong>{t("quality.noSummary")}</strong></section>;
  }
  const percent = (value: number) => `${Math.round(value * 100)}%`;
  return (
    <>
      <section className="metrics" aria-label={t("quality.pilotEvaluation")}>
        <Metric label={t("quality.reviewCoverage")} value={Math.round(props.value.review_coverage * 100)} detail={t("quality.reviewedDetail", { reviewed: props.value.reviewed, total: props.value.recommendations })} />
        <Metric label={t("quality.evidenceCoverage")} value={Math.round(props.value.evidence_coverage * 100)} detail={t("quality.labeledSupported")} />
        <Metric label={t("quality.unsafeRate")} value={Math.round(props.value.unsafe_recommendation_rate * 100)} detail={t("quality.mustRemainZero")} accent={props.value.unsafe_recommendation_rate > 0} safe={props.value.unsafe_recommendation_rate === 0} />
        <Metric label={t("quality.workflowGates")} value={Math.round(props.value.workflow_gate_pass_rate * 100)} detail={t("quality.evaluatedVersions", { count: props.value.workflow_evaluations })} />
      </section>
      <section className="panel quality-summary">
        <div className="panel-heading">
          <div><span className="eyebrow">{t("quality.pilotEvaluation")}</span><h2>{t("quality.humanReviewedSignals")}</h2></div>
          <button className="secondary-button" disabled={props.busy} onClick={() => void props.onRefresh()}>{t("quality.refresh")}</button>
        </div>
        <dl>
          <div><dt>{t("quality.acceptance")}</dt><dd>{percent(props.value.recommendation_acceptance)}</dd></div>
          <div><dt>{t("quality.humanOverride")}</dt><dd>{percent(props.value.human_override_rate)}</dd></div>
          <div><dt>{t("quality.unsupported")}</dt><dd>{percent(props.value.unsupported_recommendation_rate)}</dd></div>
          <div><dt>{t("quality.incorrectNextStep")}</dt><dd>{percent(props.value.incorrect_next_step_rate)}</dd></div>
          <div><dt>{t("quality.retrievalPrecision")}</dt><dd>{percent(props.value.retrieval_precision)}</dd></div>
          <div><dt>{t("quality.llmCalls")}</dt><dd>{props.value.llm_calls}</dd></div>
          <div><dt>{t("quality.avgLatency")}</dt><dd>{Math.round(props.value.average_latency_ms)} ms</dd></div>
          <div><dt>{t("quality.tokens")}</dt><dd>{props.value.tokens_in + props.value.tokens_out}</dd></div>
          <div><dt>{t("quality.estimatedCost")}</dt><dd>{props.value.estimated_cost_micros} μ</dd></div>
        </dl>
        <p className="muted">{t("quality.muted")}</p>
      </section>
    </>
  );
}

function ReportCard({ value }: { value: MaintenanceReport }) {
  const { t } = useI18n();
  return (
    <article className="ai-result">
      <div className="result-header">
        <div><span className="eyebrow">{t("report.humanReviewableDraft")}</span><h2>{t("report.title", { revision: value.revision })}</h2></div>
        <span className="state-badge">{value.state}</span>
      </div>
      <p>{value.structured_content.summary}</p>
      <EvidenceLinks evidence={value.evidence} selected={value.structured_content.evidence_ids} />
      <small>{t("recommendation.unknowns", { value: value.structured_content.unknowns.join(" · ") })}</small>
      <small className="provenance">{value.provider}/{value.model} · {value.prompt_version}</small>
    </article>
  );
}

function EvidenceLinks(props: {
  evidence: Evidence[];
  selected: string[];
  onView?: (evidenceID: string) => Promise<void>;
}) {
  const { evidence, selected } = props;
  const items = evidence.filter((item) => selected.includes(item.id));
  const [viewed, setViewed] = useState<Set<string>>(() => new Set());
  return (
    <div className="evidence-list">
      {items.map((item) => (
        <details
          key={item.id}
          onToggle={(event) => {
            if (
              event.currentTarget.open &&
              props.onView &&
              !viewed.has(item.id)
            ) {
              const next = new Set(viewed);
              next.add(item.id);
              setViewed(next);
              void props.onView(item.id);
            }
          }}
        >
          <summary>{item.title} · {item.locator} · {item.authority}</summary>
          <p>{item.content}</p>
          <small className="mono">{item.id}</small>
        </details>
      ))}
    </div>
  );
}

function KnowledgePanel(props: {
  siteID?: string;
  documents: KnowledgeDocument[];
  busy: boolean;
  onRefresh: () => Promise<void>;
  onMutate: (action: () => Promise<void>) => Promise<void>;
}) {
  const { t } = useI18n();
  const [title, setTitle] = useState("P-302 High Vibration Inspection");
  const [assetClass, setAssetClass] = useState("CENTRIFUGAL_PUMP");
  const [file, setFile] = useState<File>();
  const [query, setQuery] = useState("high vibration lubrication bearing");
  const [results, setResults] = useState<Evidence[]>([]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!props.siteID || !file) return;
    void props.onMutate(async () => {
      const document = await api.createDocument(props.siteID!, title, "SOP");
      const revision = await api.createDocumentRevision(document.id, props.siteID!, assetClass);
      await api.uploadDocumentRevision(revision.id, props.siteID!, file);
      await props.onRefresh();
    });
  }

  return (
    <div className="knowledge-layout">
      <form className="panel knowledge-form" onSubmit={submit}>
        <div className="panel-heading"><div><span className="eyebrow">{t("knowledge.controlledIngestion")}</span><h2>{t("knowledge.addRevision")}</h2></div></div>
        <label>{t("knowledge.title")}<input value={title} onChange={(event) => setTitle(event.target.value)} /></label>
        <label>{t("knowledge.assetClass")}<input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} /></label>
        <label>{t("knowledge.pdfOrText")}<input type="file" accept=".pdf,text/plain,application/pdf" onChange={(event) => setFile(event.target.files?.[0])} /></label>
        <button className="primary-button" disabled={props.busy || !props.siteID || !file}>{t("knowledge.uploadQueue")}</button>
        <p className="form-note">{t("knowledge.formNote")}</p>
      </form>
      <section className="panel">
        <div className="panel-heading"><div><span className="eyebrow">{t("knowledge.validityAware")}</span><h2>{t("knowledge.documentRevisions")}</h2></div><span className="count">{props.documents.length}</span></div>
        <div className="document-list">
          {props.documents.map((document) => (
            <article key={document.id}>
              <strong>{document.title}</strong><small>{document.document_type} · {document.authority}</small>
              {document.revisions.map((revision) => (
                <span key={revision.id} className="revision-row">
                  <span className="mono">{revision.revision}</span>
                  <span>{revision.approval_status}</span>
                  <span>{revision.ingestion_state}</span>
                </span>
              ))}
            </article>
          ))}
          {props.documents.length === 0 && <div className="empty">{t("knowledge.noDocuments")}</div>}
        </div>
      </section>
      <section className="panel search-panel">
        <div className="panel-heading"><div><span className="eyebrow">{t("knowledge.authorizationFirstRrf")}</span><h2>{t("knowledge.evidenceSearch")}</h2></div></div>
        <div className="search-row"><input value={query} onChange={(event) => setQuery(event.target.value)} /><button className="secondary-button" disabled={!props.siteID || props.busy} onClick={() => void props.onMutate(async () => setResults((await api.searchKnowledge(props.siteID!, query)).items))}>{t("knowledge.search")}</button></div>
        <EvidenceLinks evidence={results} selected={results.map((item) => item.id)} />
      </section>
    </div>
  );
}

function HandoverPanel(props: {
  siteID?: string;
  handover?: ShiftHandover;
  busy: boolean;
  onPrepare: () => Promise<void>;
  onCapture: () => Promise<void>;
}) {
  const { t } = useI18n();
  return (
    <section className="panel handover-panel">
      <div className="panel-heading">
        <div><span className="eyebrow">{t("handover.normalWorkIntelligence")}</span><h2>{t("handover.draft")}</h2></div>
        <div className="heading-actions">
          {props.handover && (
            <button className="secondary-button" disabled={props.busy} onClick={() => void props.onCapture()}>
              {t("handover.startDemonstration")}
            </button>
          )}
          <button className="primary-button" disabled={!props.siteID || props.busy} onClick={() => void props.onPrepare()}>{t("handover.prepare")}</button>
        </div>
      </div>
      {!props.handover ? (
        <div className="empty">{t("handover.empty")}</div>
      ) : (
        <div className="handover-content">
          <div className="result-header"><p>{props.handover.structured_content.summary}</p><span className="state-badge">{props.handover.state}</span></div>
          <HandoverSection title={t("handover.openIncidents")} items={props.handover.structured_content.open_incidents} empty={t("handover.none")} />
          <HandoverSection title={t("handover.activeExecutions")} items={props.handover.structured_content.active_executions} empty={t("handover.none")} />
          <HandoverSection title={t("handover.safetyConcerns")} items={props.handover.structured_content.safety_concerns} empty={t("handover.none")} />
          <HandoverSection title={t("handover.followUp")} items={props.handover.structured_content.follow_up} empty={t("handover.none")} />
          <EvidenceLinks evidence={props.handover.evidence} selected={props.handover.structured_content.evidence_ids} />
          <small className="provenance">{props.handover.provider}/{props.handover.model} · {props.handover.prompt_version} · {t("handover.humanAcceptance")}</small>
        </div>
      )}
    </section>
  );
}

function HandoverSection({ title, items, empty }: { title: string; items: string[]; empty: string }) {
  return <div className="handover-section"><strong>{title}</strong>{items.length ? <ul>{items.map((item, index) => <li key={`${title}-${index}`}>{item}</li>)}</ul> : <p>{empty}</p>}</div>;
}

function DemonstrationPanel(props: {
  values: Demonstration[];
  selected?: Demonstration;
  busy: boolean;
  onSelect: (value: Demonstration) => Promise<void>;
  onComplete: (value: Demonstration, outcome: string) => Promise<void>;
  onRedact: (
    value: Demonstration,
    eventID: string,
    path: string,
    reason: string
  ) => Promise<void>;
  onReview: (
    value: Demonstration,
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED",
    reason: string
  ) => Promise<void>;
}) {
  const { t, locale } = useI18n();
  const selected = props.selected;

  function complete() {
    if (!selected) return;
    const outcome = window.prompt(
      t("demo.outcomePrompt"),
      t("demo.outcomeDefault")
    );
    if (outcome?.trim()) void props.onComplete(selected, outcome.trim());
  }

  function review(
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED"
  ) {
    if (!selected) return;
    const reason = window.prompt(
      t("demo.reviewReasonPrompt"),
      decision === "APPROVED"
        ? t("demo.reviewApproveDefault")
        : t("demo.reviewRejectDefault")
    );
    if (reason?.trim()) void props.onReview(selected, decision, reason.trim());
  }

  function redact(eventID: string) {
    if (!selected) return;
    const path = window.prompt(
      t("demo.jsonPathPrompt"),
      t("demo.jsonPathDefault")
    );
    if (!path?.trim()) return;
    const reason = window.prompt(
      t("demo.redactReasonPrompt"),
      t("demo.redactReasonDefault")
    );
    if (reason?.trim()) {
      void props.onRedact(selected, eventID, path.trim(), reason.trim());
    }
  }

  return (
    <div className="demonstration-layout">
      <section className="panel demonstration-list">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">{t("demo.organizationalMemory")}</span>
            <h2>{t("demo.semanticDemonstrations")}</h2>
          </div>
          <span className="count">{props.values.length}</span>
        </div>
        {props.values.map((value) => (
          <button
            key={value.id}
            className={
              selected?.id === value.id
                ? "demonstration-item selected"
                : "demonstration-item"
            }
            onClick={() => void props.onSelect(value)}
          >
            <span>
              <strong>{value.workflow_key}</strong>
              <small>
                {value.subject_kind} · {t("demo.semanticEvents", { count: value.events.length })}
              </small>
            </span>
            <span className="state-badge">{value.status}</span>
            <small>
              {t("demo.review", { status: value.review_status })} · {relativeTime(value.started_at, locale)}
            </small>
          </button>
        ))}
        {props.values.length === 0 && (
          <div className="empty">
            {t("demo.startCapture")}
          </div>
        )}
      </section>

      <section className="panel demonstration-timeline">
        {!selected ? (
          <div className="empty-state">
            <strong>{t("demo.select")}</strong>
            <p>
              {t("demo.reviewGuidance")}
            </p>
          </div>
        ) : (
          <>
            <div className="panel-heading">
              <div>
                <span className="eyebrow">SDK Observation v{selected.schema_version}</span>
                <h2>{selected.workflow_key}</h2>
              </div>
              <div className="heading-actions">
                <span className="state-badge">{selected.review_status}</span>
                {selected.status === "recording" && (
                  <button
                    className="primary-button"
                    disabled={props.busy}
                    onClick={complete}
                  >
                    {t("demo.completeCapture")}
                  </button>
                )}
              </div>
            </div>
            <div className="capture-health">
              <span>
                <strong>{selected.capture.applied}</strong> {t("demo.applied")}
              </span>
              <span>
                <strong>{selected.capture.pending}</strong> {t("demo.pending")}
              </span>
              <span className={selected.capture.failed ? "capture-failed" : ""}>
                <strong>{selected.capture.failed}</strong> {t("demo.failed")}
              </span>
              <small className="mono">{t("demo.session", { id: selected.session_id })}</small>
            </div>
            {selected.capture.last_error && (
              <div className="notice">{selected.capture.last_error}</div>
            )}
            <div className="semantic-timeline">
              {selected.events.map((event) => (
                <article
                  key={event.id}
                  className={
                    event.correction_of
                      ? "semantic-event correction-event"
                      : "semantic-event"
                  }
                >
                  <span className="timeline-ordinal">{event.ordinal}</span>
                  <div>
                    <div className="event-heading">
                      <strong>{event.action}</strong>
                      <span className="source-badge">
                        {event.trust.replaceAll("_", " ")}
                      </span>
                    </div>
                    {event.intent && <p>{event.intent}</p>}
                    <small>
                      {event.entity?.type ?? t("demo.eventFallback")} ·{" "}
                      <span className="mono">{event.entity?.id ?? event.id}</span>
                    </small>
                    <small>
                      {t("demo.actor")} <span className="mono">{event.actor_id}</span> ·{" "}
                      {new Date(event.timestamp).toLocaleString()}
                    </small>
                    {event.correction_of && (
                      <span className="correction-link">
                        {t("demo.correctsEvent", { id: event.correction_of })}
                      </span>
                    )}
                    <details>
                      <summary>{t("demo.structuredPayload")}</summary>
                      <pre>
                        {JSON.stringify(
                          {
                            output: event.output,
                            decision: event.decision,
                            result: event.result,
                            context: event.context
                          },
                          null,
                          2
                        )}
                      </pre>
                    </details>
                    <div className="event-provenance">
                      <span>{event.source}</span>
                      <span>{event.sensitivity}</span>
                      <span className="mono">
                        {event.domain_event_id
                          ? t("demo.domain", { id: event.domain_event_id })
                          : t("demo.manualCapture")}
                      </span>
                      <button
                        className="secondary-button"
                        disabled={props.busy}
                        onClick={() => redact(event.id)}
                      >
                        {t("demo.redactField")}
                      </button>
                    </div>
                  </div>
                </article>
              ))}
            </div>
            {selected.status === "completed" && (
              <div className="review-actions">
                <span>
                  <strong>{t("demo.humanGovernance")}</strong>
                  <small>
                    {t("demo.governanceNote")}
                  </small>
                </span>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => review("REDACTION_REQUIRED")}
                >
                  {t("demo.requestRedaction")}
                </button>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => review("REJECTED")}
                >
                  {t("demo.reject")}
                </button>
                <button
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() => review("APPROVED")}
                >
                  {t("demo.approveTrace")}
                </button>
              </div>
            )}
          </>
        )}
      </section>
    </div>
  );
}

function WorkflowLearningPanel(props: {
  values: WorkflowVersion[];
  demonstrations: Demonstration[];
  selected?: WorkflowVersion;
  busy: boolean;
  onSelect: (value: WorkflowVersion) => void;
  onCompile: (demonstrationIDs: string[]) => Promise<void>;
  onReview: (
    value: WorkflowVersion,
    decision: "APPROVED" | "REJECTED" | "REVIEW_REQUIRED",
    reason: string
  ) => Promise<void>;
  onPublish: (value: WorkflowVersion, reason: string) => Promise<void>;
  onRetire: (value: WorkflowVersion, reason: string) => Promise<void>;
}) {
  const { t } = useI18n();
  const [selectedDemonstrations, setSelectedDemonstrations] = useState<string[]>([]);
  const reviewed = props.demonstrations.filter(
    (value) => value.status === "completed" && value.review_status === "APPROVED"
  );
  const selected = props.selected;

  function toggleDemonstration(id: string) {
    setSelectedDemonstrations((current) =>
      current.includes(id)
        ? current.filter((value) => value !== id)
        : [...current, id]
    );
  }

  function reason(
    promptKey: MessageKey,
    defaultKey: MessageKey
  ): string | undefined {
    const value = window.prompt(t(promptKey), t(defaultKey))?.trim();
    return value || undefined;
  }

  return (
    <div className="workbench workflow-workbench">
      <section className="panel queue-panel">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">{t("workflow.reviewedEvidenceOnly")}</span>
            <h2>{t("workflow.compileCandidate")}</h2>
          </div>
          <span className="count">{t("workflow.traces", { count: reviewed.length })}</span>
        </div>
        <p className="muted">
          {t("workflow.compileHint")}
        </p>
        <div className="selection-list">
          {reviewed.map((value) => (
            <label className="selection-row" key={value.id}>
              <input
                type="checkbox"
                checked={selectedDemonstrations.includes(value.id)}
                onChange={() => toggleDemonstration(value.id)}
              />
              <span>
                <strong>{value.workflow_key}</strong>
                <small>
                  {value.subject_kind} · {t("demo.semanticEvents", { count: value.events.length })}
                </small>
              </span>
            </label>
          ))}
        </div>
        <button
          className="primary-button"
          disabled={props.busy || selectedDemonstrations.length < 2}
          onClick={() => void props.onCompile(selectedDemonstrations)}
        >
          {t("workflow.compileButton")}
        </button>
        <div className="candidate-list">
          {props.values.map((value) => (
            <button
              className={
                selected?.workflow_id === value.workflow_id &&
                selected.version === value.version
                  ? "queue-item selected"
                  : "queue-item"
              }
              key={`${value.workflow_id}:${value.version}`}
              onClick={() => props.onSelect(value)}
            >
              <span>
                <strong>{value.name}</strong>
                <small>
                  v{value.version} · {value.asset_class}
                </small>
              </span>
              <span className="state-badge">
                {value.status}
              </span>
            </button>
          ))}
        </div>
      </section>

      <section className="panel detail-panel workflow-review">
        {!selected && (
          <div className="empty-state">
            <strong>{t("workflow.select")}</strong>
            <p>{t("workflow.selectGuidance")}</p>
          </div>
        )}
        {selected && (
          <>
            <div className="panel-heading">
              <div>
                <span className="eyebrow">{t("workflow.humanControlled")}</span>
                <h2>{selected.name}</h2>
                <small className="mono">
                  {selected.workflow_key} · v{selected.version}
                </small>
              </div>
              <span className="state-badge">
                {selected.status}
              </span>
            </div>

            <div className="workflow-facts">
              <span>
                <strong>
                  {Math.round(
                    (selected.analysis.sequence_consistency ?? 0) * 100
                  )}%
                </strong>
                {t("workflow.sequenceConsistency")}
              </span>
              <span>
                <strong>{selected.analysis.conflicts?.length ?? 0}</strong>
                {t("workflow.ambiguousTransitions")}
              </span>
              <span>
                <strong>{selected.source_demonstration_ids.length}</strong>
                {t("workflow.sourceDemos")}
              </span>
              <span>
                <strong>{selected.improvement_candidates.length}</strong>
                {t("workflow.correctionCandidates")}
              </span>
            </div>

            <div className="workflow-steps">
              {selected.steps.map((step, index) => (
                <article className="workflow-step" key={step.id}>
                  <span className="timeline-ordinal">{index + 1}</span>
                  <div>
                    <strong>{step.name || step.id}</strong>
                    <small className="mono">{step.tool_name || step.kind}</small>
                    <p>
                      {t("workflow.eventRefs", {
                        count: step.evidence.reduce(
                          (total, value) => total + value.event_ids.length,
                          0
                        ),
                        traces: step.evidence.length
                      })}
                    </p>
                    <details>
                      <summary>{t("workflow.evidenceIdentities")}</summary>
                      <pre>{JSON.stringify(step.evidence, null, 2)}</pre>
                    </details>
                  </div>
                </article>
              ))}
            </div>

            <div className="workflow-review-grid">
              <article>
                <span className="eyebrow">{t("workflow.behavioralDiff")}</span>
                <pre>{JSON.stringify(selected.behavioral_changes, null, 2)}</pre>
              </article>
              <article>
                <span className="eyebrow">{t("workflow.applicabilityValidity")}</span>
                <pre>
                  {JSON.stringify(
                    {
                      applicability: selected.applicability,
                      prerequisites: selected.prerequisites,
                      competencies: selected.required_competencies,
                      effective_at: selected.effective_at,
                      review_at: selected.review_at
                    },
                    null,
                    2
                  )}
                </pre>
              </article>
            </div>

            {selected.improvement_candidates.length > 0 && (
              <div className="notice">
                {t("workflow.improvementNotice")}
              </div>
            )}

            <div className="review-actions">
              <span>
                <strong>{t("workflow.governedRelease")}</strong>
                <small>
                  {t("workflow.governedReleaseNote")}
                </small>
              </span>
              {(selected.status === "CANDIDATE" ||
                selected.status === "REVIEW_REQUIRED") && (
                <>
                  <button
                    className="secondary-button"
                    disabled={props.busy}
                    onClick={() => {
                      const value = reason(
                        "workflow.reviewRequiredPrompt",
                        "workflow.reviewRequiredDefault"
                      );
                      if (value) void props.onReview(selected, "REVIEW_REQUIRED", value);
                    }}
                  >
                    {t("workflow.requireReview")}
                  </button>
                  <button
                    className="secondary-button"
                    disabled={props.busy}
                    onClick={() => {
                      const value = reason(
                        "workflow.rejectPrompt",
                        "workflow.rejectDefault"
                      );
                      if (value) void props.onReview(selected, "REJECTED", value);
                    }}
                  >
                    {t("workflow.reject")}
                  </button>
                  <button
                    className="primary-button"
                    disabled={props.busy}
                    onClick={() => {
                      const value = reason(
                        "workflow.approvePrompt",
                        "workflow.approveDefault"
                      );
                      if (value) void props.onReview(selected, "APPROVED", value);
                    }}
                  >
                    {t("workflow.approveCandidate")}
                  </button>
                </>
              )}
              {selected.status === "APPROVED" && (
                <button
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() => {
                    const value = reason(
                      "workflow.publishPrompt",
                      "workflow.publishDefault"
                    );
                    if (value) void props.onPublish(selected, value);
                  }}
                >
                  {t("workflow.evaluatePublish")}
                </button>
              )}
              {selected.status === "PUBLISHED" && (
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => {
                    const value = reason(
                      "workflow.retirePrompt",
                      "workflow.retireDefault"
                    );
                    if (value) void props.onRetire(selected, value);
                  }}
                >
                  {t("workflow.retireVersion")}
                </button>
              )}
            </div>
          </>
        )}
      </section>
    </div>
  );
}

function MeasurementForm(props: {
  busy: boolean;
  onSubmit: (type: string, value: string, unit: string) => Promise<void>;
}) {
  const { t } = useI18n();
  const [type, setType] = useState("VIBRATION_VELOCITY");
  const [value, setValue] = useState("8.1");
  const unit = useMemo(() => type === "TEMPERATURE" ? "DEG_C" : "MM_PER_S", [type]);
  function submit(event: FormEvent) {
    event.preventDefault();
    void props.onSubmit(type, value, unit);
  }
  return (
    <form className="measurement-form" onSubmit={submit}>
      <span className="eyebrow">{t("measurement.record")}</span>
      <label>{t("measurement.type")}<select value={type} onChange={(event) => setType(event.target.value)}><option value="VIBRATION_VELOCITY">{t("measurement.vibrationVelocity")}</option><option value="TEMPERATURE">{t("measurement.bearingTemperature")}</option></select></label>
      <label>{t("measurement.value")}<input inputMode="decimal" value={value} onChange={(event) => setValue(event.target.value)} /></label>
      <label>{t("measurement.unit")}<input value={unit} readOnly /></label>
      <button className="secondary-button" disabled={props.busy}>{t("measurement.recordButton")}</button>
    </form>
  );
}

function CreateAssetForm(props: {
  siteID?: string;
  busy: boolean;
  onCancel: () => void;
  onCreate: (value: {
    site_id: string;
    tag: string;
    name: string;
    class: string;
  }) => Promise<void>;
}) {
  const { t } = useI18n();
  const [tag, setTag] = useState("P-302");
  const [name, setName] = useState("Process Pump P-302");
  const [assetClass, setAssetClass] = useState("CENTRIFUGAL_PUMP");
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!props.siteID) return;
    void props.onCreate({
      site_id: props.siteID,
      tag,
      name,
      class: assetClass
    });
  }
  return (
    <form className="command-form panel" onSubmit={submit}>
      <div><span className="eyebrow">{t("form.lightweightNative")}</span><h2>{t("form.createAsset")}</h2></div>
      <label>{t("form.tag")}<input value={tag} onChange={(event) => setTag(event.target.value)} required /></label>
      <label>{t("form.name")}<input value={name} onChange={(event) => setName(event.target.value)} required /></label>
      <label>{t("form.classLabel")}<input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} required /></label>
      <div className="form-actions"><button type="button" className="secondary-button" onClick={props.onCancel}>{t("form.cancel")}</button><button className="primary-button" disabled={props.busy || !props.siteID}>{t("form.create")}</button></div>
    </form>
  );
}

function CreateIncidentForm(props: {
  assets: Asset[];
  busy: boolean;
  onCancel: () => void;
  onCreate: (value: {
    site_id: string;
    asset_id: string;
    summary: string;
    severity: string;
  }) => Promise<void>;
}) {
  const { t } = useI18n();
  const [assetID, setAssetID] = useState(props.assets[0]?.id ?? "");
  const [summary, setSummary] = useState("High vibration");
  const [severity, setSeverity] = useState("HIGH");
  const asset = props.assets.find((item) => item.id === assetID);
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!asset) return;
    void props.onCreate({
      site_id: asset.site_id,
      asset_id: asset.id,
      summary,
      severity
    });
  }
  return (
    <form className="command-form panel incident-command" onSubmit={submit}>
      <div><span className="eyebrow">{t("form.abnormalCondition")}</span><h2>{t("form.createIncident")}</h2></div>
      <label>{t("form.asset")}<select value={assetID} onChange={(event) => setAssetID(event.target.value)} required><option value="" disabled>{t("form.selectAsset")}</option>{props.assets.map((item) => <option key={item.id} value={item.id}>{item.tag} · {item.name}</option>)}</select></label>
      <label>{t("form.summary")}<input value={summary} onChange={(event) => setSummary(event.target.value)} required /></label>
      <label>{t("form.severity")}<select value={severity} onChange={(event) => setSeverity(event.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option><option>CRITICAL</option></select></label>
      <div className="form-actions"><button type="button" className="secondary-button" onClick={props.onCancel}>{t("form.cancel")}</button><button className="primary-button" disabled={props.busy || !asset}>{t("form.create")}</button></div>
    </form>
  );
}

function EmptyRow({ columns, labelKey }: { columns: number; labelKey: MessageKey }) {
  const { t } = useI18n();
  return <tr><td className="empty" colSpan={columns}>{t(labelKey)}</td></tr>;
}
