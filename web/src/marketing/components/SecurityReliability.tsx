import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { MonoStat } from "./shared/MonoStat";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * SecurityReliability: 2-col stat cards with display numbers.
 * Claims are real repo capabilities (OIDC, role scoping, SHA-256 integrity,
 * audit events, offline-first). No invented precision.
 */
export function SecurityReliability() {
  const { t } = useI18n();
  return (
    <Section
      id="security"
      eyebrow={t("landing.security.eyebrow")}
      title={t("landing.security.title")}
      lead={t("landing.security.lead")}
    >
      <Reveal>
        <div className="card-grid">
          <MonoStat
            value="OIDC"
            label={t("landing.security.oidc.label")}
            detail={t("landing.security.oidc.detail")}
          />
          <MonoStat
            value="SHA-256"
            label={t("landing.security.sha.label")}
            detail={t("landing.security.sha.detail")}
          />
          <MonoStat
            value="Advisory"
            label={t("landing.security.advisory.label")}
            detail={t("landing.security.advisory.detail")}
          />
          <MonoStat
            value="Audit"
            label={t("landing.security.audit.label")}
            detail={t("landing.security.audit.detail")}
          />
          <MonoStat
            value="Offline"
            label={t("landing.security.offline.label")}
            detail={t("landing.security.offline.detail")}
          />
          <MonoStat
            value="Self-host"
            label={t("landing.security.selfhost.label")}
            detail={t("landing.security.selfhost.detail")}
          />
        </div>
      </Reveal>
    </Section>
  );
}
