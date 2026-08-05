import { useState } from "react";
import { Check } from "@phosphor-icons/react";
import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";

/**
 * UseCases: accessible tabs. Keyboard-driven (roving tabindex + arrow keys
 * would be ideal; buttons with aria-selected + role=tablist are the base).
 */
const USE_CASES = [
  {
    id: "reliability",
    label: "Reliability teams",
    title: "Keep the knowledge when the expert leaves",
    body: "Senior technicians retire and take decades of judgment with them. Skawld captures that judgment as reviewed workflows the whole team can follow, with the safety gates intact.",
    points: [
      "Turn tacit expertise into published, reviewable workflows",
      "Evidence-backed assistance reduces guesswork on unfamiliar equipment",
      "Corrections are captured, so the guidance gets better over time",
    ],
  },
  {
    id: "operations",
    label: "Plant operations",
    title: "Consistent execution, shift after shift",
    body: "Every shift inherits a different level of experience. Skawld gives technicians the same evidence-backed steps, measurements, and handover structure regardless of who is on the floor.",
    points: [
      "Structured handovers surface open incidents and safety concerns",
      "LOTO gates and supervisor sign-off stay in the workflow",
      "Reports record what actually happened, not what was assumed",
    ],
  },
  {
    id: "enterprise",
    label: "Enterprise rollout",
    title: "Self-hosted, auditable, standards-based",
    body: "Your data plane, your identity provider, your evidence store. Skawld runs inside your infrastructure and integrates with the systems you already operate.",
    points: [
      "OIDC SSO, role-scoped permissions, and full audit trails",
      "S3-compatible object storage and PostgreSQL, self-hosted",
      "SDK contract and OpenAPI surface for internal integration",
    ],
  },
];

export function UseCases() {
  const [active, setActive] = useState(0);
  const current = USE_CASES[active];

  return (
    <Section
      id="use-cases"
      eyebrow="Use cases"
      title="One platform, three operating realities"
    >
      <div style={{ marginTop: 40 }}>
        <div
          role="tablist"
          aria-label="Use cases"
          style={{
            display: "flex",
            gap: 8,
            flexWrap: "wrap",
            borderBottom: "1px solid var(--line)",
            paddingBottom: 0,
          }}
        >
          {USE_CASES.map((useCase, index) => (
            <button
              key={useCase.id}
              role="tab"
              id={`use-case-tab-${useCase.id}`}
              aria-selected={active === index}
              aria-controls={`use-case-panel-${useCase.id}`}
              onClick={() => setActive(index)}
              style={{
                appearance: "none",
                border: "none",
                borderBottom:
                  active === index
                    ? "2px solid var(--brand)"
                    : "2px solid transparent",
                background: "transparent",
                color: active === index ? "var(--ink)" : "var(--ink-muted)",
                padding: "12px 16px",
                fontSize: 14,
                fontWeight: 600,
                cursor: "pointer",
                marginBottom: -1,
              }}
            >
              {useCase.label}
            </button>
          ))}
        </div>
        <Reveal key={current.id}>
          <div
            role="tabpanel"
            id={`use-case-panel-${current.id}`}
            aria-labelledby={`use-case-tab-${current.id}`}
            style={{
              paddingTop: 36,
              display: "grid",
              gridTemplateColumns: "1fr",
              gap: 24,
            }}
          >
            <div>
              <h3 style={{ fontSize: 24, fontWeight: 700, color: "var(--ink)", margin: 0 }}>
                {current.title}
              </h3>
              <p
                style={{
                  marginTop: 14,
                  fontSize: 15.5,
                  lineHeight: 1.65,
                  color: "var(--ink-soft)",
                  maxWidth: "58ch",
                }}
              >
                {current.body}
              </p>
            </div>
            <ul
              style={{
                listStyle: "none",
                margin: 0,
                padding: 0,
                display: "grid",
                gap: 12,
              }}
            >
              {current.points.map((point) => (
                <li
                  key={point}
                  style={{
                    display: "flex",
                    gap: 12,
                    alignItems: "flex-start",
                    fontSize: 14.5,
                    color: "var(--ink)",
                    lineHeight: 1.5,
                  }}
                >
                  <span
                    aria-hidden="true"
                    style={{
                      width: 20,
                      height: 20,
                      borderRadius: "50%",
                      background: "color-mix(in srgb, var(--brand) 14%, transparent)",
                      color: "var(--brand)",
                      display: "grid",
                      placeItems: "center",
                      flexShrink: 0,
                      marginTop: 1,
                    }}
                  >
                    <Check size={12} weight="bold" />
                  </span>
                  {point}
                </li>
              ))}
            </ul>
          </div>
        </Reveal>
      </div>
    </Section>
  );
}
