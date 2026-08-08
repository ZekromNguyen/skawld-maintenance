import { Outlet } from "react-router";
import { Sidebar } from "./Sidebar";
import { GlobalBar } from "./GlobalBar";
import { usePrincipal } from "../usePrincipal";
import { SiteProvider } from "../state/SiteContext";

/**
 * ConsoleLayout: shell for the routed console. Renders the skip link,
 * role-aware sidebar, persistent GlobalBar (command palette, search,
 * site/theme/language), and a site-scoped outlet.
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
        <div className="app-main">
          <GlobalBar principal={principal} />
          <main id="main-content" style={{ minWidth: 0, padding: "0 28px 40px" }}>
            <Outlet />
          </main>
        </div>
      </div>
    </SiteProvider>
  );
}
