import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * Integrations: compact card grid. Self-contained monogram tiles (text
 * letters, no hand-rolled SVG paths) instead of CDN icons, so nothing
 * depends on a third-party host at render time.
 */
const INTEGRATIONS: Array<{ name: string; href: string; initial: string; color: string }> = [
  { name: "Keycloak", href: "https://www.keycloak.org", initial: "K", color: "#008aaa" },
  { name: "PostgreSQL", href: "https://www.postgresql.org", initial: "P", color: "#336791" },
  { name: "S3 / MinIO", href: "https://min.io", initial: "M", color: "#c72e49" },
  { name: "GitHub", href: "https://github.com", initial: "G", color: "#f0f6fc" },
  { name: "Slack", href: "https://slack.com", initial: "S", color: "#611f69" },
  { name: "Docker", href: "https://www.docker.com", initial: "D", color: "#2496ED" },
];

function IntegrationCard({
  name,
  href,
  initial,
  color,
}: {
  name: string;
  href: string;
  initial: string;
  color: string;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      title={name}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        border: "1px solid var(--line)",
        borderRadius: 16,
        padding: "18px 20px",
        background: "var(--surface)",
        textDecoration: "none",
        transition: "border-color 0.15s ease, background-color 0.15s ease",
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = "var(--brand-dim)";
        e.currentTarget.style.background = "var(--raised)";
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = "var(--line)";
        e.currentTarget.style.background = "var(--surface)";
      }}
    >
      <span
        aria-hidden="true"
        style={{
          width: 28,
          height: 28,
          display: "grid",
          placeItems: "center",
          borderRadius: 8,
          border: "1px solid",
          borderColor: `color-mix(in srgb, ${color} 45%, transparent)`,
          background: `color-mix(in srgb, ${color} 10%, var(--surface))`,
          fontFamily: '"Geist Mono Variable", monospace',
          fontSize: 12,
          fontWeight: 800,
          color: "var(--ink)",
          flexShrink: 0,
        }}
      >
        {initial}
      </span>
      <span style={{ fontSize: 14, fontWeight: 600, color: "var(--ink)" }}>
        {name}
      </span>
    </a>
  );
}

export function Integrations() {
  const { t } = useI18n();
  return (
    <Section
      id="integrations"
      title={t("landing.integrations.title")}
      lead={t("landing.integrations.lead")}
      className=""
    >
      <Reveal>
        <div className="card-grid">
          {INTEGRATIONS.map((integration) => (
            <IntegrationCard key={integration.name} {...integration} />
          ))}
        </div>
      </Reveal>
    </Section>
  );
}
