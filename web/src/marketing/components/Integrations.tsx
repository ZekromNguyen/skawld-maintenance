import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";

/**
 * Integrations: compact card grid. Real marks where they exist
 * (Simple Icons CDN) and monogram tiles for the platform surface names.
 */
const INTEGRATIONS = [
  { name: "Keycloak", href: "https://www.keycloak.org", slug: "keycloak" },
  { name: "PostgreSQL", href: "https://www.postgresql.org", slug: "postgresql" },
  { name: "S3 / MinIO", href: "https://min.io", slug: "minio" },
  { name: "GitHub", href: "https://github.com", slug: "github" },
  { name: "Slack", href: "https://slack.com", slug: "slack" },
  { name: "Docker", href: "https://www.docker.com", slug: "docker" },
];

function IntegrationCard({
  name,
  href,
  slug,
}: {
  name: string;
  href: string;
  slug: string;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
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
      <img
        src={`https://cdn.simpleicons.org/${slug}/9fb0a4`}
        alt=""
        width={22}
        height={22}
        loading="lazy"
        style={{ flexShrink: 0 }}
      />
      <span style={{ fontSize: 14, fontWeight: 600, color: "var(--ink)" }}>
        {name}
      </span>
    </a>
  );
}

export function Integrations() {
  return (
    <Section
      id="integrations"
      title="Works with the stack you already run"
      lead="Self-hosted, standards-based, and deliberately boring about infrastructure: OIDC for identity, PostgreSQL for truth, S3-compatible object storage for evidence."
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
