import { useState } from "react";
import { Link } from "react-router";
import { api } from "../../api";
import { usePaginatedList } from "../usePaginatedList";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { Tabs, tabPanelId } from "../ui/Tabs";
import { executionStateLabelKey, executionStateTone } from "../labels";
import type { Execution } from "../../types";

type StateFilter = "ALL" | "IN_PROGRESS" | "ASSIGNED" | "COMPLETED";
const STATE_FILTERS: StateFilter[] = ["ALL", "ASSIGNED", "IN_PROGRESS", "COMPLETED"];

export function ExecutionsPage() {
  const { t } = useI18n();
  const [filter, setFilter] = useState<StateFilter>("ALL");
  const executions = usePaginatedList(
    (params) =>
      api.listExecutions({
        ...params,
        state: filter === "ALL" ? undefined : [filter],
      }),
    [filter],
  );

  const rows = executions.items;

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("pageTitle.executions")} />
        <p className="section-lead">{t("executions.lead")}</p>
        <Tabs
          label={t("executions.state")}
          tabs={STATE_FILTERS.map((state) => ({
            id: state,
            label: state === "ALL" ? t("incidents.tabs.all") : t(executionStateLabelKey(state)),
          }))}
          active={filter}
          onChange={(id) => setFilter(id as StateFilter)}
        />
        <div
          id={tabPanelId(t("executions.state"))}
          role="tabpanel"
          aria-labelledby={`${tabPanelId(t("executions.state"))}-${filter}`}
        >
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
                    {execution.incident_number ?? execution.asset_tag ?? execution.incident_id}
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
                  tone={executionStateTone(execution.state)}
                  label={t(executionStateLabelKey(execution.state))}
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
        </div>
        {executions.hasMore ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void executions.loadMore()}
            disabled={executions.loading}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}
          </button>
        ) : null}
      </section>
    </PageTrailProvider>
  );
}
