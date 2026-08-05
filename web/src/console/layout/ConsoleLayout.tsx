import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { Breadcrumbs, type TrailItem } from "./Breadcrumbs";
import { usePrincipal } from "../usePrincipal";

/**
 * ConsoleLayout: shell for the routed console. Pages render their own
 * Topbar (title + actions); the layout provides the sidebar (with the
 * resolved principal for role-aware nav), breadcrumb slot, and the outlet.
 */
export function ConsoleLayout({ trail }: { trail?: TrailItem[] }) {
  const { data: principal } = usePrincipal();
  return (
    <div className="app-shell">
      <Sidebar principal={principal} />
      <main style={{ minWidth: 0, padding: "0 28px 40px" }}>
        {trail && trail.length > 0 ? <Breadcrumbs trail={trail} /> : null}
        <Outlet />
      </main>
    </div>
  );
}
