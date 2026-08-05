import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import {
  UsersThree,
  FlowArrow,
  SealCheck,
  ShieldCheck,
  MagnifyingGlass,
  type Icon,
} from "@phosphor-icons/react";

/**
 * FeaturesBento: asymmetric bento, exactly 6 cells for 6 features.
 * At least 3 cells carry real visual variation (tinted background,
 * live data composition) per the design system's bento diversity rule.
 */
function FeatureCell({
  Icon,
  title,
  body,
  tinted = false,
}: {
  Icon: Icon;
  title: string;
  body: string;
  tinted?: boolean;
}) {
  return (
    <div
      style={{
        border: "1px solid var(--line)",
        borderRadius: 16,
        padding: 26,
        background: tinted
          ? "color-mix(in srgb, var(--brand) 7%, var(--surface))"
          : "var(--surface)",
        display: "flex",
        flexDirection: "column",
        gap: 12,
      }}
    >
      <span
        aria-hidden="true"
        style={{
          width: 40,
          height: 40,
          display: "grid",
          placeItems: "center",
          borderRadius: 10,
          border: "1px solid var(--brand-dim)",
          background: "var(--sunken)",
          color: "var(--brand)",
        }}
      >
        <Icon size={20} weight="duotone" />
      </span>
      <h3 style={{ fontSize: 16.5, fontWeight: 650, color: "var(--ink)", margin: 0 }}>
        {title}
      </h3>
      <p
        style={{
          margin: 0,
          fontSize: 13.5,
          lineHeight: 1.6,
          color: "var(--ink-soft)",
        }}
      >
        {body}
      </p>
    </div>
  );
}

/** Live mini rail: a real step-state visual cell for bento diversity. */
function WorkflowPeekCell() {
  const steps = [
    { label: "Capture", state: "done" as const },
    { label: "Review", state: "done" as const },
    { label: "Publish", state: "current" as const },
    { label: "Assist", state: "pending" as const },
    { label: "Verify", state: "pending" as const },
  ];
  return (
    <div
      style={{
        border: "1px solid var(--line)",
        borderRadius: 16,
        padding: 26,
        background: "var(--raised)",
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        gap: 12,
      }}
    >
      <span
        className="eyebrow"
        style={{ marginBottom: 2 }}
      >
        Workflow lifecycle
      </span>
      {steps.map((step, index) => (
        <div key={step.label} style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              background:
                step.state === "done"
                  ? "var(--brand)"
                  : step.state === "current"
                    ? "var(--amber)"
                    : "var(--line-soft)",
              flexShrink: 0,
            }}
          />
          <span
            style={{
              fontSize: 12.5,
              fontFamily: '"Geist Mono Variable", monospace',
              color: step.state === "pending" ? "var(--ink-muted)" : "var(--ink)",
            }}
          >
            {step.label}
          </span>
          {index < steps.length - 1 ? (
            <span
              style={{
                flex: 1,
                height: 1,
                background: "var(--line)",
                marginLeft: 4,
              }}
            />
          ) : null}
        </div>
      ))}
    </div>
  );
}

const FEATURES = [
  {
    Icon: UsersThree,
    title: "Capture real expertise",
    body: "Technicians demonstrate their work; Skawld records decisions, corrections, and provenance as structured events, not free text.",
    tinted: true,
  },
  {
    Icon: SealCheck,
    title: "Human authority preserved",
    body: "Recommendations are advisory and require human confirmation. LOTO gates and supervisor sign-off stay in the loop.",
  },
  {
    Icon: MagnifyingGlass,
    title: "Evidence-backed assistance",
    body: "Every recommendation cites its sources: SOP revisions, resolved incidents, and measurements. No unsupported advice.",
  },
  {
    Icon: FlowArrow,
    title: "Reviewed, reusable workflows",
    body: "Demonstrations are compiled, reviewed, and published into workflows your team can follow, with safety gates built in.",
    tinted: true,
  },
  {
    Icon: ShieldCheck,
    title: "Safety first, by design",
    body: "Lockout-tagout verification gates, risk-leveled steps, and a strict boundary between advisory assistance and control.",
  },
];

export function FeaturesBento() {
  return (
    <Section
      id="features"
      eyebrow="Features"
      title="A copilot that learns from your own people"
      lead="Built around the workflow your technicians already trust: capture, review, publish, assist, verify. Skawld fits into it instead of replacing it."
    >
      <Reveal>
        <div className="bento-grid">
          {FEATURES.map((feature) => (
            <FeatureCell key={feature.title} {...feature} />
          ))}
          <WorkflowPeekCell />
        </div>
      </Reveal>
    </Section>
  );
}
