import { useNavigate } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { Overview } from "../components/Overview";

/**
 * DashboardPage: role-aware landing. Fetches its own data and renders the
 * migrated Overview with shortcut navigation.
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

  return (
    <section>
      <Topbar title={t("nav.overview")} principal={principal} />
      {incidents.error && (
        <div className="toast-error" role="alert">{incidents.error}</div>
      )}
      {loading ? (
        <div style={{ display: "grid", gap: 12 }}>
          <div className="skeleton" style={{ height: 108 }} />
          <div className="skeleton" style={{ height: 220 }} />
        </div>
      ) : (
        <Overview
          assets={assetItems}
          incidents={incidentItems}
          openCount={openIncidents.length}
          criticalAssetCount={criticalAssets.length}
          onOpenIncidents={() => navigate("/incidents")}
        />
      )}
    </section>
  );
}
