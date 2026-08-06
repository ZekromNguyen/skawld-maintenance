import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { ConfirmDialog } from "../feedback/ConfirmDialog";
import { StatusBadge } from "../ui/StatusBadge";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { approvalTone, approvalLabelKey } from "../labels";
import type { DocumentRevision } from "../../types";

/**
 * DocumentDetailPage: knowledge document revisions with lifecycle actions.
 * Retire only on APPROVED/SUPERSEDED revisions, both destructive actions
 * confirmed, per-revision busy state, ingest errors inline.
 */
export function DocumentDetailPage() {
  const { documentId } = useParams<{ documentId: string }>();
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const document = useQuery(() => api.document(documentId ?? ""));
  const [pending, setPending] = useState<{ kind: "approve" | "retire"; revision: DocumentRevision } | null>(null);

  const perms = principal?.permissions ?? [];
  const canApprove = perms.includes("knowledge:approve");
  const canWrite = perms.includes("knowledge:write");

  const approve = useCommand(
    (id: string) => api.approveDocumentRevision(id),
    {
      successMessage: t("document.approveSuccess"),
      onSuccess: () => {
        setPending(null);
        void document.refetch();
      },
    },
  );
  const retire = useCommand(
    (id: string) => api.retireDocumentRevision(id, "superseded"),
    {
      successMessage: t("document.retireSuccess"),
      onSuccess: () => {
        setPending(null);
        void document.refetch();
      },
    },
  );
  const ingest = useCommand(
    (id: string) => api.requestDocumentIngestion(id),
    {
      successMessage: t("document.ingestSuccess"),
      onSuccess: () => void document.refetch(),
    },
  );

  if (document.error) {
    return (
      <section>
        <PageHeader title={t("nav.knowledge")} principal={principal} />
        <ErrorState message={document.error} onRetry={() => void document.refetch()} onBack={() => navigate("/knowledge")} />
      </section>
    );
  }
  if (document.loading || !document.data) {
    return (
      <section>
        <PageHeader title={t("nav.knowledge")} principal={principal} />
        <Skeleton height={300} />
      </section>
    );
  }

  const value = document.data;
  const canRetire = (revision: DocumentRevision) =>
    canWrite &&
    revision.approval_status !== "RETIRED" &&
    (revision.approval_status === "APPROVED" || revision.approval_status === "SUPERSEDED");

  return (
    <PageTrailProvider trail={[{ label: t("nav.knowledge"), to: "/knowledge" }, { label: value.title }]}>
      <section>
        <PageHeader title={value.title} principal={principal} />
        <div style={{ display: "grid", gap: 16 }}>
          <div className="panel">
            <div className="panel-heading">
              <div>
                <span className="eyebrow">{value.document_type}</span>
                <h2>{value.title}</h2>
              </div>
              <span className="source-badge">{value.authority}</span>
            </div>
            <div className="incident-facts">
              <div>
                <span className="eyebrow">{t("document.revisions")}</span>
                <span className="count">{value.revisions.length}</span>
              </div>
              {value.source_reference ? (
                <div>
                  <span className="eyebrow">{t("document.sourceReference")}</span>
                  <span className="mono">{value.source_reference}</span>
                </div>
              ) : null}
            </div>
          </div>
          {value.revisions.map((revision) => (
            <div className="panel" key={revision.id}>
              <div className="panel-heading">
                <h2 className="mono">{revision.revision}</h2>
                <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                  <StatusBadge tone={approvalTone(revision.approval_status)} label={approvalLabelKey(revision.approval_status) ? t(approvalLabelKey(revision.approval_status)!) : revision.approval_status} />
                  <StatusBadge tone="info" label={revision.ingestion_state.replace("_", " ")} />
                </div>
              </div>
              <div style={{ padding: 16, display: "grid", gap: 10 }}>
                <div>{t("document.language")} <span className="mono">{revision.language}</span></div>
                {revision.version ? (
                  <div>{t("document.version")} <span className="mono">{revision.version}</span></div>
                ) : null}
                {revision.ingestion_error && (
                  <div className="form-error" role="alert">{revision.ingestion_error}</div>
                )}
                {revision.applicability.length > 0 && (
                  <div>
                    <span className="eyebrow">{t("document.applicability")}</span>
                    <ul style={{ margin: "8px 0 0", paddingLeft: 18 }}>
                      {revision.applicability.map((item, index) => (
                        <li key={index} style={{ fontSize: 12 }}>
                          {[item.site_id, item.asset_class, item.manufacturer, item.model]
                            .filter(Boolean)
                            .join(" · ") || t("document.none")}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                  {(revision.approval_status === "DRAFT" || revision.approval_status === "REVIEW_REQUIRED") &&
                    canApprove && (
                      <button
                        className="primary-button"
                        disabled={approve.pending || retire.pending}
                        onClick={() => setPending({ kind: "approve", revision })}
                      >
                        {t("document.approve")}
                      </button>
                    )}
                  {revision.ingestion_state === "AWAITING_UPLOAD" && canWrite && (
                    <button
                      className="secondary-button"
                      disabled={ingest.pending}
                      onClick={() => void ingest.run(revision.id)}
                    >
                      {t("document.requestIngestion")}
                    </button>
                  )}
                  {canRetire(revision) && (
                    <button
                      className="secondary-button"
                      disabled={approve.pending || retire.pending}
                      onClick={() => setPending({ kind: "retire", revision })}
                    >
                      {t("document.retire")}
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
        <ConfirmDialog
          open={pending !== null}
          onOpenChange={(open) => {
            if (!open) setPending(null);
          }}
          title={pending?.kind === "approve" ? t("document.approveConfirm") : t("document.retireConfirm")}
          message={pending?.kind === "approve" ? t("document.approveConfirmBody") : t("document.retireConfirmBody")}
          confirmLabel={pending?.kind === "approve" ? t("document.approve") : t("document.retire")}
          pending={approve.pending || retire.pending}
          onConfirm={() => {
            if (!pending) return;
            if (pending.kind === "approve") void approve.run(pending.revision.id);
            else void retire.run(pending.revision.id);
          }}
        />
      </section>
    </PageTrailProvider>
  );
}
