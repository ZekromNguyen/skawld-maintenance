import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { formatCustomValue } from "../components/customFieldFormat";
import type { CustomFieldDefinition } from "../../types";
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
  priorityTone,
  priorityLabelKey,
  incidentStatusTone,
  incidentStatusLabelKey,
} from "../labels";
import type { Attachment, Execution } from "../../types";

function formatDuration(totalSeconds: number): string {
  const days = Math.floor(totalSeconds / 86_400);
  const hours = Math.floor((totalSeconds % 86_400) / 3_600);
  const minutes = Math.floor((totalSeconds % 3_600) / 60);
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes}m`);
  return parts.join(" ");
}

function isVideo(mime: string): boolean {
  return mime.startsWith("video/");
}

function MediaGallery(props: { attachments: Attachment[]; empty: string }) {
  const available = props.attachments.filter((item) => item.download_url);
  if (available.length === 0) {
    return <p className="incident-details-body">{props.empty}</p>;
  }
  return (
    <div className="incident-media-gallery">
      {available.map((item) =>
        isVideo(item.verified_mime || item.declared_mime) ? (
          <video key={item.id} controls src={item.download_url} className="incident-media" />
        ) : (
          <img key={item.id} src={item.download_url} alt={item.original_filename} className="incident-media" />
        ),
      )}
    </div>
  );
}

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

  function customFieldEntries(value: { custom_values?: Record<string, unknown>; custom_fields?: CustomFieldDefinition[] }) {
    return (value.custom_fields ?? [])
      .filter((field) => field.status === "ACTIVE")
      .map((field) => ({ field, value: value.custom_values?.[field.id] }))
      .filter((entry) => entry.value !== undefined && entry.value !== null && entry.value !== "");
  }
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

  const closeIncidentCommand = useCommand(
    (id: string) => api.closeIncident(id, incident.data?.version ?? 0),
    {
      successMessage: t("incident.close.success"),
      onSuccess: () => void incident.refetch(),
    },
  );

  const reopenCommand = useCommand(
    (id: string) => api.reopenIncident(id, incident.data?.version ?? 0),
    {
      successMessage: t("incident.reopen.success"),
      onSuccess: () => void incident.refetch(),
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
              {value.status !== "RESOLVED" && (
                <GatedButton
                  allowed={canResolve}
                  reason={permissionReason}
                  className="secondary-button"
                  onClick={() => setConfirmResolve(true)}
                >
                  {t("incident.resolve")}
                </GatedButton>
              )}
              {value.status === "RESOLVED" && (
                <GatedButton
                  allowed={canResolve}
                  reason={permissionReason}
                  className="secondary-button"
                  disabled={closeIncidentCommand.pending}
                  onClick={() => void closeIncidentCommand.run(value.id)}
                >
                  {t("incident.close")}
                </GatedButton>
              )}
              {value.status === "RESOLVED" && (
                <GatedButton
                  allowed={canResolve}
                  reason={permissionReason}
                  className="secondary-button"
                  disabled={reopenCommand.pending}
                  onClick={() => void reopenCommand.run(value.id)}
                >
                  {t("incident.reopen")}
                </GatedButton>
              )}
              {value.status !== "RESOLVED" && (
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
        <div className="incident-subtitle">
          <span className="mono">{value.number}</span>
          <span className="incident-pills" aria-label={t("incident.facts")}>
            <StatusBadge tone={priorityTone(value.priority)} label={t(priorityLabelKey(value.priority))} />
            <StatusBadge tone={incidentStatusTone(value.status)} label={t(incidentStatusLabelKey(value.status))} />
          </span>
        </div>
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
                  header: t("incident.status"),
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
          {value.details ? (
            <div className="panel">
              <div className="panel-heading">
                <h2>{t("incident.details")}</h2>
              </div>
              <p className="incident-details-body">{value.details}</p>
            </div>
          ) : null}
          <div className="panel">
            <div className="panel-heading">
              <h2>{t("incident.media")}</h2>
              <span className="count">{(value.attachments ?? []).length}</span>
            </div>
            <MediaGallery attachments={value.attachments ?? []} empty={t("incident.media.empty")} />
          </div>
          </div>
          <aside className="metadata-rail" aria-label={t("incident.facts")}>
            {customFieldEntries(value).length > 0 ? (
              <div className="panel">
                <div className="panel-heading">
                  <h2>{t("incident.customFields")}</h2>
                </div>
                <div className="incident-facts">
                  {customFieldEntries(value).map(({ field, value: fieldValue }) => (
                    <div key={field.id}>
                      <span className="eyebrow">{field.label}</span>
                      <span>{formatCustomValue(field, fieldValue)}</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
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
                  <span className="eyebrow">{t("incident.priority")}</span>
                  <StatusBadge tone={priorityTone(value.priority)} label={t(priorityLabelKey(value.priority))} />
                </div>
                <div>
                  <span className="eyebrow">{t("incident.status")}</span>
                  <StatusBadge tone={incidentStatusTone(value.status)} label={t(incidentStatusLabelKey(value.status))} />
                </div>
                {value.assignee_name ? (
                  <div>
                    <span className="eyebrow">{t("incident.assignee")}</span>
                    <span>{value.assignee_name}</span>
                  </div>
                ) : null}
                {value.reporter_name ? (
                  <div>
                    <span className="eyebrow">{t("incident.reporter")}</span>
                    <span>{value.reporter_name}</span>
                  </div>
                ) : null}
                {value.team_name ? (
                  <div>
                    <span className="eyebrow">{t("incident.team")}</span>
                    <span>{value.team_name}</span>
                  </div>
                ) : null}
                <div>
                  <span className="eyebrow">{t("incident.date")}</span>
                  <RelativeTime time={value.occurred_at ?? value.detected_at} locale={locale} />
                </div>
                <div>
                  <span className="eyebrow">{t("incident.timeToCreate")}</span>
                  <RelativeTime time={value.detected_at} locale={locale} />
                </div>
                <div>
                  <span className="eyebrow">{t("incident.timeToComplete")}</span>
                  {value.time_to_complete_seconds != null
                    ? formatDuration(value.time_to_complete_seconds)
                    : "—"}
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
