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
import { StatusBadge } from "../ui/StatusBadge";
import { EmptyState } from "../ui/EmptyState";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { GatedButton } from "../ui/GatedButton";
import { EvidenceLinks } from "../components/EvidenceLinks";
import { reportStateTone, reportStateLabelKey } from "../labels";

/**
 * ReportDetailPage: maintenance report lifecycle (draft -> submit -> approve)
 * with the human-review flag surfaced, execution link, and metadata.
 */
export function ReportDetailPage() {
  const { reportId } = useParams<{ reportId: string }>();
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const report = useQuery(() => api.report(reportId ?? ""));
  const [confirmApprove, setConfirmApprove] = useState(false);

  const perms = principal?.permissions ?? [];
  const canWrite = perms.includes("report:write");
  const canApprove = perms.includes("report:approve");
  const permissionReason = t("action.permissionRequired");

  const submit = useCommand(
    (id: string) => api.submitReport(id),
    { successMessage: t("report.submitSuccess"), onSuccess: () => void report.refetch() },
  );
  const approve = useCommand(
    (id: string) => api.approveReport(id),
    {
      successMessage: t("report.approveSuccess"),
      onSuccess: () => {
        setConfirmApprove(false);
        void report.refetch();
      },
    },
  );

  if (report.error) {
    return (
      <section>
        <PageHeader title={t("nav.reports")} principal={principal} />
        <ErrorState message={report.error} onRetry={() => void report.refetch()} onBack={() => navigate("/reports")} />
      </section>
    );
  }
  if (report.loading || !report.data) {
    return (
      <section>
        <PageHeader title={t("nav.reports")} principal={principal} />
        <Skeleton height={300} />
      </section>
    );
  }

  const value = report.data;
  const requiresReview = value.structured_content.requires_human_review === true;
  const sections: Array<{ key: string; label: string; items: string[] }> = [
    { key: "measurements", label: t("report.measurements"), items: value.structured_content.measurements },
    { key: "observations", label: t("report.observations"), items: value.structured_content.observations },
    { key: "actions", label: t("report.actions"), items: value.structured_content.actions },
    { key: "unknowns", label: t("report.unknowns"), items: value.structured_content.unknowns }
  ];

  return (
    <PageTrailProvider
      trail={[{ label: t("nav.reports"), to: "/reports" }, { label: `RP-${value.revision}` }]}
    >
      <section>
        <PageHeader
          title={t("report.title", { revision: value.revision })}
          principal={principal}
          actions={
            <>
              <button className="secondary-button" onClick={() => window.print()}>
                {t("report.print")}
              </button>
              {value.state === "DRAFT" && (
                <GatedButton
                  allowed={canWrite}
                  reason={permissionReason}
                  className="primary-button"
                  disabled={submit.pending}
                  onClick={() => void submit.run(value.id)}
                >
                  {t("report.submit")}
                </GatedButton>
              )}
              {value.state === "SUBMITTED" && (
                <GatedButton
                  allowed={canApprove}
                  reason={permissionReason}
                  className="primary-button"
                  onClick={() => setConfirmApprove(true)}
                >
                  {t("report.approve")}
                </GatedButton>
              )}
            </>
          }
        />
        {requiresReview ? (
          <div className="notice" role="alert">
            {t("report.requiresHumanReview")}
          </div>
        ) : null}
        <div style={{ display: "grid", gap: 16 }}>
          <div className="panel">
            <div className="panel-heading">
              <div>
                <span className="eyebrow">{t("report.workOrder")}</span>
                <h2>{value.structured_content.summary}</h2>
              </div>
              <StatusBadge tone={reportStateTone(value.state)} label={reportStateLabelKey(value.state) ? t(reportStateLabelKey(value.state)!) : value.state} />
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
                  <EmptyState title={t("report.none")} />
                )}
              </div>
            </div>
          ))}
          <div className="panel">
            <div className="panel-heading"><h2>{t("report.evidence")}</h2></div>
            <div style={{ padding: 16 }}>
              <EvidenceLinks evidence={value.evidence} selected={value.structured_content.evidence_ids} emptyTitle={t("report.noEvidence")} />
            </div>
          </div>
          <div className="panel">
            <div className="panel-heading"><h2>{t("report.metadata")}</h2></div>
            <div className="incident-facts">
              {value.execution_id ? (
                <div>
                  <span className="eyebrow">{t("report.execution")}</span>
                  <Link to={`/executions/${value.execution_id}`} className="strong">
                    {value.execution_id.slice(0, 12)}
                  </Link>
                </div>
              ) : null}
              {value.provider ? (
                <div>
                  <span className="eyebrow">{t("report.provider")}</span>
                  <span className="strong">{value.provider}</span>
                </div>
              ) : null}
              {value.model ? (
                <div>
                  <span className="eyebrow">{t("report.model")}</span>
                  <span className="mono">{value.model}</span>
                </div>
              ) : null}
              {value.prompt_version ? (
                <div>
                  <span className="eyebrow">{t("report.promptVersion")}</span>
                  <span className="mono">{value.prompt_version}</span>
                </div>
              ) : null}
              <div>
                <span className="eyebrow">{t("report.state")}</span>
                <StatusBadge tone={reportStateTone(value.state)} label={reportStateLabelKey(value.state) ? t(reportStateLabelKey(value.state)!) : value.state} />
              </div>
            </div>
          </div>
        </div>
        <ConfirmDialog
          open={confirmApprove}
          onOpenChange={setConfirmApprove}
          title={t("report.approveConfirm")}
          message={t("report.approveConfirmBody")}
          confirmLabel={t("report.approve")}
          pending={approve.pending}
          onConfirm={() => void approve.run(value.id)}
        />
      </section>
    </PageTrailProvider>
  );
}
