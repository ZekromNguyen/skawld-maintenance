import { Link, useLocation } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import type { Locale, MessageKey } from "../../i18n/messages";
import type { Principal } from "../../types";

type NavItem = { to: string; key: MessageKey; permission?: string };

const SECTIONS: Array<{ heading: MessageKey; items: NavItem[] }> = [
  {
    heading: "sidebar.operations",
    items: [
      { to: "/", key: "nav.overview" },
      { to: "/incidents", key: "nav.incidents" },
      { to: "/handovers", key: "nav.handover" }
    ]
  },
  {
    heading: "sidebar.knowledge",
    items: [
      { to: "/knowledge", key: "nav.knowledge" },
      { to: "/search", key: "nav.search" }
    ]
  },
  {
    heading: "sidebar.records",
    items: [{ to: "/reports", key: "nav.reports", permission: "report:write" }]
  },
  {
    heading: "sidebar.learning",
    items: [
      { to: "/demonstrations", key: "nav.demonstrations" },
      { to: "/workflows", key: "nav.workflows" }
    ]
  },
  {
    heading: "sidebar.quality",
    items: [{ to: "/quality", key: "nav.quality" }]
  }
];

/**
 * Sidebar: role-aware grouped navigation. Sections and items are hidden for
 * principals lacking the required permission (report:write for Records).
 */
export function Sidebar({ principal }: { principal?: Principal }) {
  const { t, locale, setLocale } = useI18n();
  const location = useLocation();
  const has = (permission?: string) =>
    permission ? (principal?.permissions.includes(permission) ?? false) : true;

  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">S</span>
        <span>
          <strong>Skawld</strong>
          <small>Maintenance Intelligence</small>
        </span>
      </div>
      <nav aria-label={t("nav.overview")}>
        {SECTIONS.map((section) => {
          const visible = section.items.filter((item) => has(item.permission));
          if (visible.length === 0) return null;
          return (
            <div key={section.heading}>
              <span className="nav-section">{t(section.heading)}</span>
              {visible.map((item) => (
                <Link
                  key={item.to}
                  to={item.to}
                  className={`nav-item${location.pathname === item.to ? " active" : ""}`}
                >
                  <span className="nav-indicator" />
                  {t(item.key)}
                </Link>
              ))}
            </div>
          );
        })}
      </nav>
      <label className="lang-switch">
        <span className="eyebrow">{t("lang.label")}</span>
        <select value={locale} onChange={(event) => setLocale(event.target.value as Locale)}>
          <option value="en">English</option>
          <option value="vi">Tiếng Việt</option>
        </select>
      </label>
      <div className="safety-boundary">
        <span className="eyebrow">{t("sidebar.safetyBoundary")}</span>
        <strong>{t("sidebar.advisoryOnly")}</strong>
        <p>{t("sidebar.noControl")}</p>
      </div>
      <button className="logout-button" onClick={() => signOut()}>
        {t("nav.signOut")}
      </button>
    </aside>
  );
}

// Submit a POST form so the browser follows the full 303 -> Keycloak -> SPA
// redirect chain natively. fetch-based logout cannot cross the Keycloak
// redirect without CORS, leaving the page frozen until the fallback fires.
function signOut() {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/auth/logout";
  form.style.display = "none";
  document.body.appendChild(form);
  form.submit();
}
