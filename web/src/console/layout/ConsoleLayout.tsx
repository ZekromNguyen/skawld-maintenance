import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { Breadcrumbs, type TrailItem } from "./Breadcrumbs";

/**
 * ConsoleLayout: shell for the routed console. Pages render their own
 * Topbar (title + actions); the layout provides the sidebar, breadcrumb
 * slot, and the routed content outlet.
 */
export function ConsoleLayout({ trail }: { trail?: TrailItem[] }) {
  return (
    <div className="app-shell">
      <Sidebar />
      <main style={{ minWidth: 0, padding: "0 28px 40px" }}>
        {trail && trail.length > 0 ? <Breadcrumbs trail={trail} /> : null}
        <Outlet />
      </main>
    </div>
  );
}
