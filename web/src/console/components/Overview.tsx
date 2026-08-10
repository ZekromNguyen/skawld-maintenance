import { Link } from "react-router";
import { useI18n } from "../../i18n/I18nProvider";
import { AnnunciatorStrip } from "./AnnunciatorStrip";
import { Skeleton } from "../ui/Skeleton";
import type { Asset, Execution, Incident } from "../../types";
import type { FocusConfig } from "../dashboardFocus";

/**
 * Overview: dashboard signature band. The annunciator strip carries live
 * operational counts with whole-cell click-through; the role focus panel
 * links into the work a role owns. The tabbed "For you" queue lives in
 * ForYouTabs (same data, below this component).
 */
export function Overview({
  focus,
  incidents,
  executions,
  assets,
  pendingHandoverCount,
  loading,
  onRetry,
}: {
  focus: FocusConfig;
  incidents: Incident[];
  executions: Execution[];
  assets: Asset[];
  pendingHandoverCount: number;
  loading: boolean;
  error?: string;
  onRetry: () => void;
}) {
  const { t } = useI18n();
  const open = incidents.filter((incident) => incident.status !== "RESOLVED");
  const inProgress = executions.filter((execution) => execution.state === "IN_PROGRESS");
  const critical = assets.filter((asset) => asset.criticality?.rating === "A");

  if (loading) {
    return (
      <div style={{ display: "grid", gap: 16 }}>
        <Skeleton height={108} />
        <Skeleton height={64} />
      </div>
    );
  }

  return (
    <>
      <AnnunciatorStrip
        cells={[
          {
            key: "open-incidents",
            label: t("dashboard.openIncidents"),
            value: String(open.length),
            tone: "critical",
            to: "/incidents",
          },
          {
            key: "in-progress",
            label: t("dashboard.executionsInProgress"),
            value: String(inProgress.length),
            tone: "medium",
            to: "/executions",
          },
          {
            key: "critical-assets",
            label: t("dashboard.criticalAssets"),
            value: String(critical.length),
            tone: "high",
            to: "/assets",
          },
          {
            key: "pending-handovers",
            label: t("dashboard.pendingHandovers"),
            value: String(pendingHandoverCount),
            tone: "info",
            to: "/handovers",
          },
        ]}
      />

      <section className="panel" style={{ marginBottom: 20 }}>
        <div className="panel-heading">
          <h2>{t(focus.titleKey)}</h2>
        </div>
        <div className="focus-links">
          {focus.links.map((link) => (
            <Link key={link.to} to={link.to} className="secondary-button">
              {t(link.labelKey)}
            </Link>
          ))}
        </div>
      </section>
    </>
  );
}
