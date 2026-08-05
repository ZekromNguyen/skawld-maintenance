import { useState } from "react";
import { useParams } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { Breadcrumbs } from "../layout/Breadcrumbs";
import type { DocumentRevision } from "../../types";

/**
 * DocumentDetailPage: knowledge document revisions with lifecycle actions.
 */
export function DocumentDetailPage() {
  const { documentId } = useParams<{ documentId: string }>();
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const document = useApi(() => api.document(documentId!));
  const [busy, setBusy] = useState(false);

  const perms = principal?.permissions ?? [];
  const canApprove = perms.includes("knowledge:approve");
  const canWrite = perms.includes("knowledge:write");

  if (document.error) {
    return <div className="toast-error" role="alert">{document.error}</div>;
  }
  if (document.loading || !document.data) {
    return <div className="skeleton" style={{ height: 300 }} />;
  }
  const value = document.data;

  const mutate = async (action: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await action();
      await document.refetch();
    } finally {
      setBusy(false);
    }
  };

  const revisionActions = (revision: DocumentRevision) => {
    const actions: Array<{ key: string; label: string; run: () => void }> = [];
    if (revision.approval_status === "DRAFT" && canApprove) {
      actions.push({
        key: "approve",
        label: t("document.approve"),
        run: () => void mutate(() => api.approveDocumentRevision(revision.id))
      });
    }
    if (revision.ingestion_state === "AWAITING_UPLOAD" && canWrite) {
      actions.push({
        key: "ingest",
        label: t("document.requestIngestion"),
        run: () => void mutate(() => api.requestDocumentIngestion(revision.id))
      });
    }
    if (revision.approval_status !== "RETIRED" && canWrite) {
      actions.push({
        key: "retire",
        label: t("document.retire"),
        run: () => void mutate(() => api.retireDocumentRevision(revision.id, "superseded"))
      });
    }
    return actions;
  };

  return (
    <section>
      <Breadcrumbs
        trail={[
          { label: t("nav.knowledge"), to: "/knowledge" },
          { label: value.title }
        ]}
      />
      <Topbar title={value.title} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <div>
              <span className="eyebrow">{value.document_type}</span>
              <h2>{value.title}</h2>
            </div>
            <span className="source-badge">{value.authority}</span>
          </div>
          <div style={{ padding: 16 }}>
            <div>{t("document.revisions")} <span className="count">{value.revisions.length}</span></div>
          </div>
        </div>
        {value.revisions.map((revision) => (
          <div className="panel" key={revision.id}>
            <div className="panel-heading">
              <h2 className="mono">{revision.revision}</h2>
              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <span className={`state-badge ${revision.approval_status.toLowerCase()}`}>
                  {revision.approval_status}
                </span>
                <span className="state-badge">{revision.ingestion_state}</span>
              </div>
            </div>
            <div style={{ padding: 16, display: "grid", gap: 10 }}>
              <div>{t("document.language")} <span className="mono">{revision.language}</span></div>
              {revision.ingestion_error && (
                <div className="toast-error" role="alert">{revision.ingestion_error}</div>
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
                {revisionActions(revision).map((action) => (
                  <button
                    key={action.key}
                    className={action.key === "approve" ? "primary-button" : "secondary-button"}
                    disabled={busy}
                    onClick={action.run}
                  >
                    {action.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
