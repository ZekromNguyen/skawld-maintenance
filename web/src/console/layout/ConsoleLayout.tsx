import { Outlet } from "react-router";
import { XCircle } from "@phosphor-icons/react";
import { Sidebar } from "./Sidebar";
import { GlobalBar } from "./GlobalBar";
import { usePrincipal } from "../usePrincipal";
import { SiteProvider } from "../state/SiteContext";
import { usePreview } from "../state/PreviewProvider";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * ConsoleLayout: shell for the routed console. Renders the skip link,
 * role-aware sidebar, persistent GlobalBar (command palette, search,
 * site/theme/language), a preview-role banner when active, and a
 * site-scoped outlet.
 */
export function ConsoleLayout() {
  const { data: principal } = usePrincipal();
  const { previewRole, setPreviewRole } = usePreview();
  const { t } = useI18n();
  return (
    <SiteProvider principal={principal}>
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      <div className="app-shell">
        <Sidebar principal={principal} />
        <div className="app-main">
          <GlobalBar principal={principal} />
          {previewRole ? (
            <div className="preview-banner" role="status">
              <span>{t("account.previewBanner", { role: previewRole })}</span>
              <button
                type="button"
                className="preview-banner-exit"
                onClick={() => setPreviewRole(null)}
              >
                <XCircle size={14} aria-hidden="true" />
                {t("account.exitPreview")}
              </button>
            </div>
          ) : null}
          <main id="main-content" style={{ minWidth: 0, padding: "0 28px 40px" }}>
            <Outlet />
          </main>
        </div>
      </div>
    </SiteProvider>
  );
}
