import { Link, useNavigate } from "react-router-dom";
import { api } from "../../api";
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
  const reports = useQuery(() => api.reports().then((list) => list.items));

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.reports")} principal={principal} />
        <div className="panel">
          <div className="panel-heading">
            <h2>{t("report.library")}</h2>
            <span className="count">{reports.data?.length ?? 0}</span>
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
            rows={reports.data ?? []}
            rowKey={(report) => report.id}
            onRowClick={(report) => navigate(`/reports/${report.id}`)}
            emptyTitle={t("report.none")}
            loading={reports.loading}
            error={reports.error}
            onRetry={() => void reports.refetch()}
          />
        </div>
      </section>
    </PageTrailProvider>
  );
}
