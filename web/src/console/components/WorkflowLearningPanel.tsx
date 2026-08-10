import { useState } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import { PromptDialog } from "../feedback/PromptDialog";
import { StatusBadge, type Tone } from "../ui/StatusBadge";
import { GatedButton } from "../ui/GatedButton";
import type { Demonstration, WorkflowVersion } from "../../types";

interface PromptSpec {
  title: string;
  label: string;
  defaultValue: string;
  confirmLabel: string;
  onConfirm: (value: string) => void;
}

const STATUS_TONE: Record<string, Tone> = {
  CANDIDATE: "info",
  REVIEW_REQUIRED: "medium",
  APPROVED: "success",
  PUBLISHED: "success",
  REJECTED: "critical",
  RETIRED: "low"
};

export function WorkflowLearningPanel(props: {
  values: WorkflowVersion[];
  demonstrations: Demonstration[];
  selected?: WorkflowVersion;
  busy: boolean;
  canReview: boolean;
  canPublish: boolean;
  onSelect: (value: WorkflowVersion) => void;
  onCompile: (demonstrationIDs: string[]) => void;
  onReview: (
    value: WorkflowVersion,
    decision: "APPROVED" | "REJECTED" | "REVIEW_REQUIRED",
    reason: string
  ) => void;
  onPublish: (value: WorkflowVersion, reason: string) => void;
  onRetire: (value: WorkflowVersion, reason: string) => void;
}) {
  const { t } = useI18n();
  const permissionReason = t("action.permissionRequired");
  const [selectedDemonstrations, setSelectedDemonstrations] = useState<string[]>([]);
  const [compileError, setCompileError] = useState<string | undefined>(undefined);
  const [prompt, setPrompt] = useState<PromptSpec | null>(null);

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
    setCompileError(undefined);
  }

  function compile() {
    const keys = new Set(
      selectedDemonstrations
        .map((id) => props.demonstrations.find((demo) => demo.id === id)?.workflow_key)
        .filter(Boolean)
    );
    if (keys.size > 1) {
      setCompileError(t("workflow.compileKeyMismatch"));
      return;
    }
    setCompileError(undefined);
    props.onCompile(selectedDemonstrations);
  }

  const lastRejected = selected?.reviews?.find((review) => review.decision === "REJECTED");
  const sequenceConsistency = selected?.analysis?.sequence_consistency;

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
        <p className="muted">{t("workflow.compileHint")}</p>
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
        {compileError ? (
          <p className="form-error" role="alert" style={{ margin: "0 16px 8px" }}>{compileError}</p>
        ) : null}
        <button
          className="primary-button"
          disabled={props.busy || selectedDemonstrations.length < 2}
          onClick={compile}
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
              aria-current={
                selected?.workflow_id === value.workflow_id && selected.version === value.version
                  ? "true"
                  : undefined
              }
              key={`${value.workflow_id}:${value.version}`}
              onClick={() => props.onSelect(value)}
            >
              <span>
                <strong>{value.name || value.workflow_key}</strong>
                <small>v{value.version} · {value.asset_class}</small>
              </span>
              <StatusBadge tone={STATUS_TONE[value.status] ?? "info"} label={value.status.replace("_", " ")} />
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
                <h2>{selected.name || selected.workflow_key}</h2>
                <small className="mono">
                  {selected.workflow_key} · v{selected.version}
                </small>
              </div>
              <StatusBadge tone={STATUS_TONE[selected.status] ?? "info"} label={(selected.status ?? "").replace("_", " ")} />
            </div>
            {lastRejected ? (
              <div className="notice" role="alert">{t("workflow.rejectedNotice")}</div>
            ) : null}

            <div className="workflow-facts">
              <span>
                <strong>
                  {sequenceConsistency == null
                    ? "—"
                    : `${Math.round(sequenceConsistency * 100)}%`}
                </strong>
                {sequenceConsistency == null ? t("workflow.noAnalysis") : t("workflow.sequenceConsistency")}
              </span>
              <span>
                <strong>{selected?.analysis?.conflicts?.length ?? 0}</strong>
                {t("workflow.ambiguousTransitions")}
              </span>
              <span>
                <strong>{(selected.source_demonstration_ids ?? []).length}</strong>
                {t("workflow.sourceDemos")}
              </span>
              <span>
                <strong>{(selected.improvement_candidates ?? []).length}</strong>
                {t("workflow.correctionCandidates")}
              </span>
            </div>

            <div className="workflow-steps">
              {(selected.steps ?? []).map((step, index) => (
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

            <div className="panel governance-panel">
              <div className="panel-heading"><h2>{t("workflow.governanceTrail")}</h2></div>
              {(selected.reviews?.length ?? 0) > 0 ? (
                <div className="readable-list">
                  {(selected.reviews ?? []).map((review, index) => (
                    <div key={index} className="readable-row">
                      <div className="readable-key-value">
                        <span className="mono muted">decision</span>
                        <span>{review.decision}</span>
                      </div>
                      <div className="readable-key-value">
                        <span className="mono muted">reason</span>
                        <span>{review.reason}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="empty">{t("workflow.noReviews")}</div>
              )}
            </div>

            {(selected.improvement_candidates ?? []).length > 0 && (
              <div className="notice">{t("workflow.improvementNotice")}</div>
            )}

            <div className="review-actions">
              <span>
                <strong>{t("workflow.governedRelease")}</strong>
                <small>{t("workflow.governedReleaseNote")}</small>
              </span>
              {(selected.status === "CANDIDATE" || selected.status === "REVIEW_REQUIRED") && (
                <>
                  <GatedButton
                    allowed={props.canReview}
                    reason={permissionReason}
                    className="secondary-button"
                    disabled={props.busy}
                    onClick={() =>
                      setPrompt({
                        title: t("workflow.requireReview"),
                        label: t("workflow.reviewRequiredPrompt"),
                        defaultValue: t("workflow.reviewRequiredDefault"),
                        confirmLabel: t("workflow.requireReview"),
                        onConfirm: (value) => props.onReview(selected, "REVIEW_REQUIRED", value)
                      })
                    }
                  >
                    {t("workflow.requireReview")}
                  </GatedButton>
                  <GatedButton
                    allowed={props.canReview}
                    reason={permissionReason}
                    className="secondary-button"
                    disabled={props.busy}
                    onClick={() =>
                      setPrompt({
                        title: t("workflow.reject"),
                        label: t("workflow.rejectPrompt"),
                        defaultValue: t("workflow.rejectDefault"),
                        confirmLabel: t("workflow.reject"),
                        onConfirm: (value) => props.onReview(selected, "REJECTED", value)
                      })
                    }
                  >
                    {t("workflow.reject")}
                  </GatedButton>
                  <GatedButton
                    allowed={props.canReview}
                    reason={permissionReason}
                    className="primary-button"
                    disabled={props.busy}
                    onClick={() =>
                      setPrompt({
                        title: t("workflow.approveCandidate"),
                        label: t("workflow.approvePrompt"),
                        defaultValue: t("workflow.approveDefault"),
                        confirmLabel: t("workflow.approveCandidate"),
                        onConfirm: (value) => props.onReview(selected, "APPROVED", value)
                      })
                    }
                  >
                    {t("workflow.approveCandidate")}
                  </GatedButton>
                </>
              )}
              {selected.status === "APPROVED" && (
                <GatedButton
                  allowed={props.canPublish}
                  reason={permissionReason}
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() =>
                    setPrompt({
                      title: t("workflow.evaluatePublish"),
                      label: t("workflow.publishPrompt"),
                      defaultValue: t("workflow.publishDefault"),
                      confirmLabel: t("workflow.evaluatePublish"),
                      onConfirm: (value) => props.onPublish(selected, value)
                    })
                  }
                >
                  {t("workflow.evaluatePublish")}
                </GatedButton>
              )}
              {selected.status === "PUBLISHED" && (
                <GatedButton
                  allowed={props.canPublish}
                  reason={permissionReason}
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() =>
                    setPrompt({
                      title: t("workflow.retireVersion"),
                      label: t("workflow.retirePrompt"),
                      defaultValue: t("workflow.retireDefault"),
                      confirmLabel: t("workflow.retireVersion"),
                      onConfirm: (value) => props.onRetire(selected, value)
                    })
                  }
                >
                  {t("workflow.retireVersion")}
                </GatedButton>
              )}
            </div>
          </>
        )}
      </section>

      <PromptDialog
        open={prompt !== null}
        onOpenChange={(open) => {
          if (!open) setPrompt(null);
        }}
        title={prompt?.title ?? ""}
        label={prompt?.label ?? ""}
        defaultValue={prompt?.defaultValue ?? ""}
        confirmLabel={prompt?.confirmLabel ?? ""}
        pending={props.busy}
        onConfirm={(value) => {
          prompt?.onConfirm(value);
        }}
      />
    </div>
  );
}
