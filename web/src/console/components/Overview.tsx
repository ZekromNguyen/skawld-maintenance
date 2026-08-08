import { Link, useNavigate } from "react-router";
import { useI18n } from "../../i18n/I18nProvider";
import { MetricCard } from "../ui/MetricCard";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { EmptyState } from "../ui/EmptyState";
import { RelativeTime } from "../ui/RelativeTime";
import { Skeleton } from "../ui/Skeleton";
import {
  severityTone,
  severityLabelKey,
  incidentStateTone,
  incidentStateLabelKey,
  executionStateTone,
  executionStateLabelKey,
} from "../labels";
import type { DashboardSummary, Execution, Incident } from "../../types";
import type { FocusConfig } from "../dashboardFocus";

/**
 * Overview: dashboard main content. Metric cards come from the /summary
 * payload (truthful counts), a "my queue" panel for in-progress executions,
 * and the active-incident table (open incidents only, clickable rows).
 * No fabricated metrics.
 */
export function Overview({
  focus,
  summary,
  incidents,
  executions,
  pendingHandoverCount,
  loading,
  error,
  onRetry,
}: {
  focus: FocusConfig;
  summary?: DashboardSummary;
  incidents: Incident[];
  executions: Execution[];
  pendingHandoverCount: number;
  loading: boolean;
  error?: string;
  onRetry: () => void;
}) {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const open = incidents.filter((incident) => incident.state !== "RESOLVED");
  const inProgress = executions.filter((execution) => execution.state === "IN_PROGRESS");

  if (loading) {
    return (
      <div style={{ display: "grid", gap: 16 }}>
        <Skeleton height={108} />
        <Skeleton height={220} />
      </div>
    );
  }

  return (
    <>
      <section className="metrics" aria-label={t("overview.operationalStatus")}>
        <MetricCard
          label={t("dashboard.openIncidents")}
          value={String(summary?.open_incidents ?? open.length)}
          to="/incidents"
          tone="critical"
        />
        <MetricCard
          label={t("dashboard.executionsInProgress")}
          value={String(summary?.active_executions ?? inProgress.length)}
          to="/executions"
          tone="medium"
        />
        <MetricCard
          label={t("dashboard.criticalAssets")}
          value={String(summary?.critical_assets ?? 0)}
          to="/assets"
          tone="high"
        />
        <MetricCard
          label={t("dashboard.pendingHandovers")}
          value={String(pendingHandoverCount)}
          to="/handovers"
          tone="info"
        />
      </section>

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

      <section className="panel" style={{ marginBottom: 20 }}>
        <div className="panel-heading">
          <h2>{t("dashboard.myQueue")}</h2>
          <Link to="/executions" className="secondary-button">
            {t("dashboard.viewAll")}
          </Link>
        </div>
        {inProgress.length === 0 ? (
          <EmptyState title={t("dashboard.myQueueEmpty")} />
        ) : (
          <DataTable<Execution>
            columns={[
              {
                key: "purpose",
                header: t("executions.title"),
                render: (execution) => (
                  <Link to={`/executions/${execution.id}`} className="strong">
                    {execution.purpose}
                  </Link>
                ),
                sortValue: (e) => e.purpose,
              },
              {
                key: "incident",
                header: t("executions.incident"),
                render: (execution) =>
                  execution.incident_id ? (
                    <Link to={`/incidents/${execution.incident_id}`}>
                      {execution.asset_tag ?? execution.incident_id}
                    </Link>
                  ) : (
                    "—"
                  ),
              },
              {
                key: "state",
                header: t("executions.state"),
                render: (execution) => (
                  <StatusBadge
                    tone={executionStateTone(execution.state)}
                    label={t(executionStateLabelKey(execution.state))}
                  />
                ),
                sortValue: (e) => e.state,
              },
            ]}
            rows={inProgress}
            rowKey={(execution) => execution.id}
            emptyTitle={t("dashboard.myQueueEmpty")}
          />
        )}
      </section>

      <section className="panel">
        <div className="panel-heading">
          <h2>{t("dashboard.activeIncidents")}</h2>
        </div>
        {error ? (
          <div className="error-state" role="alert" style={{ margin: 16 }}>
            <strong>Something went wrong</strong>
            <p>{error}</p>
            <div className="error-actions">
              <button className="secondary-button" onClick={onRetry}>
                Retry
              </button>
            </div>
          </div>
        ) : (
          <DataTable<Incident>
            columns={[
              {
                key: "number",
                header: t("dashboard.table.incident"),
                render: (incident) => (
                  <Link to={`/incidents/${incident.id}`} className="strong">
                    {incident.number}
                  </Link>
                ),
                sortValue: (i) => i.number,
              },
              {
                key: "summary",
                header: t("executions.title"),
                render: (incident) => <span className="summary-cell">{incident.summary}</span>,
              },
              {
                key: "asset",
                header: t("dashboard.table.asset"),
                render: (incident) => incident.asset_tag ?? "—",
              },
              {
                key: "severity",
                header: t("dashboard.table.severity"),
                render: (incident) => (
                  <StatusBadge
                    tone={severityTone(incident.severity)}
                    label={t(severityLabelKey(incident.severity))}
                  />
                ),
                sortValue: (i) => i.severity,
              },
              {
                key: "state",
                header: t("dashboard.table.state"),
                render: (incident) => (
                  <StatusBadge
                    tone={incidentStateTone(incident.state)}
                    label={t(incidentStateLabelKey(incident.state))}
                  />
                ),
                sortValue: (i) => i.state,
              },
              {
                key: "age",
                header: t("dashboard.table.age"),
                render: (incident) => <RelativeTime time={incident.detected_at} locale={locale} />,
              },
            ]}
            rows={open}
            rowKey={(incident) => incident.id}
            onRowClick={(incident) => navigate(`/incidents/${incident.id}`)}
            emptyTitle={t("dashboard.myQueueEmpty")}
          />
        )}
      </section>
    </>
  );
}
