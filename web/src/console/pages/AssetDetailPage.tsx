import { useParams } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { Breadcrumbs } from "../layout/Breadcrumbs";

/**
 * AssetDetailPage: asset identity, criticality, and linked history.
 */
export function AssetDetailPage() {
  const { assetId } = useParams<{ assetId: string }>();
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const asset = useApi(() => api.asset(assetId!));
  const canApprove = principal?.permissions.includes("asset:criticality:approve") ?? false;

  if (asset.error) {
    return <div className="toast-error" role="alert">{asset.error}</div>;
  }
  if (asset.loading || !asset.data) {
    return <div className="skeleton" style={{ height: 220 }} />;
  }
  const value = asset.data;
  return (
    <section>
      <Breadcrumbs
        trail={[
          { label: t("nav.assets"), to: "/assets" },
          { label: value.tag }
        ]}
      />
      <Topbar title={value.tag} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <h2>{value.name}</h2>
            <span className="state-badge">{value.status}</span>
          </div>
          <div
            style={{
              padding: 16,
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit,minmax(160px,1fr))",
              gap: 12
            }}
          >
            <div><small>{t("assets.class")}</small><div className="strong">{value.class}</div></div>
            {value.manufacturer && <div><small>{t("assets.manufacturer")}</small><div className="strong">{value.manufacturer}</div></div>}
            {value.model && <div><small>{t("assets.model")}</small><div className="strong">{value.model}</div></div>}
            <div><small>{t("assets.authority")}</small><div className="strong">{value.source_of_truth}</div></div>
          </div>
        </div>
        {value.criticality && (
          <div className="panel">
            <div className="panel-heading"><h2>{t("asset.criticality")}</h2></div>
            <div style={{ padding: 16, display: "grid", gap: 10 }}>
              <div>
                {t("assets.criticality")}{" "}
                <span className={`criticality rating-${value.criticality.rating}`}>
                  {value.criticality.rating}
                </span>
              </div>
              <div>{t("asset.safetyImpact")} <span className="mono">{value.criticality.safety_impact}</span></div>
              <div>{t("asset.productionImpact")} <span className="mono">{value.criticality.production_impact}</span></div>
              <p>{value.criticality.rationale}</p>
            </div>
          </div>
        )}
        {canApprove && (
          <button
            className="primary-button"
            onClick={() => void api.approveAssetCriticality(value.id)}
          >
            {t("asset.approveCriticality")}
          </button>
        )}
      </div>
    </section>
  );
}
