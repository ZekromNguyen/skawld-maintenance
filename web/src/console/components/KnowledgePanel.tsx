import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import { StatusBadge } from "../ui/StatusBadge";
import { DataTable } from "../ui/DataTable";
import { approvalTone, approvalLabelKey } from "../labels";
import type { KnowledgeDocument } from "../../types";

/**
 * KnowledgePanel: document registry. Each document links to its detail page;
 * revision states use tone badges. The evidence search moved to /search.
 */
export function KnowledgePanel({
  documents,
  loading,
  error,
  onRetry
}: {
  documents: KnowledgeDocument[];
  loading: boolean;
  error?: string;
  onRetry: () => void;
}) {
  const { t } = useI18n();
  return (
    <div className="panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("knowledge.validityAware")}</span>
          <h2>{t("knowledge.documentRevisions")}</h2>
        </div>
        <span className="count">{documents.length}</span>
      </div>
      <DataTable<KnowledgeDocument>
        columns={[
          {
            key: "title",
            header: t("knowledge.title"),
            render: (document) => (
              <Link to={`/knowledge/${document.id}`} className="strong">
                {document.title}
              </Link>
            ),
            sortValue: (d) => d.title
          },
          {
            key: "type",
            header: t("assets.class"),
            render: (document) => document.document_type
          },
          {
            key: "authority",
            header: t("assets.authority"),
            render: (document) => (
              <span className="source-badge">
                {document.authority === "SITE_APPROVED" ? t("assets.skawldNative") : document.authority}
              </span>
            )
          },
          {
            key: "revision",
            header: t("document.version"),
            render: (document) => {
              const latest = document.revisions[document.revisions.length - 1];
              return latest ? (
                <StatusBadge
                  tone={approvalTone(latest.approval_status)}
                  label={`${latest.revision} · ${approvalLabelKey(latest.approval_status) ? t(approvalLabelKey(latest.approval_status)!) : latest.approval_status}`}
                />
              ) : (
                "—"
              );
            }
          }
        ]}
        rows={documents}
        rowKey={(document) => document.id}
        emptyTitle={t("knowledge.noDocuments")}
        loading={loading}
        error={error}
        onRetry={onRetry}
      />
    </div>
  );
}
