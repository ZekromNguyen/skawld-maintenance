import { Link } from "react-router";
import { useI18n } from "../../i18n/I18nProvider";
import { StatusBadge } from "../ui/StatusBadge";
import { DataTable } from "../ui/DataTable";
import { approvalTone, approvalLabelKey } from "../labels";
import type { KnowledgeDocument } from "../../types";

/**
 * KnowledgePanel: document registry. Each document links to its detail page;
 * revisions render as chips with an effective marker on the latest approved
 * revision (or the latest revision when none is approved yet).
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
              const effective = effectiveRevision(document);
              if (!effective) return "—";
              return (
                <span className="revision-chips">
                  {document.revisions.map((revision) => (
                    <StatusBadge
                      key={revision.id}
                      tone={approvalTone(revision.approval_status)}
                      label={`${revision.revision} · ${approvalLabelKey(revision.approval_status) ? t(approvalLabelKey(revision.approval_status)!) : revision.approval_status}${revision.id === effective.id ? ` · ${t("knowledge.effective")}` : ""}`}
                    />
                  ))}
                </span>
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

/** Effective revision: latest APPROVED, else the latest revision. */
function effectiveRevision(document: KnowledgeDocument) {
  const approved = document.revisions.filter((r) => r.approval_status === "APPROVED");
  if (approved.length > 0) return approved[approved.length - 1];
  return document.revisions[document.revisions.length - 1];
}
