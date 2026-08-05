import { Link } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";

/**
 * ReportsPage: maintenance report library with lifecycle state badges.
 */
export function ReportsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const reports = useApi(() => api.reports());

  return (
    <section>
      <Topbar title={t("nav.reports")} principal={principal} />
      {reports.error && (
        <div className="toast-error" role="alert">{reports.error}</div>
      )}
      {reports.loading && !reports.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <div className="panel">
          <div className="panel-heading">
            <h2>{t("report.library")}</h2>
            <span className="count">{reports.data?.items.length ?? 0}</span>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>{t("report.workOrder")}</th>
                  <th>{t("report.summary")}</th>
                  <th>{t("report.state")}</th>
                </tr>
              </thead>
              <tbody>
                {(reports.data?.items ?? []).map((report) => (
                  <tr key={report.id}>
                    <td className="mono strong">
                      <Link to={`/reports/${report.id}`} style={{ color: "inherit" }}>
                        RP-{report.revision}
                      </Link>
                    </td>
                    <td className="summary-cell">{report.structured_content.summary}</td>
                    <td><span className={`state-badge ${report.state.toLowerCase()}`}>{report.state}</span></td>
                  </tr>
                ))}
                {(reports.data?.items ?? []).length === 0 && (
                  <tr>
                    <td colSpan={3} className="empty">{t("report.none")}</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  );
}
