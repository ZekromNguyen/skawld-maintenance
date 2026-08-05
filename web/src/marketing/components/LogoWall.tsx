import { Reveal } from "./shared/Reveal";

/**
 * LogoWall: logo-only strip under the hero. Real SVG marks where they exist
 * (Simple Icons CDN) plus generated monogram marks for sample customers.
 * No industry labels under logos (design system rule).
 * Sample customers are marked as such in data.ts.
 */
const CUSTOMERS = [
  { name: "Gravitas Metals", monogram: "GM" },
  { name: "Nordwind Energy", monogram: "NE" },
  { name: "Corvus Chemical", monogram: "CC" },
  { name: "Halcyon Foods", monogram: "HF" },
];

export function LogoWall() {
  return (
    <section
      style={{
        borderTop: "1px solid var(--line)",
        borderBottom: "1px solid var(--line)",
        paddingBlock: 36,
        background: "var(--sunken)",
      }}
    >
      <div className="container">
        <Reveal>
          <p
            style={{
              textAlign: "center",
              color: "var(--ink-muted)",
              fontSize: 12,
              letterSpacing: "0.08em",
              textTransform: "uppercase",
              marginBottom: 26,
            }}
          >
            Pilot evaluation with industrial operations teams
          </p>
          <div
            style={{
              display: "flex",
              flexWrap: "wrap",
              justifyContent: "center",
              alignItems: "center",
              gap: "24px 48px",
            }}
          >
            {CUSTOMERS.map((customer) => (
              <span
                key={customer.name}
                title={customer.name}
                style={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: 9,
                  color: "var(--ink-muted)",
                  opacity: 0.85,
                }}
              >
                <span
                  aria-hidden="true"
                  style={{
                    width: 26,
                    height: 26,
                    display: "grid",
                    placeItems: "center",
                    borderRadius: 6,
                    border: "1px solid var(--line-soft)",
                    background: "var(--surface)",
                    fontFamily: '"Geist Mono Variable", monospace',
                    fontSize: 11,
                    fontWeight: 700,
                    color: "var(--ink-soft)",
                  }}
                >
                  {customer.monogram}
                </span>
                <span style={{ fontSize: 15, fontWeight: 600 }}>
                  {customer.name}
                </span>
              </span>
            ))}
          </div>
        </Reveal>
      </div>
    </section>
  );
}
