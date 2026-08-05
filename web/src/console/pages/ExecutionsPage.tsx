import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { DataTable } from "../ui/DataTable";
import { StatusBadge, type Tone } from "../ui/StatusBadge";
import type { Execution } from "../../types";

const EXECUTION_TONE: Record<string, Tone> = {
  ASSIGNED: "info",
  IN_PROGRESS: "medium",
  COMPLETED: "success",
};

type StateFilter = "ALL" | "IN_PROGRESS" | "ASSIGNED" | "COMPLETED";
const STATE_FILTERS: StateFilter[] = ["ALL", "ASSIGNED", "IN_PROGRESS", "COMPLETED"];

export function ExecutionsPage() {
  const { t } = useI18n();
  const executions = useQuery(() => api.listExecutions().then((list) => list.items));
  const [filter, setFilter] = useState<StateFilter>("ALL");

  const rows = useMemo(() => {
    const items = executions.data ?? [];
    return filter === "ALL" ? items : items.filter((execution) => execution.state === filter);
  }, [executions.data, filter]);

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("pageTitle.executions")} />
        <p className="section-lead">{t("executions.lead")}</p>
        <div className="incident-tabs" role="tablist" aria-label={t("executions.state")}>
          {STATE_FILTERS.map((state) => (
            <button
              key={state}
              role="tab"
              aria-selected={filter === state}
              className={`tab${filter === state ? " active" : ""}`}
              onClick={() => setFilter(state)}
            >
              {state === "ALL" ? t("incidents.tabs.all") : state.replace("_", " ")}
            </button>
          ))}
        </div>
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
                  <span className="muted">—</span>
                ),
            },
            {
              key: "asset",
              header: t("executions.asset"),
              render: (execution) => execution.asset_tag ?? "—",
            },
            {
              key: "state",
              header: t("executions.state"),
              render: (execution) => (
                <StatusBadge
                  tone={EXECUTION_TONE[execution.state] ?? "info"}
                  label={execution.state.replace("_", " ")}
                />
              ),
              sortValue: (e) => e.state,
            },
            {
              key: "progress",
              header: t("executions.progress"),
              render: (execution) => {
                const total = execution.steps?.length ?? 0;
                const done = execution.steps?.filter((s) => s.state === "COMPLETED").length ?? 0;
                const pct = total ? Math.round((done / total) * 100) : 0;
                return `${pct}%`;
              },
              sortValue: (e) => e.steps?.filter((s) => s.state === "COMPLETED").length ?? 0,
            },
          ]}
          rows={rows}
          rowKey={(execution) => execution.id}
          emptyTitle={t("executions.empty")}
          loading={executions.loading}
          error={executions.error}
          onRetry={() => void executions.refetch()}
        />
      </section>
    </PageTrailProvider>
  );
}
