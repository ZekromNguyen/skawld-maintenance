import { useI18n } from "../../i18n/I18nProvider";
import type { Step } from "../../types";

/**
 * StepRail: LOTO/safety-gated step list for the execution workbench.
 */
export function StepRail({
  steps,
  onToggle
}: {
  steps: Step[];
  onToggle: (step: Step) => void;
}) {
  const { t } = useI18n();
  return (
    <div className="panel">
      <div className="panel-heading"><h2>{t("execution.steps")}</h2></div>
      <div>
        {steps.map((step) => (
          <button
            key={step.id}
            type="button"
            className="step-row"
            disabled={step.state === "BLOCKED"}
            onClick={() => onToggle(step)}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 10,
              width: "100%",
              padding: "12px 16px",
              border: 0,
              borderBottom: "1px solid var(--line)",
              background: "transparent",
              color: "inherit",
              textAlign: "left",
              cursor: step.state === "BLOCKED" ? "not-allowed" : "pointer"
            }}
          >
            <span
              aria-hidden="true"
              style={{
                width: 10,
                height: 10,
                borderRadius: "50%",
                background:
                  step.state === "COMPLETED"
                    ? "var(--brand)"
                    : step.state === "BLOCKED"
                      ? "var(--red)"
                      : "var(--line-soft)",
                flexShrink: 0
              }}
            />
            <span style={{ flex: 1, minWidth: 0 }}>
              <strong>{step.title}</strong>
              {step.required_prerequisite ? (
                <small style={{ display: "block", color: "var(--ink-muted)" }}>
                  {t("execution.prerequisite")} {step.required_prerequisite}
                </small>
              ) : null}
            </span>
            {step.risk_level !== "INFORMATIONAL" && (
              <span className="severity high">{step.risk_level}</span>
            )}
            {step.state === "BLOCKED" && step.blocked_reason && (
              <span className="state-badge">{step.blocked_reason}</span>
            )}
            <span className="state-badge">{step.state}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
