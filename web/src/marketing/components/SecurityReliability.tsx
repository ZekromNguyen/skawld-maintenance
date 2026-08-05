import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { MonoStat } from "./shared/MonoStat";

/**
 * SecurityReliability: 2-col stat cards with display numbers.
 * Claims are real repo capabilities (OIDC, role scoping, SHA-256 integrity,
 * audit events, offline-first). No invented precision.
 */
export function SecurityReliability() {
  return (
    <Section
      id="security"
      eyebrow="Security"
      title="Built for plants, not for demos"
      lead="Industrial assistance has to earn trust. Skawld is designed around evidence integrity, human authority, and auditability from day one."
    >
      <Reveal>
        <div className="card-grid">
          <MonoStat
            value="OIDC"
            label="Standards-based identity"
            detail="Keycloak or any OIDC provider. Per-role permissions scoped to organizations and sites."
          />
          <MonoStat
            value="SHA-256"
            label="Evidence integrity"
            detail="Attachments, revisions, and retrieved content are checksum-verified end to end."
          />
          <MonoStat
            value="Advisory"
            label="Human in control"
            detail="Every recommendation requires human confirmation. The system advises; your people decide."
          />
          <MonoStat
            value="Audit"
            label="Corrections and provenance"
            detail="Every decision, correction, and demonstration carries provenance you can follow back to the source."
          />
          <MonoStat
            value="Offline"
            label="Works in the field"
            detail="The technician client is offline-first with local persistence; connectivity returns, data reconciles."
          />
          <MonoStat
            value="Self-host"
            label="Your data plane"
            detail="PostgreSQL, S3-compatible storage, and your chosen models run inside your infrastructure."
          />
        </div>
      </Reveal>
    </Section>
  );
}
