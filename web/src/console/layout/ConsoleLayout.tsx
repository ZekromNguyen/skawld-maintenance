import { Outlet } from "react-router";
import { Sidebar } from "./Sidebar";
import { CommandPalette } from "../components/CommandPalette";
import { NotificationBell } from "../components/monitoring/NotificationBell";
import { usePrincipal } from "../usePrincipal";
import { SiteProvider } from "../state/SiteContext";
import { can } from "../permissions";

/**
 * ConsoleLayout: shell for the routed console. Renders the skip link,
 * role-aware sidebar, and a site-scoped outlet. Pages provide their own
 * PageHeader (title, breadcrumb trail, actions).
 */
export function ConsoleLayout() {
  const { data: principal } = usePrincipal();
  return (
    <SiteProvider principal={principal}>
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      <div className="app-shell">
        <Sidebar principal={principal} />
        <main id="main-content" style={{ minWidth: 0, padding: "0 28px 40px" }}>
          {can(principal, "monitoring:read") ? (
            <div className="console-topbar">
              <NotificationBell />
            </div>
          ) : null}
          <Outlet />
        </main>
      </div>
      <CommandPalette principal={principal} />
    </SiteProvider>
  );
}
