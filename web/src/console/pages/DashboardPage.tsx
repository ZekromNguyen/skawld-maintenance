import { useNavigate } from "react-router-dom";
import { WarningCircle, ClipboardText, ArrowsLeftRight } from "@phosphor-icons/react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";
import { Topbar } from "../layout/Topbar";
import { Overview } from "../components/Overview";
import { ShortcutCard } from "../components/ShortcutCard";

/**
 * DashboardPage: role-aware landing. Shortcut row branches on
 * principal.permissions; Overview is the shared main content.
 */
export function DashboardPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const incidents = useApi(() => api.incidents());
  const assets = useApi(() => api.assets());

  const loading = !principal || incidents.loading || assets.loading;
  const incidentItems = incidents.data?.items ?? [];
  const assetItems = assets.data?.items ?? [];
  const openIncidents = incidentItems.filter((incident) => incident.state !== "RESOLVED");
  const criticalAssets = assetItems.filter((asset) => asset.criticality?.rating === "A");
  const perms = principal?.permissions ?? [];

  const shortcuts: Array<{ key: MessageKey; icon: typeof WarningCircle; to: string }> = [];
  if (perms.includes("report:approve") || perms.includes("report:write")) {
    shortcuts.push({ key: "dashboard.openIncidents", icon: WarningCircle, to: "/incidents" });
    shortcuts.push({ key: "dashboard.pendingReports", icon: ClipboardText, to: "/reports" });
  }
  if (perms.includes("execution:write")) {
    shortcuts.push({ key: "dashboard.assignedExecutions", icon: ClipboardText, to: "/incidents" });
  }
  if (perms.includes("handover:accept")) {
    shortcuts.push({ key: "dashboard.pendingHandovers", icon: ArrowsLeftRight, to: "/handovers" });
  }

  return (
    <section>
      <Topbar title={t("nav.overview")} principal={principal} />
      {incidents.error && (
        <div className="toast-error" role="alert">{incidents.error}</div>
      )}
      {loading ? (
        <div style={{ display: "grid", gap: 12 }}>
          <div className="skeleton" style={{ height: 64 }} />
          <div className="skeleton" style={{ height: 108 }} />
          <div className="skeleton" style={{ height: 220 }} />
        </div>
      ) : (
        <>
          {shortcuts.length > 0 && (
            <div className="shortcut-row">
              {shortcuts.map((shortcut) => (
                <ShortcutCard
                  key={shortcut.key}
                  icon={shortcut.icon}
                  label={t(shortcut.key)}
                  detail={t("dashboard.tapToOpen")}
                  to={shortcut.to}
                />
              ))}
            </div>
          )}
          <Overview
            assets={assetItems}
            incidents={incidentItems}
            openCount={openIncidents.length}
            criticalAssetCount={criticalAssets.length}
            onOpenIncidents={() => navigate("/incidents")}
          />
        </>
      )}
    </section>
  );
}
