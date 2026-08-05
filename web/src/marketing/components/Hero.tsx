import { Reveal } from "./shared/Reveal";
import { Button } from "./shared/Button";

/**
 * InstrumentPanel: the hero visual. A live product-shaped composition of
 * real Skawld domain data (incident severity, step states, evidence,
 * LOTO gate). Not a div-fake-screenshot: these are the actual data shapes
 * the operator console renders, simplified for a marketing context.
 */
function InstrumentPanel() {
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
      aria-label="Skawld workflow panel: a pump incident execution with completed LOTO steps, an in-progress inspection step, and evidence-backed guidance."
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
        <span
          style={{
            fontSize: 12,
            fontWeight: 600,
            color: "var(--ink)",
            letterSpacing: "0.01em",
          }}
        >
          P-302 · Circulation pump inspection
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
          EXECUTION IN PROGRESS
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
            Copilot recommendation · advisory
          </div>
          <p
            style={{
              margin: 0,
              fontSize: 13,
              lineHeight: 1.55,
              color: "var(--ink)",
            }}
          >
            Rotor shaft alignment is 0.05 mm out of spec. Review the two cited
            SOP revisions before proceeding.
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
            <span>evidence · 3 sources</span>
            <span>confidence · 0.94</span>
            <span>requires human confirmation</span>
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
          {stepBar("done", "LOTO applied · energy isolated")}
          {stepBar("done", "Torque values recorded · 8.1 mm/s vibration")}
          {stepBar("gate", "LOTO verification gate · supervisor sign-off")}
          {stepBar("current", "Inspection step 4 · in progress")}
          {stepBar("pending", "Report and handover")}
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
            ["8.1", "mm/s vibration"],
            ["94", "°C bearing temp"],
            ["0.94", "evidence score"],
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
      </div>
    </div>
  );
}

/**
 * Hero: asymmetric split. Left copy (eyebrowless H1, short subtext, CTAs),
 * right instrument panel. Max 4 text elements, no version labels.
 */
export function Hero() {
  return (
    <section
      style={{
        paddingTop: "clamp(64px, 8vw, 96px)",
        paddingBottom: "clamp(48px, 6vw, 72px)",
      }}
    >
      <div className="container hero-grid">
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
              Capture how your best technicians work. Then help every
              technician work like them.
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
              Skawld turns expert work into reviewed, reusable workflows, and
              gives every technician evidence-backed guidance. Human authority
              stays in control.
            </p>
            <div
              style={{
                display: "flex",
                gap: 12,
                flexWrap: "wrap",
                alignItems: "center",
              }}
            >
              <Button href="/auth/login">Book a pilot</Button>
              <Button href="#features" variant="ghost">
                See how it works
              </Button>
            </div>
          </div>
        </Reveal>
        <Reveal delay={120}>
          <InstrumentPanel />
        </Reveal>
      </div>
    </section>
  );
}
