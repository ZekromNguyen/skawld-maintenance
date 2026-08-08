import { useMemo, useState } from "react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { Tabs, tabPanelId } from "../ui/Tabs";
import type { MonitoringMetric } from "../../types";
import { DataTable, type Column } from "../ui/DataTable";
import { EmptyState } from "../ui/EmptyState";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { StatusCard } from "../components/monitoring/StatusCard";
import { ThresholdEditor } from "../components/monitoring/ThresholdEditor";
import { toCSV } from "../components/monitoring/Sparkline";


const TABS = [
  { id: "overview", labelKey: "monitoring.overview" },
  { id: "system", labelKey: "monitoring.system" },
  { id: "alerts", labelKey: "monitoring.alerts" },
  { id: "thresholds", labelKey: "monitoring.thresholds" },
] as const;

type TabID = (typeof TABS)[number]["id"];

/**
 * MonitoringPage: operational and system monitors with configurable
 * thresholds. Overview/System show status cards, Alerts a filterable table,
 * Thresholds the editor (gated on monitoring:configure).
 */
export function MonitoringPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const [tab, setTab] = useState<TabID>("overview");
  const [editing, setEditing] = useState(false);

  const summary = useQuery(() => api.monitoringSummary());
  const alerts = useQuery(() => api.monitoringAlerts());

  const history = useQuery(
    () => {
      const entries = summary.data?.items ?? [];
      return Promise.all(
        entries.map((entry) =>
          api
            .monitoringMetrics(entry.metric_key)
            .then((page) => ({ key: entry.metric_key, rows: page.items }))
            .catch(() => ({ key: entry.metric_key, rows: [] })),
        ),
      );
    },
    [summary.data],
  );

  const saveThreshold = useCommand(
    (value: Parameters<typeof api.setMonitoringThreshold>[0]) =>
      api.setMonitoringThreshold(value),
    {
      successMessage: t("monitoring.thresholdSaved"),
      onSuccess: () => {
        setEditing(false);
        void summary.refetch();
      },
    },
  );

  const canConfigure = principal?.permissions.includes("monitoring:configure") ?? false;
  const entries = summary.data?.items ?? [];
  const historyByKey = useMemo(() => {
    const map = new Map<string, { rows: MonitoringMetric[] }>();
    for (const item of history.data ?? []) map.set(item.key, item);
    return map;
  }, [history.data]);

  const overviewEntries = entries;
  const systemEntries = entries.filter((entry) => entry.group === "system");
  const metrics = entries.map((entry) => entry.metric_key);

  const alertRows = (alerts.data?.items ?? []).map((alert) => ({
    id: String(alert.id),
    metricKey: alert.metric_key,
    state: alert.state,
    value: alert.value,
    site: alert.site_id ?? "—",
    opened: new Date(alert.opened_at).toLocaleString(),
  }));
  const alertColumns: Array<Column<(typeof alertRows)[number]>> = [
    { key: "metricKey", header: t("monitoring.metric"), render: (row) => row.metricKey },
    { key: "state", header: t("monitoring.alerts"), render: (row) => row.state },
    { key: "value", header: "value", render: (row) => row.value },
    { key: "site", header: t("monitoring.site"), render: (row) => row.site },
    { key: "opened", header: "opened_at", render: (row) => row.opened },
  ];
  const thresholdRows = metrics.map((key) => ({ id: key, key }));
  const thresholdColumns: Array<Column<(typeof thresholdRows)[number]>> = [
    { key: "key", header: t("monitoring.metric"), render: (row) => row.key },
    { key: "comparator", header: t("monitoring.comparator"), render: () => "—" },
    { key: "warn", header: t("monitoring.warnValue"), render: () => "—" },
    { key: "crit", header: t("monitoring.critValue"), render: () => "—" },
    { key: "enabled", header: t("monitoring.enabled"), render: () => "—" },
  ];

  const exportAlerts = () => {
    const rows = (alerts.data?.items ?? []).map((alert) => [
      alert.metric_key, alert.state, alert.value, alert.opened_at,
    ]);
    toCSV("monitoring-alerts.csv", ["metric_key", "state", "value", "opened_at"], rows);
  };

  return (
    <PageTrailProvider trail={[{ label: t("nav.monitoring") }]}>
      <section>
        <PageHeader
          title={t("nav.monitoring")}
          principal={principal}
          actions={
            <button
              className="secondary-button"
              disabled={summary.loading}
              onClick={() => {
                void summary.refetch();
                void alerts.refetch();
              }}
            >
              {t("monitoring.refresh")}
            </button>
          }
        />
        {summary.error ? (
          <ErrorState message={summary.error} onRetry={() => void summary.refetch()} />
        ) : summary.loading && !summary.data ? (
          <div style={{ display: "grid", gap: 12 }}>
            <Skeleton height={120} />
            <Skeleton height={200} />
          </div>
        ) : (
          <>
            <Tabs
              label={t("nav.monitoring")}
              tabs={TABS.map((item) => ({ id: item.id, label: t(item.labelKey as never) }))}
              active={tab}
              onChange={(id) => setTab(id as TabID)}
            />
            <div id={tabPanelId(t("nav.monitoring"))} className="tab-panel">
              {tab === "overview" &&
                (overviewEntries.length > 0 ? (
                  <div className="metric-grid">
                    {overviewEntries.map((entry) => (
                      <StatusCard
                        key={entry.metric_key}
                        entry={entry}
                        history={historyByKey.get(entry.metric_key)?.rows ?? []}
                      />
                    ))}
                  </div>
                ) : (
                  <EmptyState title={t("monitoring.noData")} />
                ))}
              {tab === "system" &&
                (systemEntries.length > 0 ? (
                  <div className="metric-grid">
                    {systemEntries.map((entry) => (
                      <StatusCard
                        key={entry.metric_key}
                        entry={entry}
                        history={historyByKey.get(entry.metric_key)?.rows ?? []}
                      />
                    ))}
                  </div>
                ) : (
                  <EmptyState title={t("monitoring.noData")} />
                ))}
              {tab === "alerts" && (
                <div style={{ display: "grid", gap: 12 }}>
                  <button className="secondary-button" onClick={exportAlerts}>
                    {t("monitoring.exportCsv")}
                  </button>
                  <DataTable
                    columns={alertColumns}
                    rows={alertRows}
                    rowKey={(row) => row.id}
                    emptyTitle={t("monitoring.noAlerts")}
                  />
                </div>
              )}
              {tab === "thresholds" && (
                <div style={{ display: "grid", gap: 12 }}>
                  {canConfigure && (
                    <button className="primary-button" onClick={() => setEditing(true)}>
                      {t("monitoring.configure")}
                    </button>
                  )}
                  <DataTable
                    columns={thresholdColumns}
                    rows={thresholdRows}
                    rowKey={(row) => row.id}
                    emptyTitle={t("monitoring.noData")}
                  />
                </div>
              )}
            </div>
          </>
        )}
        {canConfigure && (
          <ThresholdEditor
            open={editing}
            metrics={metrics}
            pending={saveThreshold.pending}
            onClose={() => setEditing(false)}
            onSave={(value) => void saveThreshold.run(value)}
          />
        )}
      </section>
    </PageTrailProvider>
  );
}

