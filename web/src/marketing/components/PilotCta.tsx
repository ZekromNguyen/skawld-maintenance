import { Reveal } from "./shared/Reveal";
import { Button } from "./shared/Button";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * PilotCta: full-width conversion band. Single CTA intent ("Book a pilot"),
 * consistent with the header and hero. No duplicate labels on the page.
 */
export function PilotCta() {
  const { t } = useI18n();
  return (
    <section
      style={{
        position: "relative",
        paddingBlock: "clamp(72px, 9vw, 120px)",
        borderTop: "1px solid var(--line)",
        background:
          "linear-gradient(180deg, var(--bg) 0%, color-mix(in srgb, var(--brand) 6%, var(--bg)) 100%)",
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
            "radial-gradient(ellipse 70% 60% at 50% 100%, black 25%, transparent 75%)",
          WebkitMaskImage:
            "radial-gradient(ellipse 70% 60% at 50% 100%, black 25%, transparent 75%)",
        }}
      />
      <div className="container" style={{ position: "relative" }}>
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
              {t("landing.pilot.title")}
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
              {t("landing.pilot.body")}
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
              <Button href="/auth/login">{t("landing.bookPilot")}</Button>
              <Button href="/openapi.yaml" variant="ghost">
                {t("landing.pilot.readApiSpec")}
              </Button>
            </div>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
