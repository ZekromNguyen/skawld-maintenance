import { Link, useLocation, useNavigate } from "react-router";
import {
  House,
  WarningCircle,
  ListChecks,
  ArrowsLeftRight,
  Books,
  MagnifyingGlass,
  FileText,
  Play,
  GitBranch,
  Gauge,
  Cube,
  type Icon,
} from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";
import type { Principal } from "../../types";
import { can, focusRole, type PermissionKey } from "../permissions";
import { FOCUS_CONFIG } from "../dashboardFocus";
import { useSite } from "../state/SiteContext";

type NavItem = { to: string; key: MessageKey; icon: Icon; permission?: PermissionKey };

const PRIMARY: NavItem[] = [
  { to: "/", key: "nav.overview", icon: House },
  { to: "/incidents", key: "nav.incidents", icon: WarningCircle },
  { to: "/executions", key: "nav.executions", icon: ListChecks },
  { to: "/handovers", key: "nav.handover", icon: ArrowsLeftRight },
];

const CROSS_LINKS: NavItem[] = [
  { to: "/assets", key: "nav.assets", icon: Cube },
  { to: "/knowledge", key: "nav.knowledge", icon: Books },
  { to: "/search", key: "nav.search", icon: MagnifyingGlass },
  { to: "/reports", key: "nav.reports", icon: FileText, permission: "report:write" },
  { to: "/quality", key: "nav.quality", icon: Gauge },
  { to: "/demonstrations", key: "nav.demonstrations", icon: Play },
  { to: "/workflows", key: "nav.workflows", icon: GitBranch },
];

function isActive(pathname: string, to: string) {
  return pathname === to || pathname.startsWith(to === "/" ? "/" : `${to}/`);
}

/**
 * Sidebar: Jira-style grouped navigation. Primary holds the status-driven
 * queues; Recents lists the principal's sites; Recommended carries the
 * role's focus links; Cross-links reaches the remaining surfaces. The
 * customize footer keeps the safety boundary and sign-out.
 */
export function Sidebar({ principal }: { principal?: Principal }) {
  const { t } = useI18n();
  const location = useLocation();
  const navigate = useNavigate();
  const { setSiteId } = useSite();
  const focus = FOCUS_CONFIG[focusRole(principal)];
  const sites = principal?.site_ids ?? [];

  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">S</span>
        <span>
          <strong>Skawld</strong>
          <small>Maintenance Intelligence</small>
        </span>
      </div>
      <nav aria-label={t("nav.mainNavigation")}>
        <span className="nav-section">{t("sidebar.primary")}</span>
        {PRIMARY.map((item) => {
          const IconComponent = item.icon;
          const active = isActive(location.pathname, item.to);
          return (
            <Link
              key={item.to}
              to={item.to}
              className={`nav-item${active ? " active" : ""}`}
              aria-current={active ? "page" : undefined}
            >
              <IconComponent className="nav-icon" size={15} aria-hidden="true" />
              {t(item.key)}
            </Link>
          );
        })}

        {sites.length > 1 ? (
          <>
            <span className="nav-section">{t("sidebar.recents")}</span>
            {sites.map((site) => (
              <button
                key={site}
                type="button"
                className="nav-item"
                onClick={() => {
                  setSiteId(site);
                  navigate("/");
                }}
              >
                <span className="nav-icon mono">{site.slice(0, 2).toUpperCase()}</span>
                <span className="mono">{site}</span>
              </button>
            ))}
          </>
        ) : null}

        <span className="nav-section">{t("sidebar.recommended")}</span>
        {focus.links.map((link) => (
          <Link key={link.to} to={link.to} className="nav-item">
            <span className="nav-icon" aria-hidden="true">
              ·
            </span>
            {t(link.labelKey)}
          </Link>
        ))}

        <span className="nav-section">{t("sidebar.crossLinks")}</span>
        {CROSS_LINKS.filter((item) => can(principal, item.permission)).map((item) => {
          const IconComponent = item.icon;
          const active = isActive(location.pathname, item.to);
          return (
            <Link
              key={item.to}
              to={item.to}
              className={`nav-item${active ? " active" : ""}`}
              aria-current={active ? "page" : undefined}
            >
              <IconComponent className="nav-icon" size={15} aria-hidden="true" />
              {t(item.key)}
            </Link>
          );
        })}
      </nav>
      <div className="sidebar-footer">
        <span className="nav-section">{t("sidebar.customize")}</span>
        <div className="safety-boundary">
          <span className="eyebrow">{t("sidebar.safetyBoundary")}</span>
          <strong>{t("sidebar.advisoryOnly")}</strong>
          <p>{t("sidebar.noControl")}</p>
        </div>
        <button className="logout-button" onClick={() => signOut()}>
          {t("nav.signOut")}
        </button>
      </div>
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
