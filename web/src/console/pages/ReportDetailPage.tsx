import { useState } from "react";
import { useParams } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { Breadcrumbs } from "../layout/Breadcrumbs";
import { EvidenceLinks } from "../components/EvidenceLinks";

/**
 * ReportDetailPage: maintenance report lifecycle (draft -> submit -> approve).
 */
export function ReportDetailPage() {
  const { reportId } = useParams<{ reportId: string }>();
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const report = useApi(() => api.report(reportId!));
  const [busy, setBusy] = useState(false);

  const perms = principal?.permissions ?? [];
  const canWrite = perms.includes("report:write");
  const canApprove = perms.includes("report:approve");

  if (report.error) {
    return <div className="toast-error" role="alert">{report.error}</div>;
  }
  if (report.loading || !report.data) {
    return <div className="skeleton" style={{ height: 300 }} />;
  }
  const value = report.data;

  const mutate = async (action: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await action();
      await report.refetch();
    } finally {
      setBusy(false);
    }
  };

  const submit = () => void mutate(() => api.submitReport(value.id));
  const approve = () => void mutate(() => api.approveReport(value.id));

  const sections: Array<{ key: string; label: string; items: string[] }> = [
    { key: "measurements", label: t("report.measurements"), items: value.structured_content.measurements },
    { key: "observations", label: t("report.observations"), items: value.structured_content.observations },
    { key: "actions", label: t("report.actions"), items: value.structured_content.actions },
    { key: "unknowns", label: t("report.unknowns"), items: value.structured_content.unknowns }
  ];

  return (
    <section>
      <Breadcrumbs
        trail={[
          { label: t("nav.reports"), to: "/reports" },
          { label: `RP-${value.revision}` }
        ]}
      />
      <Topbar title={t("report.title", { revision: value.revision })} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <div>
              <span className="eyebrow">{t("report.workOrder")}</span>
              <h2>{value.structured_content.summary}</h2>
            </div>
            <span className={`state-badge ${value.state.toLowerCase()}`}>{value.state}</span>
          </div>
          <div style={{ padding: 16, display: "flex", gap: 10, flexWrap: "wrap" }}>
            {value.state === "DRAFT" && canWrite && (
              <button className="primary-button" disabled={busy} onClick={submit}>
                {t("report.submit")}
              </button>
            )}
            {value.state === "SUBMITTED" && canApprove && (
              <button className="primary-button" disabled={busy} onClick={approve}>
                {t("report.approve")}
              </button>
            )}
          </div>
        </div>
        <div className="panel">
          <div className="panel-heading"><h2>{t("report.outcome")}</h2></div>
          <div style={{ padding: 16 }}>{value.structured_content.outcome || t("report.noOutcome")}</div>
        </div>
        {sections.map((section) => (
          <div className="panel" key={section.key}>
            <div className="panel-heading"><h2>{section.label}</h2></div>
            <div style={{ padding: 16, display: "grid", gap: 8 }}>
              {section.items.length > 0 ? (
                section.items.map((item, index) => (
                  <div key={index} style={{ fontSize: 13 }}>{item}</div>
                ))
              ) : (
                <div className="empty">{t("report.none")}</div>
              )}
            </div>
          </div>
        ))}
        <div className="panel">
          <div className="panel-heading"><h2>{t("report.evidence")}</h2></div>
          <div style={{ padding: 16 }}>
            <EvidenceLinks evidence={value.evidence} selected={value.structured_content.evidence_ids} />
          </div>
        </div>
      </div>
    </section>
  );
}
