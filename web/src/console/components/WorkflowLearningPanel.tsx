import { useState } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import type { Demonstration, WorkflowVersion } from "../../types";
import type { MessageKey } from "../../i18n/messages";

export function WorkflowLearningPanel(props: {
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
