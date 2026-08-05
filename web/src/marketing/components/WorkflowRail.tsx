import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { ShieldCheck, UserCheck } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";

/**
 * WorkflowRail: the architecture / workflow illustration.
 * Capture -> Review -> Publish -> Assist -> Verify, with LOTO and
 * human-confirmation gates highlighted. Content, not decoration.
 */
const STEPS: Array<{
  phaseKey: MessageKey;
  titleKey: MessageKey;
  bodyKey: MessageKey;
  gateKey: MessageKey | null;
}> = [
  {
    phaseKey: "landing.phase.capture",
    titleKey: "landing.workflow.step1.title",
    bodyKey: "landing.workflow.step1.body",
    gateKey: null,
  },
  {
    phaseKey: "landing.phase.review",
    titleKey: "landing.workflow.step2.title",
    bodyKey: "landing.workflow.step2.body",
    gateKey: "landing.workflow.gate.humanReview",
  },
  {
    phaseKey: "landing.phase.publish",
    titleKey: "landing.workflow.step3.title",
    bodyKey: "landing.workflow.step3.body",
    gateKey: "landing.workflow.gate.safety",
  },
  {
    phaseKey: "landing.phase.assist",
    titleKey: "landing.workflow.step4.title",
    bodyKey: "landing.workflow.step4.body",
    gateKey: "landing.workflow.gate.humanConfirmation",
  },
  {
    phaseKey: "landing.phase.verify",
    titleKey: "landing.workflow.step5.title",
    bodyKey: "landing.workflow.step5.body",
    gateKey: null,
  },
];

export function WorkflowRail() {
  const { t } = useI18n();
  return (
    <Section
      id="workflow"
      title={t("landing.workflow.title")}
      lead={t("landing.workflow.lead")}
    >
      <div style={{ marginTop: 56, display: "grid", gap: 0 }}>
        {STEPS.map((step, index) => (
          <Reveal key={step.phaseKey} delay={index * 60}>
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "auto 1fr",
                gap: "20px 24px",
                position: "relative",
                paddingBottom: index < STEPS.length - 1 ? 40 : 0,
              }}
            >
              {/* Rail */}
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                }}
              >
                <span
                  style={{
                    width: 44,
                    height: 44,
                    borderRadius: "50%",
                    border: "1px solid var(--brand-dim)",
                    background: "var(--sunken)",
                    color: "var(--brand)",
                    display: "grid",
                    placeItems: "center",
                    fontFamily: '"Geist Mono Variable", monospace',
                    fontSize: 13,
                    fontWeight: 700,
                    flexShrink: 0,
                  }}
                >
                  {String(index + 1).padStart(2, "0")}
                </span>
                {index < STEPS.length - 1 ? (
                  <span
                    aria-hidden="true"
                    style={{
                      width: 1,
                      flex: 1,
                      minHeight: 40,
                      background: "var(--line)",
                    }}
                  />
                ) : null}
              </div>
              {/* Content */}
              <div style={{ paddingBottom: 0 }}>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    flexWrap: "wrap",
                    marginBottom: 6,
                  }}
                >
                  <span
                    className="eyebrow"
                    style={{ marginBottom: 0, color: "var(--brand)" }}
                  >
                    {t(step.phaseKey)}
                  </span>
                  {step.gateKey ? (
                    <span
                      style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 5,
                        fontSize: 10.5,
                        fontFamily: '"Geist Mono Variable", monospace',
                        color: "var(--amber)",
                        border: "1px solid color-mix(in srgb, var(--amber) 45%, transparent)",
                        borderRadius: 999,
                        padding: "3px 9px",
                      }}
                    >
                      {step.gateKey === "landing.workflow.gate.safety" ? (
                        <ShieldCheck size={12} weight="bold" aria-hidden="true" />
                      ) : (
                        <UserCheck size={12} weight="bold" aria-hidden="true" />
                      )}
                      {t(step.gateKey)}
                    </span>
                  ) : null}
                </div>
                <h3 style={{ fontSize: 20, fontWeight: 650, color: "var(--ink)", margin: 0 }}>
                  {t(step.titleKey)}
                </h3>
                <p
                  style={{
                    margin: "8px 0 0",
                    fontSize: 14.5,
                    lineHeight: 1.6,
                    color: "var(--ink-soft)",
                    maxWidth: "56ch",
                  }}
                >
                  {t(step.bodyKey)}
                </p>
              </div>
            </div>
          </Reveal>
        ))}
      </div>
    </Section>
  );
}
