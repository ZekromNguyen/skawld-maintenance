import { useI18n } from "../../i18n/I18nProvider";
import { StatusBadge } from "../ui/StatusBadge";
import { riskTone, stepStateTone } from "../labels";
import type { Step } from "../../types";

/**
 * StepRail: LOTO/safety-gated step list for the execution workbench.
 * Blocked steps keep their reason readable (no disabled row swallowing
 * focus); completed steps show a marker instead of a re-complete click.
 */
export function StepRail({
  steps,
  onToggle,
  canWrite
}: {
  steps: Step[];
  onToggle: (step: Step) => void;
  canWrite: boolean;
}) {
  const { t } = useI18n();
  const risk = (level: string) => riskTone(level);
  return (
    <div className="panel">
      <div className="panel-heading">
        <h2>{t("execution.steps")}</h2>
        <span className="count">{steps.length}</span>
      </div>
      <div className="step-rail">
        {steps.map((step) => {
          const tone = risk(step.risk_level);
          const actionable =
            canWrite && step.state !== "COMPLETED" && step.state !== "BLOCKED";
          return (
            <div
              key={step.id}
              className={`step-row${step.state === "COMPLETED" ? " completed" : ""}${step.state === "BLOCKED" ? " blocked" : ""}`}
            >
              <span
                className="step-dot"
                data-state={step.state}
                aria-hidden="true"
              />
              <div className="step-body">
                <strong>{step.title}</strong>
                {step.required_prerequisite ? (
                  <small className="prerequisite">
                    {t("execution.prerequisite")} {step.required_prerequisite}
                  </small>
                ) : null}
                {step.state === "BLOCKED" && step.blocked_reason ? (
                  <p className="blocked-reason">{step.blocked_reason}</p>
                ) : null}
              </div>
              {tone ? (
                <StatusBadge tone={tone} label={step.risk_level} />
              ) : null}
              <StatusBadge tone={stepStateTone(step.state)} label={step.state.replace("_", " ")} />
              {actionable ? (
                <button className="step-action" onClick={() => onToggle(step)}>
                  {t("execution.complete")}
                </button>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
