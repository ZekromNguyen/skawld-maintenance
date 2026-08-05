import { useI18n } from "../../i18n/I18nProvider";
import type { Principal } from "../../types";

/**
 * Topbar: page title + operator presence. Rendered by each page.
 */
export function Topbar({ title, principal }: { title: string; principal?: Principal }) {
  const { t } = useI18n();
  return (
    <header className="topbar">
      <div>
        <span className="eyebrow">{t("topbar.maintenanceOps")}</span>
        <h1>{title}</h1>
      </div>
      <div className="operator">
        <span className="presence" />
        <span>
          <strong>{principal?.display_name ?? t("topbar.connecting")}</strong>
          <small>
            {principal?.site_ids.length
              ? t("topbar.siteScope", { count: principal.site_ids.length })
              : t("topbar.allSiteScope")}
          </small>
        </span>
      </div>
    </header>
  );
}
