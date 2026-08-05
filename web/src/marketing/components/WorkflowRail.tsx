import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { ShieldCheck, UserCheck } from "@phosphor-icons/react";

/**
 * WorkflowRail: the architecture / workflow illustration.
 * Capture -> Review -> Publish -> Assist -> Verify, with LOTO and
 * human-confirmation gates highlighted. Content, not decoration.
 */
const STEPS = [
  {
    phase: "Capture",
    title: "Technicians demonstrate real work",
    body: "Executions and shift handovers are captured as structured domain events: decisions, measurements, corrections, and provenance.",
    gate: null,
  },
  {
    phase: "Review",
    title: "Experts review every demonstration",
    body: "Reviewed demonstrations become trusted training signal. Corrections are linked to the exact event they fix.",
    gate: "Human review required",
  },
  {
    phase: "Publish",
    title: "Reviewed workflows are published",
    body: "Compiled workflows pass safety and privilege gates before your team can follow them. Nothing is promoted automatically.",
    gate: "Safety gate",
  },
  {
    phase: "Assist",
    title: "Evidence-backed guidance on the job",
    body: "Technicians get advisory recommendations citing SOP revisions, resolved incidents, and live measurements. Requires human confirmation.",
    gate: "Human confirmation",
  },
  {
    phase: "Verify",
    title: "Outcomes feed back into the loop",
    body: "Reports and handovers record what actually happened, so the next capture starts from verified ground truth.",
    gate: null,
  },
];

export function WorkflowRail() {
  return (
    <Section
      id="workflow"
      title="From field expertise to team-wide capability"
      lead="One loop, five stages. Every stage preserves evidence, authority, and the safety boundary that makes industrial assistance trustworthy."
    >
      <div style={{ marginTop: 56, display: "grid", gap: 0 }}>
        {STEPS.map((step, index) => (
          <Reveal key={step.phase} delay={index * 60}>
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
                    {step.phase}
                  </span>
                  {step.gate ? (
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
                      {step.gate === "Safety gate" ? (
                        <ShieldCheck size={12} weight="bold" aria-hidden="true" />
                      ) : (
                        <UserCheck size={12} weight="bold" aria-hidden="true" />
                      )}
                      {step.gate}
                    </span>
                  ) : null}
                </div>
                <h3 style={{ fontSize: 19, fontWeight: 650, color: "var(--ink)", margin: 0 }}>
                  {step.title}
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
                  {step.body}
                </p>
              </div>
            </div>
          </Reveal>
        ))}
      </div>
    </Section>
  );
}
