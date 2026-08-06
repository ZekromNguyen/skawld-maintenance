import type { MessageKey } from "../i18n/messages";
import type { RoleFocus } from "./permissions";

export type FocusLink = { labelKey: MessageKey; to: string };
export type FocusConfig = { titleKey: MessageKey; links: FocusLink[] };

/** Per-role dashboard focus, keyed by the permission-derived RoleFocus. */
export const FOCUS_CONFIG: Record<RoleFocus, FocusConfig> = {
  admin: {
    titleKey: "dashboard.focus.admin",
    links: [
      { labelKey: "nav.quality", to: "/quality" },
      { labelKey: "dashboard.openIncidents", to: "/incidents" },
      { labelKey: "dashboard.criticalAssets", to: "/assets" },
    ],
  },
  supervisor: {
    titleKey: "dashboard.focus.supervisor",
    links: [
      { labelKey: "dashboard.openIncidents", to: "/incidents" },
      { labelKey: "dashboard.pendingReports", to: "/reports" },
      { labelKey: "nav.quality", to: "/quality" },
    ],
  },
  senior: {
    titleKey: "dashboard.focus.senior",
    links: [
      { labelKey: "dashboard.assignedExecutions", to: "/executions" },
      { labelKey: "dashboard.pendingHandovers", to: "/handovers" },
      { labelKey: "nav.knowledge", to: "/knowledge" },
    ],
  },
  technician: {
    titleKey: "dashboard.focus.technician",
    links: [
      { labelKey: "dashboard.assignedExecutions", to: "/executions" },
      { labelKey: "nav.knowledge", to: "/knowledge" },
    ],
  },
  manager: {
    titleKey: "dashboard.focus.manager",
    links: [
      { labelKey: "dashboard.pendingHandovers", to: "/handovers" },
      { labelKey: "dashboard.openIncidents", to: "/incidents" },
      { labelKey: "nav.quality", to: "/quality" },
    ],
  },
};
