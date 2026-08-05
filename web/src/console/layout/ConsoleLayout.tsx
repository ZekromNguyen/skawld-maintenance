import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { usePrincipal } from "../usePrincipal";
import { SiteProvider } from "../state/SiteContext";

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
          <Outlet />
        </main>
      </div>
    </SiteProvider>
  );
}
