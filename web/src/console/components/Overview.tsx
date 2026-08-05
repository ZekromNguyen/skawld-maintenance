import { useI18n } from "../../i18n/I18nProvider";
import type { Asset, Incident } from "../../types";
import { Metric } from "./Metric";
import { IncidentTable } from "./IncidentTable";

export function Overview(props: {
  assets: Asset[];
  incidents: Incident[];
  openCount: number;
  criticalAssetCount: number;
  onOpenIncidents: () => void;
}) {
  const { t } = useI18n();
  return (
    <>
      <section className="metrics" aria-label={t("overview.operationalStatus")}>
        <Metric label={t("overview.registeredAssets")} value={props.assets.length} detail={t("overview.nativeExternal")} />
        <Metric label={t("overview.openIncidents")} value={props.openCount} detail={t("overview.requireAttention")} accent />
        <Metric label={t("overview.criticalAssets")} value={props.criticalAssetCount} detail={t("overview.criticalityA")} />
        <Metric label={t("overview.unsafeAiActions")} value={0} detail={t("overview.safetyBoundaryEnforced")} safe />
      </section>
      <section className="panel">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">{t("overview.priorityQueue")}</span>
            <h2>{t("overview.activeIncidents")}</h2>
          </div>
          <button className="secondary-button" onClick={props.onOpenIncidents}>{t("overview.openWorkbench")}</button>
        </div>
        <IncidentTable incidents={props.incidents.slice(0, 8)} />
      </section>
    </>
  );
}
