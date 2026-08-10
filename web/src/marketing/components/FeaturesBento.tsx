import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";
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
        gap: 14,
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
      <h3 style={{ fontSize: 18, fontWeight: 700, lineHeight: 1.3, color: "var(--ink)", margin: 0 }}>
        {title}
      </h3>
      <p
        style={{
          margin: 0,
          fontSize: 14,
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
  const { t } = useI18n();
  const steps: Array<{ labelKey: MessageKey; state: "done" | "current" | "pending" }> = [
    { labelKey: "landing.phase.capture", state: "done" },
    { labelKey: "landing.phase.review", state: "done" },
    { labelKey: "landing.phase.publish", state: "current" },
    { labelKey: "landing.phase.assist", state: "pending" },
    { labelKey: "landing.phase.verify", state: "pending" },
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
        {t("landing.features.workflowLifecycle")}
      </span>
      {steps.map((step, index) => (
        <div key={step.labelKey} style={{ display: "flex", alignItems: "center", gap: 10 }}>
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
            {t(step.labelKey)}
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

const FEATURES: Array<{
  Icon: Icon;
  titleKey: MessageKey;
  bodyKey: MessageKey;
  tinted?: boolean;
}> = [
  {
    Icon: UsersThree,
    titleKey: "landing.features.captureExpertise.title",
    bodyKey: "landing.features.captureExpertise.body",
    tinted: true,
  },
  {
    Icon: SealCheck,
    titleKey: "landing.features.humanAuthority.title",
    bodyKey: "landing.features.humanAuthority.body",
  },
  {
    Icon: MagnifyingGlass,
    titleKey: "landing.features.evidenceBacked.title",
    bodyKey: "landing.features.evidenceBacked.body",
  },
  {
    Icon: FlowArrow,
    titleKey: "landing.features.reusable.title",
    bodyKey: "landing.features.reusable.body",
    tinted: true,
  },
  {
    Icon: ShieldCheck,
    titleKey: "landing.features.safetyFirst.title",
    bodyKey: "landing.features.safetyFirst.body",
  },
];

export function FeaturesBento() {
  const { t } = useI18n();
  return (
    <Section
      id="features"
      eyebrow={t("landing.features.eyebrow")}
      title={t("landing.features.title")}
      lead={t("landing.features.lead")}
    >
      <Reveal>
        <div className="bento-grid">
          {FEATURES.map((feature) => (
            <FeatureCell
              key={feature.titleKey}
              Icon={feature.Icon}
              title={t(feature.titleKey)}
              body={t(feature.bodyKey)}
              tinted={feature.tinted}
            />
          ))}
          <WorkflowPeekCell />
        </div>
      </Reveal>
    </Section>
  );
}
