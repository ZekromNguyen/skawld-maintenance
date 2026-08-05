import { Reveal } from "./shared/Reveal";
import { Button } from "./shared/Button";

/**
 * PilotCta: full-width conversion band. Single CTA intent ("Book a pilot"),
 * consistent with the header and hero. No duplicate labels on the page.
 */
export function PilotCta() {
  return (
    <section
      style={{
        paddingBlock: "clamp(72px, 9vw, 120px)",
        borderTop: "1px solid var(--line)",
        background:
          "linear-gradient(180deg, var(--bg) 0%, color-mix(in srgb, var(--brand) 6%, var(--bg)) 100%)",
      }}
    >
      <div className="container">
        <Reveal>
          <div style={{ textAlign: "center", maxWidth: 640, marginInline: "auto" }}>
            <h2
              style={{
                fontSize: "clamp(28px, 4vw, 44px)",
                lineHeight: 1.05,
                letterSpacing: "-0.025em",
                fontWeight: 700,
                color: "var(--ink)",
                margin: 0,
              }}
            >
              Run a pilot on one pump line, one site, one shift
            </h2>
            <p
              style={{
                margin: "18px auto 0",
                fontSize: 16,
                lineHeight: 1.6,
                color: "var(--ink-soft)",
                maxWidth: "52ch",
              }}
            >
              See Skawld capture real expertise on your equipment, with your
              people, on your infrastructure. You keep control of the scope,
              the data, and the rollout.
            </p>
            <div
              style={{
                marginTop: 32,
                display: "flex",
                justifyContent: "center",
                gap: 12,
                flexWrap: "wrap",
              }}
            >
              <Button href="/auth/login">Book a pilot</Button>
              <Button href="/openapi.yaml" variant="ghost">
                Read the API spec
              </Button>
            </div>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
