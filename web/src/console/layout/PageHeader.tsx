import type { ReactNode } from "react";
import { useNavigate } from "react-router";
import { useI18n } from "../../i18n/I18nProvider";
import type { Principal } from "../../types";
import { Breadcrumbs } from "./Breadcrumbs";
import { SiteSwitcher } from "./SiteSwitcher";
import { usePageTrail } from "./PageTrail";
import { useSite } from "../state/SiteContext";

export function PageHeader({
  title,
  actions,
  principal,
}: {
  title: string;
  actions?: ReactNode;
  principal?: Principal;
}) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const trail = usePageTrail();
  const { siteId, setSiteId } = useSite();

  return (
    <header className="page-header">
      {trail.length > 0 ? <Breadcrumbs trail={trail} /> : null}
      <div className="page-header-row">
        <h1>{title}</h1>
        <div className="page-header-right">
          {actions}
          <SiteSwitcher siteIds={principal?.site_ids ?? []} value={siteId} onChange={setSiteId} />
          <form
            role="search"
            onSubmit={(event) => {
              event.preventDefault();
              const input = event.currentTarget.elements.namedItem("q") as HTMLInputElement;
              const query = input.value.trim();
              if (query) navigate(`/search?q=${encodeURIComponent(query)}`);
            }}
          >
            <input
              name="q"
              aria-label={t("topbar.searchPlaceholder")}
              placeholder={t("topbar.searchPlaceholder")}
              className="global-search"
            />
          </form>
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
        </div>
      </div>
    </header>
  );
}
