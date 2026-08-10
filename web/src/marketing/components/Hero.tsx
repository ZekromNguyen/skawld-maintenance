import { Reveal } from "./shared/Reveal";
import { Button } from "./shared/Button";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";

/**
 * InstrumentPanel: the hero visual. A live product-shaped composition of
 * real Skawld domain data (incident severity, step states, evidence,
 * LOTO gate). Not a div-fake-screenshot: these are the actual data shapes
 * the operator console renders, simplified for a marketing context.
 */
function InstrumentPanel() {
  const { t } = useI18n();
  const stepBar = (
    state: "done" | "current" | "pending" | "gate",
    label: string,
  ) => {
    const colors: Record<string, string> = {
      done: "var(--brand)",
      current: "var(--amber)",
      pending: "var(--line-soft)",
      gate: "var(--red)",
    };
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: "50%",
            background: colors[state],
            flexShrink: 0,
          }}
        />
        <span
          style={{
            fontSize: 12,
            color: state === "pending" ? "var(--ink-muted)" : "var(--ink-soft)",
            fontFamily: '"Geist Mono Variable", monospace',
          }}
        >
          {label}
        </span>
      </div>
    );
  };

  return (
    <div
      role="img"
      aria-label={t("landing.hero.panelAria")}
      style={{
        border: "1px solid var(--line)",
        borderRadius: 16,
        background: "var(--surface)",
        overflow: "hidden",
        boxShadow: "0 24px 80px -24px rgba(0,0,0,0.55)",
      }}
    >
      {/* Panel header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "14px 18px",
          borderBottom: "1px solid var(--line)",
        }}
      >
        <span style={{ display: "grid", gap: 2 }}>
          <span
            style={{
              fontSize: 10,
              fontFamily: '"Geist Mono Variable", monospace',
              color: "var(--brand)",
              textTransform: "uppercase",
              letterSpacing: "0.14em",
            }}
          >
            {t("landing.hero.panelLabel")}
          </span>
          <span
            style={{
              fontSize: 12,
              fontWeight: 600,
              color: "var(--ink)",
              letterSpacing: "0.01em",
            }}
          >
            {t("landing.hero.panelTitle")}
          </span>
        </span>
        <span
          style={{
            fontSize: 10,
            fontFamily: '"Geist Mono Variable", monospace',
            color: "var(--brand)",
            border: "1px solid var(--brand-dim)",
            background: "color-mix(in srgb, var(--brand) 12%, transparent)",
            padding: "3px 8px",
            borderRadius: 999,
          }}
        >
          {t("landing.hero.panelStatus")}
        </span>
      </div>

      <div style={{ padding: 18, display: "grid", gap: 16 }}>
        {/* Evidence-backed guidance card */}
        <div
          style={{
            border: "1px solid var(--brand-dim)",
            borderRadius: 12,
            background: "color-mix(in srgb, var(--brand) 8%, var(--surface))",
            padding: 14,
          }}
        >
          <div
            style={{
              fontSize: 10,
              fontFamily: '"Geist Mono Variable", monospace',
              color: "var(--brand)",
              textTransform: "uppercase",
              letterSpacing: "0.14em",
              marginBottom: 6,
            }}
          >
            {t("landing.hero.recommendationLabel")}
          </div>
          <p
            style={{
              margin: 0,
              fontSize: 13,
              lineHeight: 1.55,
              color: "var(--ink)",
            }}
          >
            {t("landing.hero.recommendationBody")}
          </p>
          <div
            style={{
              marginTop: 10,
              display: "flex",
              gap: 12,
              fontSize: 10,
              fontFamily: '"Geist Mono Variable", monospace',
              color: "var(--ink-muted)",
            }}
          >
            <span>{t("landing.hero.evidence")}</span>
            <span>{t("landing.hero.confidence")}</span>
            <span>{t("landing.hero.requiresConfirmation")}</span>
          </div>
        </div>

        {/* Step rail */}
        <div
          style={{
            border: "1px solid var(--line)",
            borderRadius: 12,
            padding: "12px 14px",
            display: "grid",
            gap: 10,
          }}
        >
          {stepBar("done", t("landing.hero.step.lotoApplied"))}
          {stepBar("done", t("landing.hero.step.torqueRecorded"))}
          {stepBar("gate", t("landing.hero.step.lotoGate"))}
          {stepBar("current", t("landing.hero.step.inspection"))}
          {stepBar("pending", t("landing.hero.step.report"))}
        </div>

        {/* Metrics row */}
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(3, 1fr)",
            gap: 10,
          }}
        >
          {[
            ["8.1", t("landing.hero.metric.vibration")],
            ["94", t("landing.hero.metric.bearingTemp")],
            ["0.94", t("landing.hero.metric.evidenceScore")],
          ].map(([value, label]) => (
            <div
              key={label}
              style={{
                border: "1px solid var(--line)",
                borderRadius: 12,
                padding: "10px 12px",
                background: "var(--sunken)",
              }}
            >
              <div
                style={{
                  fontSize: 20,
                  fontWeight: 700,
                  fontFamily: '"Geist Mono Variable", monospace',
                  color: "var(--brand)",
                }}
              >
                {value}
              </div>
              <div
                style={{
                  fontSize: 10,
                  color: "var(--ink-muted)",
                  marginTop: 2,
                }}
              >
                {label}
              </div>
            </div>
          ))}
        </div>
        <div
          style={{
            fontSize: 9.5,
            fontFamily: '"Geist Mono Variable", monospace',
            color: "var(--ink-muted)",
            borderLeft: "1px solid var(--callout-line)",
            paddingLeft: 8,
            lineHeight: 1.5,
          }}
        >
          {t("landing.hero.callout")}
        </div>
      </div>
    </div>
  );
}

const PHASES: Array<{ labelKey: MessageKey; state: "done" | "current" | "pending" }> = [
  { labelKey: "landing.phase.capture", state: "done" },
  { labelKey: "landing.phase.review", state: "done" },
  { labelKey: "landing.phase.publish", state: "current" },
  { labelKey: "landing.phase.assist", state: "pending" },
  { labelKey: "landing.phase.verify", state: "pending" },
];

/** ProcedureMeter: the page's signature. The product's real workflow states
 * on a single rail, with the executing phase lit (same grammar as the bento
 * peek cell and the workflow rail). */
function ProcedureMeter() {
  const { t } = useI18n();
  return (
    <div
      className="hero-meter"
      role="group"
      aria-label={t("landing.hero.panelAria")}
      style={{ marginTop: 40, display: "flex", alignItems: "flex-start", gap: 0 }}
    >
      {PHASES.map((phase, index) => {
        const isCurrent = phase.state === "current";
        return (
          <div
            key={phase.labelKey}
            style={{ display: "flex", alignItems: "center", gap: 0, flex: 1, minWidth: 0 }}
          >
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 6 }}>
              <span
                data-current={isCurrent}
                style={{
                  width: 12,
                  height: 12,
                  borderRadius: "50%",
                  background: isCurrent
                    ? "var(--amber)"
                    : phase.state === "done"
                      ? "var(--brand)"
                      : "var(--line-soft)",
                  flexShrink: 0,
                }}
              />
              <span
                style={{
                  fontSize: 9.5,
                  fontFamily: '"Geist Mono Variable", monospace',
                  color: isCurrent ? "var(--ink)" : phase.state === "done" ? "var(--ink-soft)" : "var(--ink-muted)",
                  textTransform: "uppercase",
                  letterSpacing: "0.08em",
                  whiteSpace: "nowrap",
                }}
              >
                {t(phase.labelKey)}
              </span>
            </div>
            {index < PHASES.length - 1 ? (
              <span
                aria-hidden="true"
                style={{ flex: 1, height: 1, background: "var(--line)", margin: "0 10px", marginTop: 6 }}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

/**
 * Hero: asymmetric split. Left copy (eyebrowless H1, short subtext, CTAs),
 * right instrument panel. Max 4 text elements, no version labels.
 */
export function Hero() {
  const { t } = useI18n();
  return (
    <section
      style={{
        position: "relative",
        paddingTop: "clamp(64px, 8vw, 96px)",
        paddingBottom: "clamp(48px, 6vw, 72px)",
      }}
    >
      <div
        aria-hidden="true"
        style={{
          position: "absolute",
          inset: 0,
          pointerEvents: "none",
          backgroundImage:
            "linear-gradient(var(--grid-line) 1px, transparent 1px), linear-gradient(90deg, var(--grid-line) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
          maskImage:
            "radial-gradient(ellipse 70% 55% at 50% 0%, black 25%, transparent 72%)",
          WebkitMaskImage:
            "radial-gradient(ellipse 70% 55% at 50% 0%, black 25%, transparent 72%)",
        }}
      />
      <div className="container hero-grid" style={{ position: "relative" }}>
        <Reveal>
          <div style={{ maxWidth: 560 }}>
            <h1
              style={{
                fontSize: "clamp(36px, 5.5vw, 60px)",
                lineHeight: 1.02,
                letterSpacing: "-0.03em",
                fontWeight: 700,
                color: "var(--ink)",
                marginBottom: 22,
              }}
            >
              {t("landing.hero.title")}
            </h1>
            <p
              style={{
                fontSize: 17,
                lineHeight: 1.6,
                color: "var(--ink-soft)",
                maxWidth: "46ch",
                marginBottom: 30,
              }}
            >
              {t("landing.hero.subtitle")}
            </p>
            <div
              style={{
                display: "flex",
                gap: 12,
                flexWrap: "wrap",
                alignItems: "center",
              }}
            >
              <Button href="/auth/login">{t("landing.bookPilot")}</Button>
              <Button href="#features" variant="ghost">
                {t("landing.hero.seeHowItWorks")}
              </Button>
            </div>
            <ProcedureMeter />
          </div>
        </Reveal>
        <Reveal delay={120}>
          <InstrumentPanel />
        </Reveal>
      </div>
    </section>
  );
}
