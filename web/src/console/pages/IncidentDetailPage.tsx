import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { ConfirmDialog } from "../feedback/ConfirmDialog";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { GatedButton } from "../ui/GatedButton";
import { ActivityFeed, type ActivityEntry } from "../components/ActivityFeed";
import {
  severityTone,
  severityLabelKey,
  incidentStateTone,
  incidentStateLabelKey,
} from "../labels";
import type { Execution } from "../../types";

/**
 * IncidentDetailPage: incident context, linked asset, executions, actions.
 * Resolve requires confirmation; creation navigates straight to the workbench.
 */
export function IncidentDetailPage() {
  const { incidentId } = useParams<{ incidentId: string }>();
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const incident = useQuery(() => api.incident(incidentId ?? ""));
  const executions = useQuery(() => api.listExecutions().then((list) => list.items));
  const [confirmResolve, setConfirmResolve] = useState(false);

  const perms = principal?.permissions ?? [];
  const canCreateExecution = perms.includes("execution:write");
  const canResolve = perms.includes("incident:resolve");
  const canRecommend = perms.includes("recommendation:run");
  const permissionReason = t("action.permissionRequired");

  const resolve = useCommand(
    (id: string) => api.resolveIncident(id),
    {
      successMessage: t("incident.resolve.success"),
      onSuccess: () => {
        setConfirmResolve(false);
        void incident.refetch();
      },
    },
  );

  const createExecution = useCommand(
    (id: string) => api.createExecution(id),
    {
      successMessage: t("incident.execution.created"),
      onSuccess: (execution) => navigate(`/executions/${execution.id}`),
    },
  );

  const generateRecommendation = useCommand(
    (id: string) => api.generateRecommendation(id),
    {
      successMessage: t("execution.recommendationGenerated"),
      onSuccess: () => void incident.refetch(),
    },
  );

  if (incident.error) {
    return (
      <section>
        <PageHeader title={t("nav.incidents")} principal={principal} />
        <ErrorState
          message={incident.error}
          onRetry={() => void incident.refetch()}
          onBack={() => navigate("/incidents")}
        />
      </section>
    );
  }
  if (incident.loading || !incident.data) {
    return (
      <section>
        <PageHeader title={t("nav.incidents")} principal={principal} />
        <Skeleton height={220} />
      </section>
    );
  }

  const value = incident.data;
  const related =
    executions.data?.filter((execution) => execution.incident_id === value.id) ?? [];

  const activity: ActivityEntry[] = [
    {
      id: `detected-${value.id}`,
      kind: "detected",
      title: t("incident.activity.detected"),
      time: value.detected_at,
    },
  ];
  for (const execution of related) {
    activity.push({
      id: `execution-${execution.id}`,
      kind: "execution",
      title: execution.purpose,
      to: `/executions/${execution.id}`,
      detail: execution.state.replace("_", " "),
    });
    for (const measurement of execution.measurements ?? []) {
      activity.push({
        id: `measurement-${measurement.id}`,
        kind: "measurement",
        title: t("incident.activity.measurement"),
        detail: `${measurement.value} ${measurement.unit}`,
        time: measurement.observed_at,
      });
    }
  }

  return (
    <PageTrailProvider
      trail={[
        { label: t("nav.incidents"), to: "/incidents" },
        { label: value.number },
      ]}
    >
      <section>
        <PageHeader
          title={value.summary}
          principal={principal}
          actions={
            <>
              {canCreateExecution && (
                <button
                  className="primary-button"
                  disabled={createExecution.pending}
                  onClick={() => void createExecution.run(value.id)}
                >
                  {t("incident.createExecution")}
                </button>
              )}
              {value.state !== "RESOLVED" && (
                <GatedButton
                  allowed={canResolve}
                  reason={permissionReason}
                  className="secondary-button"
                  onClick={() => setConfirmResolve(true)}
                >
                  {t("incident.resolve")}
                </GatedButton>
              )}
              {value.state !== "RESOLVED" && (
                <GatedButton
                  allowed={canRecommend}
                  reason={permissionReason}
                  className="secondary-button"
                  disabled={generateRecommendation.pending}
                  onClick={() => void generateRecommendation.run(value.id)}
                >
                  {t("execution.generateRecommendation")}
                </GatedButton>
              )}
            </>
          }
        />
        <div className="incident-subtitle mono">{value.number}</div>
        <div className="detail-layout">
          <div className="detail-main" style={{ display: "grid", gap: 16 }}>
            <div className="panel">
              <div className="panel-heading">
                <h2>{t("incident.executions")}</h2>
                <span className="count">{related.length}</span>
            </div>
            <DataTable<Execution>
              columns={[
                {
                  key: "purpose",
                  header: t("incident.purpose"),
                  render: (execution) => (
                    <Link to={`/executions/${execution.id}`} className="strong">
                      {execution.purpose}
                    </Link>
                  ),
                },
                {
                  key: "state",
                  header: t("incident.state"),
                  render: (execution) => (
                    <StatusBadge
                      tone={
                        execution.state === "COMPLETED"
                          ? "success"
                          : execution.state === "IN_PROGRESS"
                            ? "medium"
                            : "info"
                      }
                      label={execution.state.replace("_", " ")}
                    />
                  ),
                },
                {
                  key: "open",
                  header: t("incident.openInWorkbench"),
                  render: (execution) => (
                    <Link to={`/executions/${execution.id}`}>{t("incident.openInWorkbench")}</Link>
                  ),
                },
              ]}
              rows={related}
              rowKey={(execution) => execution.id}
              emptyTitle={t("incident.noExecutions")}
              loading={executions.loading}
              error={executions.error}
              onRetry={() => void executions.refetch()}
            />
          </div>
          <div className="panel">
            <div className="panel-heading">
              <h2>{t("incident.activity")}</h2>
            </div>
            <ActivityFeed entries={activity} />
          </div>
          </div>
          <aside className="metadata-rail" aria-label={t("incident.facts")}>
            <div className="panel">
              <div className="panel-heading">
                <h2>{t("incident.facts")}</h2>
              </div>
              <div className="incident-facts">
                <div>
                  <span className="eyebrow">{t("incident.asset")}</span>
                  <Link to={`/assets/${value.asset_id}`} className="strong">
                    {value.asset_tag ?? value.asset_id}
                  </Link>
                </div>
                <div>
                  <span className="eyebrow">{t("dashboard.table.severity")}</span>
                  <StatusBadge tone={severityTone(value.severity)} label={t(severityLabelKey(value.severity))} />
                </div>
                <div>
                  <span className="eyebrow">{t("incident.state")}</span>
                  <StatusBadge tone={incidentStateTone(value.state)} label={t(incidentStateLabelKey(value.state))} />
                </div>
                <div>
                  <span className="eyebrow">{t("incident.detected")}</span>
                  <RelativeTime time={value.detected_at} locale={locale} />
                </div>
              </div>
            </div>
          </aside>
        </div>
        <ConfirmDialog
          open={confirmResolve}
          onOpenChange={setConfirmResolve}
          title={t("incident.resolve")}
          message={t("incident.resolve.message")}
          confirmLabel={t("incident.resolve")}
          pending={resolve.pending}
          onConfirm={() => void resolve.run(value.id)}
        />
      </section>
    </PageTrailProvider>
  );
}
