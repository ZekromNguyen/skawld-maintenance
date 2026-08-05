import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { relativeTime } from "../../presentation";
import { Topbar } from "../layout/Topbar";
import { Breadcrumbs } from "../layout/Breadcrumbs";
import type { Execution } from "../../types";

/**
 * IncidentDetailPage: incident context, linked asset, executions, actions.
 */
export function IncidentDetailPage() {
  const { incidentId } = useParams<{ incidentId: string }>();
  const { t, locale } = useI18n();
  const { data: principal } = usePrincipal();
  const incident = useApi(() => api.incident(incidentId!));
  const executions = useApi(() => api.listExecutions());
  const [busy, setBusy] = useState(false);

  const perms = principal?.permissions ?? [];
  const canCreateExecution = perms.includes("execution:write");
  const canResolve = perms.includes("incident:resolve");

  if (incident.error) {
    return <div className="toast-error" role="alert">{incident.error}</div>;
  }
  if (incident.loading || !incident.data) {
    return <div className="skeleton" style={{ height: 220 }} />;
  }
  const value = incident.data;
  const related: Execution[] = (executions.data?.items ?? []).filter(
    (execution) => execution.incident_id === value.id,
  );

  const createExecution = async () => {
    setBusy(true);
    try {
      await api.createExecution(value.id);
      await executions.refetch();
    } finally {
      setBusy(false);
    }
  };

  const resolve = async () => {
    setBusy(true);
    try {
      await api.resolveIncident(value.id);
      await incident.refetch();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section>
      <Breadcrumbs
        trail={[
          { label: t("nav.incidents"), to: "/incidents" },
          { label: value.number }
        ]}
      />
      <Topbar title={value.number} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <h2>{value.summary}</h2>
            <span className={`severity ${value.severity.toLowerCase()}`}>{value.severity}</span>
          </div>
          <div style={{ padding: 16, display: "grid", gap: 10 }}>
            <div>{t("incident.state")} <span className="state-badge">{value.state}</span></div>
            <div>
              {t("incident.asset")}{" "}
              <Link to={`/assets/${value.asset_id}`} className="strong">
                {value.asset_tag ?? value.asset_id}
              </Link>
            </div>
            <div>{t("incident.detected")} <span className="mono">{relativeTime(value.detected_at, locale)}</span></div>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              {canCreateExecution && (
                <button className="primary-button" disabled={busy} onClick={() => void createExecution()}>
                  {t("incident.createExecution")}
                </button>
              )}
              {canResolve && value.state !== "RESOLVED" && (
                <button className="secondary-button" disabled={busy} onClick={() => void resolve()}>
                  {t("incident.resolve")}
                </button>
              )}
            </div>
          </div>
        </div>
        <div className="panel">
          <div className="panel-heading">
            <h2>{t("incident.executions")}</h2>
            <span className="count">{related.length}</span>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>{t("incident.purpose")}</th>
                  <th>{t("incident.state")}</th>
                  <th>{t("incident.openInWorkbench")}</th>
                </tr>
              </thead>
              <tbody>
                {related.map((execution) => (
                  <tr key={execution.id}>
                    <td>{execution.purpose}</td>
                    <td><span className="state-badge">{execution.state}</span></td>
                    <td>
                      <Link to={`/executions/${execution.id}`} className="strong">
                        {t("incident.openInWorkbench")}
                      </Link>
                    </td>
                  </tr>
                ))}
                {related.length === 0 && (
                  <tr>
                    <td colSpan={3} className="empty">
                      {t("incident.noExecutions")}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  );
}
