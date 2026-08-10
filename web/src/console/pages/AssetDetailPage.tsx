import { Link, useNavigate, useParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { EmptyState } from "../ui/EmptyState";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { GatedButton } from "../ui/GatedButton";
import {
  priorityTone,
  priorityLabelKey,
  incidentStatusTone,
  incidentStatusLabelKey,
  assetStatusTone,
  assetStatusLabelKey,
} from "../labels";
import type { Incident, WorkflowVersion } from "../../types";

/**
 * AssetDetailPage: asset identity, criticality, and real linked history
 * (incidents, executions, applicable workflows). Approve is fire-and-forget
 * no more: busy state, success toast, and refetch.
 */
export function AssetDetailPage() {
  const { assetId } = useParams<{ assetId: string }>();
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const asset = useQuery(() => api.asset(assetId ?? ""));
  const incidents = useQuery(() => api.incidents().then((list) => list.items));
  const executions = useQuery(() => api.listExecutions().then((list) => list.items));
  const workflows = useQuery(
    () => (assetId ? api.applicableWorkflows(assetId).then((list) => list.items) : Promise.resolve([] as WorkflowVersion[])),
    [assetId],
  );
  const canApprove = principal?.permissions.includes("asset:criticality:approve") ?? false;

  const approve = useCommand(
    (id: string) => api.approveAssetCriticality(id),
    {
      successMessage: t("assets.approve.success"),
      onSuccess: () => void asset.refetch(),
    },
  );

  if (asset.error) {
    return (
      <section>
        <PageHeader title={t("nav.assets")} principal={principal} />
        <ErrorState message={asset.error} onRetry={() => void asset.refetch()} onBack={() => navigate("/assets")} />
      </section>
    );
  }
  if (asset.loading || !asset.data) {
    return (
      <section>
        <PageHeader title={t("nav.assets")} principal={principal} />
        <Skeleton height={220} />
      </section>
    );
  }

  const value = asset.data;
  const linkedIncidents =
    incidents.data?.filter((incident) => incident.asset_id === value.id) ?? [];
  const linkedExecutions =
    executions.data?.filter((execution) => execution.asset_id === value.id) ?? [];

  return (
    <PageTrailProvider trail={[{ label: t("nav.assets"), to: "/assets" }, { label: value.tag }]}>
      <section>
        <PageHeader
          title={value.tag}
          principal={principal}
          actions={
            value.criticality ? (
              <GatedButton
                allowed={canApprove}
                reason={t("action.permissionRequired")}
                className="primary-button"
                disabled={approve.pending}
                onClick={() => void approve.run(value.id)}
              >
                {t("asset.approveCriticality")}
              </GatedButton>
            ) : undefined
          }
        />
        <div style={{ display: "grid", gap: 16 }}>
          <div className="panel">
            <div className="panel-heading">
              <div>
                <span className="eyebrow">{t("assets.class")}</span>
                <h2>{value.name}</h2>
              </div>
              <StatusBadge tone={assetStatusTone(value.status)} label={assetStatusLabelKey(value.status) ? t(assetStatusLabelKey(value.status)!) : value.status} />
            </div>
            <div className="incident-facts">
              <div>
                <span className="eyebrow">{t("assets.class")}</span>
                <span className="strong">{value.class}</span>
              </div>
              {value.manufacturer ? (
                <div>
                  <span className="eyebrow">{t("assets.manufacturer")}</span>
                  <span className="strong">{value.manufacturer}</span>
                </div>
              ) : null}
              {value.model ? (
                <div>
                  <span className="eyebrow">{t("assets.model")}</span>
                  <span className="strong">{value.model}</span>
                </div>
              ) : null}
              <div>
                <span className="eyebrow">{t("assets.authority")}</span>
                <span className="strong">
                  {value.source_of_truth === "EXTERNAL_REFERENCE"
                    ? t("assets.externalProjection")
                    : t("assets.skawldNative")}
                </span>
              </div>
            </div>
          </div>
          {value.criticality && (
            <div className="panel">
              <div className="panel-heading"><h2>{t("asset.criticality")}</h2></div>
              <div className="incident-facts">
                <div>
                  <span className="eyebrow">{t("assets.criticality")}</span>
                  <span className={`criticality rating-${value.criticality.rating}`}>{value.criticality.rating}</span>
                </div>
                <div>
                  <span className="eyebrow">{t("asset.safetyImpact")}</span>
                  <span className="mono">{value.criticality.safety_impact}</span>
                </div>
                <div>
                  <span className="eyebrow">{t("asset.productionImpact")}</span>
                  <span className="mono">{value.criticality.production_impact}</span>
                </div>
              </div>
              <div style={{ padding: "0 16px 16px" }}>
                <p style={{ margin: 0, fontSize: 12.5, color: "var(--ink-soft)" }}>{value.criticality.rationale}</p>
              </div>
            </div>
          )}
          <div className="panel">
            <div className="panel-heading">
              <h2>{t("assets.linkedHistory")}</h2>
            </div>
            <div style={{ padding: 16, display: "grid", gap: 20 }}>
              <div>
                <span className="eyebrow">{t("assets.incidents")}</span>
                {linkedIncidents.length > 0 ? (
                  <DataTable<Incident>
                    columns={[
                      {
                        key: "number",
                        header: t("dashboard.table.incident"),
                        render: (incident) => (
                          <Link to={`/incidents/${incident.id}`} className="strong" onClick={(event) => event.stopPropagation()}>
                            {incident.number}
                          </Link>
                        ),
                      },
                      {
                        key: "summary",
                        header: t("form.summary"),
                        render: (incident) => incident.summary,
                      },
                      {
                        key: "severity",
                        header: t("dashboard.table.severity"),
                        render: (incident) => <StatusBadge tone={priorityTone(incident.priority)} label={t(priorityLabelKey(incident.priority))} />,
                      },
                      {
                        key: "state",
                        header: t("dashboard.table.state"),
                        render: (incident) => <StatusBadge tone={incidentStatusTone(incident.status)} label={t(incidentStatusLabelKey(incident.status))} />,
                      },
                    ]}
                    rows={linkedIncidents}
                    rowKey={(incident) => incident.id}
                    emptyTitle={t("assets.noIncidents")}
                    onRowClick={(incident) => navigate(`/incidents/${incident.id}`)}
                  />
                ) : (
                  <EmptyState title={t("assets.noIncidents")} />
                )}
              </div>
              <div>
                <span className="eyebrow">{t("nav.executions")}</span>
                {linkedExecutions.length > 0 ? (
                  <DataTable<typeof linkedExecutions[number]>
                    columns={[
                      {
                        key: "purpose",
                        header: t("executions.title"),
                        render: (execution) => (
                          <Link to={`/executions/${execution.id}`} className="strong" onClick={(event) => event.stopPropagation()}>
                            {execution.purpose}
                          </Link>
                        ),
                      },
                      {
                        key: "state",
                        header: t("executions.state"),
                        render: (execution) => (
                          <StatusBadge
                            tone={execution.state === "COMPLETED" ? "success" : execution.state === "IN_PROGRESS" ? "medium" : "info"}
                            label={execution.state.replace("_", " ")}
                          />
                        ),
                      },
                    ]}
                    rows={linkedExecutions}
                    rowKey={(execution) => execution.id}
                    emptyTitle={t("assets.noWorkflows")}
                  />
                ) : (
                  <EmptyState title={t("assets.noWorkflows")} />
                )}
              </div>
              <div>
                <span className="eyebrow">{t("assets.applicableWorkflows")}</span>
                {workflows.data && workflows.data.length > 0 ? (
                  <DataTable<WorkflowVersion>
                    columns={[
                      { key: "name", header: t("nav.workflows"), render: (workflow) => workflow.name ?? workflow.workflow_key },
                      {
                        key: "status",
                        header: t("executions.state"),
                        render: (workflow) => <StatusBadge tone={workflow.status === "PUBLISHED" ? "success" : "info"} label={workflow.status.replace("_", " ")} />,
                      },
                    ]}
                    rows={workflows.data}
                    rowKey={(workflow) => `${workflow.workflow_id}-${workflow.version}`}
                    emptyTitle={t("assets.noWorkflows")}
                  />
                ) : (
                  <EmptyState title={t("assets.noWorkflows")} />
                )}
              </div>
            </div>
          </div>
        </div>
      </section>
    </PageTrailProvider>
  );
}
