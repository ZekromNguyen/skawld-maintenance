import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import { relativeTime, severityTone } from "./presentation";
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
  const [view, setView] = useState<View>("overview");
  const [principal, setPrincipal] = useState<Principal>();
  const [assets, setAssets] = useState<Asset[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [documents, setDocuments] = useState<KnowledgeDocument[]>([]);
  const [recommendation, setRecommendation] = useState<Recommendation>();
  const [report, setReport] = useState<MaintenanceReport>();
  const [handover, setHandover] = useState<ShiftHandover>();
  const [demonstrations, setDemonstrations] = useState<Demonstration[]>([]);
  const [selectedDemonstration, setSelectedDemonstration] = useState<Demonstration>();
  const [workflows, setWorkflows] = useState<WorkflowVersion[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowVersion>();
  const [quality, setQuality] = useState<EvaluationSummary>();
  const [selectedIncident, setSelectedIncident] = useState<Incident>();
  const [execution, setExecution] = useState<Execution>();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string>();
  const [showAssetForm, setShowAssetForm] = useState(false);
  const [showIncidentForm, setShowIncidentForm] = useState(false);

  const reload = useCallback(async () => {
    try {
      const [identity, assetResult, incidentResult] = await Promise.all([
        api.principal(),
        api.assets(),
        api.incidents()
      ]);
      const documentResult = await api.documents(identity.site_ids[0]);
      const demonstrationResult = await api.demonstrations(identity.site_ids[0]);
      const workflowResult = await api.workflows();
      setPrincipal(identity);
      setAssets(assetResult.items);
      setIncidents(incidentResult.items);
      setDocuments(documentResult.items);
      setDemonstrations(demonstrationResult.items);
      setWorkflows(workflowResult.items);
      setMessage(undefined);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Unable to load maintenance data");
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const openIncidents = incidents.filter((incident) => incident.state !== "RESOLVED");
  const criticalAssets = assets.filter((asset) => asset.criticality?.rating === "A");
  const activeExecutionSteps = execution?.steps.filter((step) => step.state === "COMPLETED").length ?? 0;

  async function mutate(action: () => Promise<void>) {
    setBusy(true);
    setMessage(undefined);
    try {
      await action();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Command failed");
    } finally {
      setBusy(false);
    }
  }

  async function createExecution() {
    if (!selectedIncident) return;
    await mutate(async () => {
      const created = await api.createExecution(selectedIncident.id);
      setExecution(created);
      await reload();
    });
  }

  async function startExecution() {
    if (!execution) return;
    await mutate(async () => setExecution(await api.startExecution(execution)));
  }

  async function completeStep(step: Step) {
    if (!execution) return;
    await mutate(async () => {
      await api.completeStep(execution, step);
      setExecution(await api.execution(execution.id));
    });
  }

  async function recordMeasurement(
    measurementType: string,
    value: string,
    unit: string
  ) {
    if (!execution) return;
    await mutate(async () => {
      await api.recordMeasurement(execution, measurementType, value, unit);
      setExecution(await api.execution(execution.id));
    });
  }

  async function generateRecommendation() {
    if (!selectedIncident) return;
    await mutate(async () => {
      setRecommendation(await api.recommendation(selectedIncident.id, execution?.id));
    });
  }

  async function reviewRecommendation(
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
  ) {
    if (!recommendation) return;
    await mutate(async () => {
      await api.reviewRecommendation(recommendation.id, {
        outcome,
        reason,
        material_claims: materialClaims,
        supported_claims: supportedClaims,
        retrieved_evidence: retrievedEvidence,
        relevant_evidence: relevantEvidence
      });
      setQuality(await api.evaluationSummary());
      setMessage("Quality review recorded as an append-only evaluation label.");
    });
  }

  async function openQuality() {
    setView("quality");
    await mutate(async () => setQuality(await api.evaluationSummary()));
  }

  async function draftReport() {
    if (!execution) return;
    await mutate(async () => setReport(await api.draftReport(execution.id)));
  }

  async function startDemonstration(
    subjectKind: "EXECUTION" | "HANDOVER",
    subjectID: string
  ) {
    await mutate(async () => {
      const value = await api.startDemonstration(subjectKind, subjectID);
      setSelectedDemonstration(value);
      setView("demonstrations");
      await reload();
    });
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">SK</span>
          <span>
            <strong>Skawld</strong>
            <small>Maintenance Intelligence</small>
          </span>
        </div>
        <nav aria-label="Primary navigation">
          <NavItem active={view === "overview"} label="Operations overview" onClick={() => setView("overview")} />
          <NavItem active={view === "assets"} label="Asset knowledge" onClick={() => setView("assets")} />
          <NavItem active={view === "incidents"} label="Incident execution" onClick={() => setView("incidents")} />
          <NavItem active={view === "knowledge"} label="Procedures & evidence" onClick={() => setView("knowledge")} />
          <NavItem active={view === "handover"} label="Shift handover" onClick={() => setView("handover")} />
          <NavItem active={view === "demonstrations"} label="Demonstrations" onClick={() => setView("demonstrations")} />
          <NavItem active={view === "workflows"} label="Learned workflows" onClick={() => setView("workflows")} />
          <NavItem active={view === "quality"} label="AI quality & safety" onClick={() => void openQuality()} />
        </nav>
        <div className="safety-boundary">
          <span className="eyebrow">Safety boundary</span>
          <strong>Advisory only</strong>
          <p>No machinery control or permit authority.</p>
        </div>
      </aside>

      <main>
        <header className="topbar">
          <div>
            <span className="eyebrow">Maintenance operations</span>
            <h1>{titleFor(view)}</h1>
          </div>
          <div className="operator">
            <span className="presence" />
            <span>
              <strong>{principal?.display_name ?? "Connecting…"}</strong>
              <small>{principal?.site_ids.length || "All"} site scope</small>
            </span>
          </div>
        </header>

        {message && <div className="notice" role="alert">{message}</div>}

        {view === "overview" && (
          <Overview
            assets={assets}
            incidents={incidents}
            openCount={openIncidents.length}
            criticalAssetCount={criticalAssets.length}
            onOpenIncidents={() => setView("incidents")}
          />
        )}
        {view === "assets" && (
          <>
            {showAssetForm && (
              <CreateAssetForm
                siteID={principal?.site_ids[0]}
                busy={busy}
                onCancel={() => setShowAssetForm(false)}
                onCreate={(value) =>
                  mutate(async () => {
                    await api.createAsset(value);
                    setShowAssetForm(false);
                    await reload();
                  })
                }
              />
            )}
            <AssetsTable assets={assets} onCreate={() => setShowAssetForm(true)} />
          </>
        )}
        {view === "incidents" && (
          <div className="workbench">
            <IncidentQueue
              incidents={incidents}
              selected={selectedIncident}
              onSelect={(incident) => {
                setSelectedIncident(incident);
                setExecution(undefined);
              }}
              onCreate={() => setShowIncidentForm(true)}
            />
            <ExecutionPanel
              incident={selectedIncident}
              execution={execution}
              completedSteps={activeExecutionSteps}
              busy={busy}
              onCreate={createExecution}
              onStart={startExecution}
              onCompleteStep={completeStep}
              onMeasurement={recordMeasurement}
              recommendation={recommendation}
              report={report}
              onRecommend={generateRecommendation}
              onReviewRecommendation={reviewRecommendation}
              onDraftReport={draftReport}
              onCapture={() =>
                execution
                  ? startDemonstration("EXECUTION", execution.id)
                  : Promise.resolve()
              }
              onEvidenceView={(evidenceID) => {
                const active = demonstrations.find(
                  (value) =>
                    value.status === "recording" &&
                    value.subject_kind === "EXECUTION" &&
                    value.subject_id === execution?.id
                );
                return active
                  ? api.recordEvidenceView(
                      active.id,
                      evidenceID,
                      "inspect recommendation evidence"
                    ).then(() => undefined)
                  : Promise.resolve();
              }}
            />
            {showIncidentForm && (
              <CreateIncidentForm
                assets={assets}
                busy={busy}
                onCancel={() => setShowIncidentForm(false)}
                onCreate={(value) =>
                  mutate(async () => {
                    const created = await api.createIncident(value);
                    setShowIncidentForm(false);
                    setSelectedIncident(created);
                    await reload();
                  })
                }
              />
            )}
          </div>
        )}
        {view === "knowledge" && (
          <KnowledgePanel
            siteID={principal?.site_ids[0]}
            documents={documents}
            busy={busy}
            onRefresh={reload}
            onMutate={mutate}
          />
        )}
        {view === "handover" && (
          <HandoverPanel
            siteID={principal?.site_ids[0]}
            handover={handover}
            busy={busy}
            onPrepare={() =>
              mutate(async () => {
                if (principal?.site_ids[0]) {
                  setHandover(await api.prepareHandover(principal.site_ids[0]));
                }
              })
            }
            onCapture={() =>
              handover
                ? startDemonstration("HANDOVER", handover.id)
                : Promise.resolve()
            }
          />
        )}
        {view === "demonstrations" && (
          <DemonstrationPanel
            values={demonstrations}
            selected={selectedDemonstration}
            busy={busy}
            onSelect={(value) =>
              mutate(async () =>
                setSelectedDemonstration(await api.demonstration(value.id))
              )
            }
            onComplete={(value, outcome) =>
              mutate(async () => {
                setSelectedDemonstration(
                  await api.completeDemonstration(value.id, outcome)
                );
                await reload();
              })
            }
            onRedact={(value, eventID, path, reason) =>
              mutate(async () => {
                await api.redactDemonstrationEvent(
                  value.id,
                  eventID,
                  path,
                  reason
                );
                setSelectedDemonstration(await api.demonstration(value.id));
              })
            }
            onReview={(value, decision, reason) =>
              mutate(async () => {
                await api.reviewDemonstration(value.id, decision, reason);
                setSelectedDemonstration(await api.demonstration(value.id));
                await reload();
              })
            }
          />
        )}
        {view === "workflows" && (
          <WorkflowLearningPanel
            values={workflows}
            demonstrations={demonstrations}
            selected={selectedWorkflow}
            busy={busy}
            onSelect={(value) =>
              mutate(async () =>
                setSelectedWorkflow(
                  await api.workflow(value.workflow_id, value.version)
                )
              )
            }
            onCompile={(demonstrationIDs) =>
              mutate(async () => {
                const source = demonstrations.find((demonstration) =>
                  demonstrationIDs.includes(demonstration.id)
                );
                const name =
                  source?.workflow_key === "maintenance.shift_handover"
                    ? "Shift handover workflow"
                    : "High vibration pump inspection";
                const value = await api.compileWorkflow(
                  name,
                  demonstrationIDs
                );
                setSelectedWorkflow(value);
                await reload();
              })
            }
            onReview={(value, decision, reason) =>
              mutate(async () => {
                const asset = assets.find(
                  (item) =>
                    item.site_id === value.site_id &&
                    item.class === value.asset_class
                );
                const applicability = [{
                  site_id: value.site_id,
                  asset_id: asset?.id,
                  asset_class: value.asset_class,
                  manufacturer: asset?.manufacturer,
                  model: asset?.model,
                  validation_status: "VALIDATED" as const
                }];
                setSelectedWorkflow(
                  await api.reviewWorkflow(
                    value,
                    decision,
                    reason,
                    applicability
                  )
                );
                await reload();
              })
            }
            onPublish={(value, reason) =>
              mutate(async () => {
                setSelectedWorkflow(await api.publishWorkflow(value, reason));
                await reload();
              })
            }
            onRetire={(value, reason) =>
              mutate(async () => {
                setSelectedWorkflow(await api.retireWorkflow(value, reason));
                await reload();
              })
            }
          />
        )}
        {view === "quality" && <QualityPanel value={quality} busy={busy} onRefresh={openQuality} />}
      </main>
    </div>
  );
}

function NavItem(props: { active: boolean; label: string; onClick: () => void }) {
  return (
    <button className={props.active ? "nav-item active" : "nav-item"} onClick={props.onClick}>
      <span className="nav-indicator" />
      {props.label}
    </button>
  );
}

function Overview(props: {
  assets: Asset[];
  incidents: Incident[];
  openCount: number;
  criticalAssetCount: number;
  onOpenIncidents: () => void;
}) {
  return (
    <>
      <section className="metrics" aria-label="Operational status">
        <Metric label="Registered assets" value={props.assets.length} detail="Native and external projections" />
        <Metric label="Open incidents" value={props.openCount} detail="Require technician attention" accent />
        <Metric label="Critical assets" value={props.criticalAssetCount} detail="Human-approved criticality A" />
        <Metric label="Unsafe AI actions" value={0} detail="Hard safety boundary enforced" safe />
      </section>
      <section className="panel">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">Priority queue</span>
            <h2>Active maintenance incidents</h2>
          </div>
          <button className="secondary-button" onClick={props.onOpenIncidents}>Open workbench</button>
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
  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">Source-aware registry</span>
          <h2>Asset knowledge</h2>
        </div>
        <div className="heading-actions"><span className="count">{assets.length} assets</span><button className="secondary-button" onClick={onCreate}>Add asset</button></div>
      </div>
      <div className="table-wrap">
        <table>
          <thead><tr><th>Tag</th><th>Asset</th><th>Class</th><th>Criticality</th><th>Authority</th><th>Status</th></tr></thead>
          <tbody>
            {assets.map((asset) => (
              <tr key={asset.id}>
                <td className="mono strong">{asset.tag}</td>
                <td><strong>{asset.name}</strong><small>{[asset.manufacturer, asset.model].filter(Boolean).join(" · ") || "No OEM metadata"}</small></td>
                <td>{asset.class}</td>
                <td><span className={`criticality rating-${asset.criticality?.rating ?? "none"}`}>{asset.criticality?.rating ?? "—"}</span></td>
                <td><span className="source-badge">{asset.source_of_truth === "EXTERNAL_REFERENCE" ? "External projection" : "Skawld native"}</span></td>
                <td><span className="status-dot" />{asset.status}</td>
              </tr>
            ))}
            {assets.length === 0 && <EmptyRow columns={6} label="No assets in your authorized site scope." />}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function IncidentTable({ incidents }: { incidents: Incident[] }) {
  return (
    <div className="table-wrap">
      <table>
        <thead><tr><th>Incident</th><th>Asset</th><th>Summary</th><th>Severity</th><th>State</th><th>Detected</th></tr></thead>
        <tbody>
          {incidents.map((incident) => (
            <tr key={incident.id}>
              <td className="mono strong">{incident.number}</td>
              <td className="mono">{incident.asset_tag || "—"}</td>
              <td className="summary-cell">{incident.summary}</td>
              <td><span className={`severity ${severityTone(incident.severity)}`}>{incident.severity}</span></td>
              <td>{incident.state.replace("_", " ")}</td>
              <td>{relativeTime(incident.detected_at)}</td>
            </tr>
          ))}
          {incidents.length === 0 && <EmptyRow columns={6} label="No incidents in your authorized site scope." />}
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
  return (
    <section className="queue panel">
      <div className="panel-heading">
        <div><span className="eyebrow">Authorized scope</span><h2>Incident queue</h2></div>
        <div className="heading-actions"><span className="count">{props.incidents.length}</span><button className="secondary-button" onClick={props.onCreate}>New</button></div>
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
              <span className="queue-meta"><span className="mono">{incident.number}</span><span>{relativeTime(incident.detected_at)}</span></span>
              <strong>{incident.asset_tag} · {incident.summary}</strong>
              <small>{incident.state.replace("_", " ")} · {incident.severity}</small>
            </span>
          </button>
        ))}
        {props.incidents.length === 0 && <div className="empty">No incidents available.</div>}
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
  if (!props.incident) {
    return <section className="execution panel empty-state"><strong>Select an incident</strong><p>Review context and start a deterministic inspection workflow.</p></section>;
  }
  if (!props.execution) {
    return (
      <section className="execution panel empty-state">
        <span className={`severity ${severityTone(props.incident.severity)}`}>{props.incident.severity}</span>
        <strong>{props.incident.asset_tag} · {props.incident.summary}</strong>
        <p>This creates a Skawld-owned execution record. It does not create or replace an authoritative CMMS work order.</p>
        <button className="primary-button" disabled={props.busy || props.incident.state === "RESOLVED"} onClick={() => void props.onCreate()}>
          Create pump inspection
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
          <span className="eyebrow">Deterministic execution</span>
          <h2>{props.execution.asset_tag} inspection</h2>
        </div>
        <span className="state-badge">{props.execution.state.replace("_", " ")}</span>
      </div>
      <div className="progress-row"><span style={{ width: `${progress}%` }} /><small>{props.completedSteps}/{props.execution.steps.length} steps</small></div>
      {props.execution.state === "ASSIGNED" && (
        <button className="primary-button full" disabled={props.busy} onClick={() => void props.onStart()}>Start inspection</button>
      )}
      <div className="steps">
        {props.execution.steps.map((step) => (
          <article key={step.id} className={`step step-${step.state.toLowerCase()}`}>
            <span className="step-number">{step.sequence}</span>
            <div>
              <strong>{step.title}</strong>
              <small>{step.risk_level.replaceAll("_", " ")}</small>
              {step.required_prerequisite && <span className="prerequisite">Requires verified {step.required_prerequisite.replaceAll("_", " ")}</span>}
              {step.blocked_reason && <p className="blocked-reason">{step.blocked_reason}</p>}
            </div>
            <button
              className="step-action"
              disabled={props.busy || execution.state !== "IN_PROGRESS" || step.state === "COMPLETED"}
              onClick={() => void props.onCompleteStep(step)}
            >
              {step.state === "COMPLETED" ? "Done" : step.state === "BLOCKED" ? "Retry" : "Complete"}
            </button>
          </article>
        ))}
      </div>
      {props.execution.state === "IN_PROGRESS" && (
        <MeasurementForm onSubmit={props.onMeasurement} busy={props.busy} />
      )}
      {props.execution.measurements.length > 0 && (
        <div className="evidence">
          <span className="eyebrow">Recorded evidence</span>
          {props.execution.measurements.map((measurement) => (
            <span key={measurement.id}><strong>{measurement.value}</strong> {measurement.unit.replaceAll("_", "/")} · {measurement.measurement_type.replaceAll("_", " ")}</span>
          ))}
        </div>
      )}
      <div className="copilot-actions">
        <button className="primary-button" disabled={props.busy} onClick={() => void props.onCapture()}>
          Start semantic demonstration
        </button>
        <button className="secondary-button" disabled={props.busy} onClick={() => void props.onRecommend()}>
          Generate evidence-backed recommendation
        </button>
        <button className="secondary-button" disabled={props.busy} onClick={() => void props.onDraftReport()}>
          Prepare report draft
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
  const value = props.value;
  const [outcome, setOutcome] = useState<
    "ACCEPTED" | "REJECTED" | "CORRECTED" | "UNSAFE" | "UNSUPPORTED" | "INCORRECT_NEXT_STEP"
  >("ACCEPTED");
  const [reason, setReason] = useState("Reviewed against the displayed evidence packet");
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
        <div><span className="eyebrow">Advisory proposal</span><h2>{value.output.status.replaceAll("_", " ")}</h2></div>
        <span className="state-badge">{Math.round(value.output.confidence * 100)}% · {value.output.risk_level}</span>
      </div>
      <p>{value.output.recommendation || "The eligible evidence packet is insufficient for a recommendation."}</p>
      <EvidenceLinks
        evidence={value.evidence}
        selected={value.output.evidence_ids}
        onView={props.onEvidenceView}
      />
      <small>Unknowns: {value.output.unknowns.join(" · ") || "None stated"} · Human confirmation required</small>
      <small className="provenance">{value.provider}/{value.model} · {value.prompt_version}</small>
      <div className="quality-review">
        <span className="eyebrow">Pilot quality label</span>
        <select value={outcome} onChange={(event) => setOutcome(event.target.value as typeof outcome)}>
          <option value="ACCEPTED">Accepted</option>
          <option value="REJECTED">Rejected</option>
          <option value="UNSAFE">Unsafe</option>
          <option value="UNSUPPORTED">Unsupported claim</option>
          <option value="INCORRECT_NEXT_STEP">Incorrect next step</option>
        </select>
        <input value={reason} onChange={(event) => setReason(event.target.value)} aria-label="Quality review reason" />
        <div className="quality-counts">
          <label>
            Material claims
            <input
              type="number"
              min="0"
              value={materialClaims}
              onChange={(event) => setMaterialClaims(Number(event.target.value))}
            />
          </label>
          <label>
            Supported claims
            <input
              type="number"
              min="0"
              max={materialClaims}
              value={supportedClaims}
              onChange={(event) => setSupportedClaims(Number(event.target.value))}
            />
          </label>
          <label>
            Retrieved evidence
            <input
              type="number"
              min="0"
              value={retrievedEvidence}
              onChange={(event) => setRetrievedEvidence(Number(event.target.value))}
            />
          </label>
          <label>
            Relevant evidence
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
          Record review
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
  if (!props.value) {
    return <section className="panel empty-state"><strong>No quality summary loaded</strong></section>;
  }
  const percent = (value: number) => `${Math.round(value * 100)}%`;
  return (
    <>
      <section className="metrics" aria-label="AI quality and safety metrics">
        <Metric label="Review coverage" value={Math.round(props.value.review_coverage * 100)} detail={`${props.value.reviewed}/${props.value.recommendations} recommendations reviewed`} />
        <Metric label="Evidence coverage" value={Math.round(props.value.evidence_coverage * 100)} detail="Reviewer-labeled supported claims" />
        <Metric label="Unsafe rate" value={Math.round(props.value.unsafe_recommendation_rate * 100)} detail="Must remain at zero" accent={props.value.unsafe_recommendation_rate > 0} safe={props.value.unsafe_recommendation_rate === 0} />
        <Metric label="Workflow gates" value={Math.round(props.value.workflow_gate_pass_rate * 100)} detail={`${props.value.workflow_evaluations} evaluated versions`} />
      </section>
      <section className="panel quality-summary">
        <div className="panel-heading">
          <div><span className="eyebrow">Pilot evaluation</span><h2>Human-reviewed quality signals</h2></div>
          <button className="secondary-button" disabled={props.busy} onClick={() => void props.onRefresh()}>Refresh</button>
        </div>
        <dl>
          <div><dt>Acceptance</dt><dd>{percent(props.value.recommendation_acceptance)}</dd></div>
          <div><dt>Human override</dt><dd>{percent(props.value.human_override_rate)}</dd></div>
          <div><dt>Unsupported</dt><dd>{percent(props.value.unsupported_recommendation_rate)}</dd></div>
          <div><dt>Incorrect next step</dt><dd>{percent(props.value.incorrect_next_step_rate)}</dd></div>
          <div><dt>Retrieval precision</dt><dd>{percent(props.value.retrieval_precision)}</dd></div>
          <div><dt>LLM calls</dt><dd>{props.value.llm_calls}</dd></div>
          <div><dt>Average latency</dt><dd>{Math.round(props.value.average_latency_ms)} ms</dd></div>
          <div><dt>Tokens</dt><dd>{props.value.tokens_in + props.value.tokens_out}</dd></div>
          <div><dt>Estimated cost</dt><dd>{props.value.estimated_cost_micros} μ</dd></div>
        </dl>
        <p className="muted">Zero values mean “not yet labeled”, not proof of quality. Safety gates use the frozen offline evaluation dataset in CI in addition to these pilot observations.</p>
      </section>
    </>
  );
}

function ReportCard({ value }: { value: MaintenanceReport }) {
  return (
    <article className="ai-result">
      <div className="result-header">
        <div><span className="eyebrow">Human-reviewable draft</span><h2>Maintenance report R{value.revision}</h2></div>
        <span className="state-badge">{value.state}</span>
      </div>
      <p>{value.structured_content.summary}</p>
      <EvidenceLinks evidence={value.evidence} selected={value.structured_content.evidence_ids} />
      <small>Unknowns: {value.structured_content.unknowns.join(" · ")}</small>
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
        <div className="panel-heading"><div><span className="eyebrow">Controlled ingestion</span><h2>Add procedure revision</h2></div></div>
        <label>Title<input value={title} onChange={(event) => setTitle(event.target.value)} /></label>
        <label>Asset class<input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} /></label>
        <label>PDF or text<input type="file" accept=".pdf,text/plain,application/pdf" onChange={(event) => setFile(event.target.files?.[0])} /></label>
        <button className="primary-button" disabled={props.busy || !props.siteID || !file}>Upload and queue ingestion</button>
        <p className="form-note">A revision remains ineligible for retrieval until ingestion is ready and an authorized approver publishes it.</p>
      </form>
      <section className="panel">
        <div className="panel-heading"><div><span className="eyebrow">Validity-aware corpus</span><h2>Document revisions</h2></div><span className="count">{props.documents.length}</span></div>
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
          {props.documents.length === 0 && <div className="empty">No controlled documents in this site.</div>}
        </div>
      </section>
      <section className="panel search-panel">
        <div className="panel-heading"><div><span className="eyebrow">Authorization-first RRF</span><h2>Evidence search</h2></div></div>
        <div className="search-row"><input value={query} onChange={(event) => setQuery(event.target.value)} /><button className="secondary-button" disabled={!props.siteID || props.busy} onClick={() => void props.onMutate(async () => setResults((await api.searchKnowledge(props.siteID!, query)).items))}>Search</button></div>
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
  return (
    <section className="panel handover-panel">
      <div className="panel-heading">
        <div><span className="eyebrow">Normal-work intelligence</span><h2>Shift handover draft</h2></div>
        <div className="heading-actions">
          {props.handover && (
            <button className="secondary-button" disabled={props.busy} onClick={() => void props.onCapture()}>
              Start demonstration
            </button>
          )}
          <button className="primary-button" disabled={!props.siteID || props.busy} onClick={() => void props.onPrepare()}>Prepare from current records</button>
        </div>
      </div>
      {!props.handover ? (
        <div className="empty">Prepare a draft from authorized open incidents, active work, blocked prerequisites, and pending steps.</div>
      ) : (
        <div className="handover-content">
          <div className="result-header"><p>{props.handover.structured_content.summary}</p><span className="state-badge">{props.handover.state}</span></div>
          <HandoverSection title="Open incidents" items={props.handover.structured_content.open_incidents} />
          <HandoverSection title="Active executions" items={props.handover.structured_content.active_executions} />
          <HandoverSection title="Safety concerns" items={props.handover.structured_content.safety_concerns} />
          <HandoverSection title="Follow-up" items={props.handover.structured_content.follow_up} />
          <EvidenceLinks evidence={props.handover.evidence} selected={props.handover.structured_content.evidence_ids} />
          <small className="provenance">{props.handover.provider}/{props.handover.model} · {props.handover.prompt_version} · human acceptance required</small>
        </div>
      )}
    </section>
  );
}

function HandoverSection({ title, items }: { title: string; items: string[] }) {
  return <div className="handover-section"><strong>{title}</strong>{items.length ? <ul>{items.map((item, index) => <li key={`${title}-${index}`}>{item}</li>)}</ul> : <p>None in the current authorized snapshot.</p>}</div>;
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
  const selected = props.selected;

  function complete() {
    if (!selected) return;
    const outcome = window.prompt(
      "Final demonstration outcome",
      "Maintenance work completed and outcome verified"
    );
    if (outcome?.trim()) void props.onComplete(selected, outcome.trim());
  }

  function review(
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED"
  ) {
    if (!selected) return;
    const reason = window.prompt(
      "Review reason",
      decision === "APPROVED"
        ? "Semantic trace is coherent and suitable for learning review"
        : "Trace requires expert follow-up"
    );
    if (reason?.trim()) void props.onReview(selected, decision, reason.trim());
  }

  function redact(eventID: string) {
    if (!selected) return;
    const path = window.prompt(
      "JSON path to mask",
      "output.value.narrative"
    );
    if (!path?.trim()) return;
    const reason = window.prompt(
      "Redaction reason",
      "Sensitive operational detail"
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
            <span className="eyebrow">Organizational memory</span>
            <h2>Semantic demonstrations</h2>
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
                {value.subject_kind} · {value.events.length} semantic events
              </small>
            </span>
            <span className="state-badge">{value.status}</span>
            <small>
              Review {value.review_status} · {relativeTime(value.started_at)}
            </small>
          </button>
        ))}
        {props.values.length === 0 && (
          <div className="empty">
            Start capture from an incident execution or shift handover.
          </div>
        )}
      </section>

      <section className="panel demonstration-timeline">
        {!selected ? (
          <div className="empty-state">
            <strong>Select a demonstration</strong>
            <p>
              Review domain meaning, actor, trust, source event, correction
              links, and capture health.
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
                    Complete capture
                  </button>
                )}
              </div>
            </div>
            <div className="capture-health">
              <span>
                <strong>{selected.capture.applied}</strong> applied
              </span>
              <span>
                <strong>{selected.capture.pending}</strong> pending
              </span>
              <span className={selected.capture.failed ? "capture-failed" : ""}>
                <strong>{selected.capture.failed}</strong> failed
              </span>
              <small className="mono">session {selected.session_id}</small>
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
                      {event.entity?.type ?? "event"} ·{" "}
                      <span className="mono">{event.entity?.id ?? event.id}</span>
                    </small>
                    <small>
                      Actor <span className="mono">{event.actor_id}</span> ·{" "}
                      {new Date(event.timestamp).toLocaleString()}
                    </small>
                    {event.correction_of && (
                      <span className="correction-link">
                        Corrects event{" "}
                        <span className="mono">{event.correction_of}</span>
                      </span>
                    )}
                    <details>
                      <summary>Structured semantic payload</summary>
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
                        domain {event.domain_event_id || "manual capture"}
                      </span>
                      <button
                        className="secondary-button"
                        disabled={props.busy}
                        onClick={() => redact(event.id)}
                      >
                        Redact field
                      </button>
                    </div>
                  </div>
                </article>
              ))}
            </div>
            {selected.status === "completed" && (
              <div className="review-actions">
                <span>
                  <strong>Human governance</strong>
                  <small>
                    Review authority is separate from RBAC and checked by the API.
                  </small>
                </span>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => review("REDACTION_REQUIRED")}
                >
                  Request redaction
                </button>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => review("REJECTED")}
                >
                  Reject
                </button>
                <button
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() => review("APPROVED")}
                >
                  Approve trace
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
    promptText: string,
    defaultValue: string
  ): string | undefined {
    const value = window.prompt(promptText, defaultValue)?.trim();
    return value || undefined;
  }

  return (
    <div className="workbench workflow-workbench">
      <section className="panel queue-panel">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">Reviewed evidence only</span>
            <h2>Compile candidate</h2>
          </div>
          <span className="count">{reviewed.length} traces</span>
        </div>
        <p className="muted">
          Select at least two approved demonstrations with the same workflow key.
          Compilation never publishes automatically.
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
                  {value.subject_kind} · {value.events.length} semantic events
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
          Compile review-only candidate
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
            <strong>Select a workflow candidate</strong>
            <p>Review evidence, ambiguity, applicability and behavioral changes.</p>
          </div>
        )}
        {selected && (
          <>
            <div className="panel-heading">
              <div>
                <span className="eyebrow">Human-controlled workflow version</span>
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
                sequence consistency
              </span>
              <span>
                <strong>{selected.analysis.conflicts?.length ?? 0}</strong>
                ambiguous transitions
              </span>
              <span>
                <strong>{selected.source_demonstration_ids.length}</strong>
                source demonstrations
              </span>
              <span>
                <strong>{selected.improvement_candidates.length}</strong>
                correction candidates
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
                      {step.evidence.reduce(
                        (count, value) => count + value.event_ids.length,
                        0
                      )}{" "}
                      immutable event references across {step.evidence.length} traces
                    </p>
                    <details>
                      <summary>Evidence identities</summary>
                      <pre>{JSON.stringify(step.evidence, null, 2)}</pre>
                    </details>
                  </div>
                </article>
              ))}
            </div>

            <div className="workflow-review-grid">
              <article>
                <span className="eyebrow">Behavioral diff</span>
                <pre>{JSON.stringify(selected.behavioral_changes, null, 2)}</pre>
              </article>
              <article>
                <span className="eyebrow">Applicability & validity</span>
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
                Human corrections are stored as improvement candidates only.
                They have not modified this workflow.
              </div>
            )}

            <div className="review-actions">
              <span>
                <strong>Governed release</strong>
                <small>
                  RBAC, ApprovalAuthority, exact candidate digest and deterministic
                  evaluation gates are checked independently.
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
                        "Why does this candidate require more evidence?",
                        "Ambiguity or applicability requires further review"
                      );
                      if (value) void props.onReview(selected, "REVIEW_REQUIRED", value);
                    }}
                  >
                    Require review
                  </button>
                  <button
                    className="secondary-button"
                    disabled={props.busy}
                    onClick={() => {
                      const value = reason(
                        "Why is this candidate rejected?",
                        "Candidate is not supported for publication"
                      );
                      if (value) void props.onReview(selected, "REJECTED", value);
                    }}
                  >
                    Reject
                  </button>
                  <button
                    className="primary-button"
                    disabled={props.busy}
                    onClick={() => {
                      const value = reason(
                        "Record the review rationale",
                        "Evidence, safe tool mapping and applicability reviewed"
                      );
                      if (value) void props.onReview(selected, "APPROVED", value);
                    }}
                  >
                    Approve candidate
                  </button>
                </>
              )}
              {selected.status === "APPROVED" && (
                <button
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() => {
                    const value = reason(
                      "Record publication rationale",
                      "Review and deterministic evaluation gates satisfied"
                    );
                    if (value) void props.onPublish(selected, value);
                  }}
                >
                  Evaluate and publish
                </button>
              )}
              {selected.status === "PUBLISHED" && (
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => {
                    const value = reason(
                      "Record retirement rationale",
                      "Workflow superseded or no longer applicable"
                    );
                    if (value) void props.onRetire(selected, value);
                  }}
                >
                  Retire version
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
  const [type, setType] = useState("VIBRATION_VELOCITY");
  const [value, setValue] = useState("8.1");
  const unit = useMemo(() => type === "TEMPERATURE" ? "DEG_C" : "MM_PER_S", [type]);
  function submit(event: FormEvent) {
    event.preventDefault();
    void props.onSubmit(type, value, unit);
  }
  return (
    <form className="measurement-form" onSubmit={submit}>
      <span className="eyebrow">Record measurement</span>
      <label>Type<select value={type} onChange={(event) => setType(event.target.value)}><option value="VIBRATION_VELOCITY">Vibration velocity</option><option value="TEMPERATURE">Bearing temperature</option></select></label>
      <label>Value<input inputMode="decimal" value={value} onChange={(event) => setValue(event.target.value)} /></label>
      <label>Unit<input value={unit} readOnly /></label>
      <button className="secondary-button" disabled={props.busy}>Record</button>
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
      <div><span className="eyebrow">Lightweight native mode</span><h2>Create asset</h2></div>
      <label>Tag<input value={tag} onChange={(event) => setTag(event.target.value)} required /></label>
      <label>Name<input value={name} onChange={(event) => setName(event.target.value)} required /></label>
      <label>Class<input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} required /></label>
      <div className="form-actions"><button type="button" className="secondary-button" onClick={props.onCancel}>Cancel</button><button className="primary-button" disabled={props.busy || !props.siteID}>Create</button></div>
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
      <div><span className="eyebrow">Abnormal condition</span><h2>Create incident</h2></div>
      <label>Asset<select value={assetID} onChange={(event) => setAssetID(event.target.value)} required><option value="" disabled>Select asset</option>{props.assets.map((item) => <option key={item.id} value={item.id}>{item.tag} · {item.name}</option>)}</select></label>
      <label>Summary<input value={summary} onChange={(event) => setSummary(event.target.value)} required /></label>
      <label>Severity<select value={severity} onChange={(event) => setSeverity(event.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option><option>CRITICAL</option></select></label>
      <div className="form-actions"><button type="button" className="secondary-button" onClick={props.onCancel}>Cancel</button><button className="primary-button" disabled={props.busy || !asset}>Create</button></div>
    </form>
  );
}

function EmptyRow({ columns, label }: { columns: number; label: string }) {
  return <tr><td className="empty" colSpan={columns}>{label}</td></tr>;
}

function titleFor(view: View): string {
  if (view === "assets") return "Asset knowledge";
  if (view === "incidents") return "Incident execution";
  if (view === "knowledge") return "Procedures and evidence";
  if (view === "handover") return "Shift handover";
  if (view === "demonstrations") return "Expert demonstrations";
  if (view === "workflows") return "Learned workflow review";
  if (view === "quality") return "AI quality and safety";
  return "Operations overview";
}
