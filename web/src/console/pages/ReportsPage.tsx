import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { api, type ListOptions } from "../../api";
import { useQuery } from "../useQuery";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { reportStateTone, reportStateLabelKey } from "../labels";
import type { MaintenanceReport } from "../../types";

/**
 * ReportsPage: maintenance report library with lifecycle state badges,
 * execution links, and a state filter.
 */
export function ReportsPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const [stateFilter, setStateFilter] = useState<string>("ALL");
  const [history, setHistory] = useState<MaintenanceReport[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);

  const options = useMemo<ListOptions>(
    () => ({
      ...(stateFilter !== "ALL" ? { state: [stateFilter] } : {}),
      page_size: 25
    }),
    [stateFilter],
  );
  const reports = useQuery(() => api.reports(options), [options]);
  const rows = useMemo(
    () => [...(reports.data?.items ?? []), ...history],
    [reports.data, history],
  );

  useEffect(() => {
    setHistory([]);
    setNextCursor(reports.data?.next_cursor ?? null);
  }, [reports.data, stateFilter]);

  const loadMore = async () => {
    if (!nextCursor) return;
    const page = await api.reports({ ...options, cursor: nextCursor });
    setHistory((prev) => [...prev, ...page.items]);
    setNextCursor(page.next_cursor);
  };

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.reports")} principal={principal} />
        <div className="panel">
          <div className="panel-heading">
            <h2>{t("report.library")}</h2>
            <select
              aria-label="State"
              className="filter-select"
              value={stateFilter}
              onChange={(event) => setStateFilter(event.target.value)}
            >
              <option value="ALL">{t("report.state.all")}</option>
              <option value="DRAFT">{t("report.state.draft")}</option>
              <option value="SUBMITTED">{t("report.state.submitted")}</option>
              <option value="APPROVED">{t("report.state.approved")}</option>
            </select>
            <span className="count">{rows.length}</span>
          </div>
          <DataTable<MaintenanceReport>
            columns={[
              {
                key: "workOrder",
                header: t("report.workOrder"),
                render: (report) => (
                  <Link to={`/reports/${report.id}`} className="mono strong" onClick={(event) => event.stopPropagation()}>
                    RP-{report.revision}
                  </Link>
                ),
                sortValue: (r) => r.revision
              },
              {
                key: "summary",
                header: t("report.summary"),
                render: (report) => <span className="summary-cell">{report.structured_content.summary}</span>
              },
              {
                key: "execution",
                header: t("report.execution"),
                render: (report) =>
                  report.execution_id ? (
                    <Link to={`/executions/${report.execution_id}`} onClick={(event) => event.stopPropagation()}>
                      {report.execution_id.slice(0, 8)}
                    </Link>
                  ) : (
                    "—"
                  )
              },
              {
                key: "state",
                header: t("report.state"),
                render: (report) => (
                  <StatusBadge tone={reportStateTone(report.state)} label={reportStateLabelKey(report.state) ? t(reportStateLabelKey(report.state)!) : report.state} />
                ),
                sortValue: (r) => r.state
              }
            ]}
            rows={rows}
            rowKey={(report) => report.id}
            onRowClick={(report) => navigate(`/reports/${report.id}`)}
            emptyTitle={t("report.none")}
            loading={reports.loading}
            error={reports.error}
            onRetry={() => void reports.refetch()}
          />
          {nextCursor ? (
            <button
              type="button"
              className="secondary-button"
              onClick={() => void loadMore()}
              style={{ marginTop: 12 }}
            >
              {t("common.loadMore")}
            </button>
          ) : null}
        </div>
      </section>
    </PageTrailProvider>
  );
}
