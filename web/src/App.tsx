import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { api } from "./api";
import { relativeTime, severityTone } from "./presentation";
import { useI18n } from "./i18n/I18nProvider";
import type { Locale, MessageKey } from "./i18n/messages";
import { ConsoleLayout } from "./console/layout/ConsoleLayout";
import { PlaceholderPage } from "./console/pages/PlaceholderPage";
import { DashboardPage } from "./console/pages/DashboardPage";
import { QualityPage } from "./console/pages/QualityPage";
import { KnowledgePage } from "./console/pages/KnowledgePage";
import { HandoverPage } from "./console/pages/HandoverPage";
import { DemonstrationsPage } from "./console/pages/DemonstrationsPage";
import { WorkflowsPage } from "./console/pages/WorkflowsPage";
import { EmptyRow } from "./console/components/EmptyRow";
import { EvidenceLinks } from "./console/components/EvidenceLinks";
import { AssetsPage } from "./console/pages/AssetsPage";
import { AssetDetailPage } from "./console/pages/AssetDetailPage";
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
          <Route path="/" element={<DashboardPage />} />
          <Route path="/assets" element={<AssetsPage />} />
          <Route path="/assets/:assetId" element={<AssetDetailPage />} />
          <Route path="/incidents" element={<PlaceholderPage titleKey="nav.incidents" />} />
          <Route path="/incidents/:incidentId" element={<PlaceholderPage titleKey="nav.incidents" />} />
          <Route path="/executions/:executionId" element={<PlaceholderPage titleKey="nav.executions" />} />
          <Route path="/reports" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/reports/:reportId" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/handovers" element={<HandoverPage />} />
          <Route path="/knowledge" element={<KnowledgePage />} />
          <Route path="/knowledge/:documentId" element={<PlaceholderPage titleKey="nav.knowledge" />} />
          <Route path="/search" element={<PlaceholderPage titleKey="nav.search" />} />
          <Route path="/quality" element={<QualityPage />} />
          <Route path="/demonstrations" element={<DemonstrationsPage />} />
          <Route path="/workflows" element={<WorkflowsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
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

