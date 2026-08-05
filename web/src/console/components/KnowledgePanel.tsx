import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import { api } from "../../api";
import type { Evidence, KnowledgeDocument } from "../../types";
import { EvidenceLinks } from "./EvidenceLinks";

export function KnowledgePanel(props: {
  siteID?: string;
  documents: KnowledgeDocument[];
  busy: boolean;
  onRefresh: () => Promise<void>;
  onMutate: (action: () => Promise<void>) => Promise<void>;
}) {
  const { t } = useI18n();
  const [title, setTitle] = useState("P-302 High Vibration Inspection");
  const [assetClass, setAssetClass] = useState("CENTRIFUGAL_PUMP");
  const [file, setFile] = useState<File>();
  const [query, setQuery] = useState("high vibration lubrication bearing");
  const [results, setResults] = useState<Evidence[]>([]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!props.siteID || !file) return;
    void props.onMutate(async () => {
      const document = await api.createDocument(props.siteID!, title, "SOP");
      const revision = await api.createDocumentRevision(document.id, props.siteID!, assetClass);
      await api.uploadDocumentRevision(revision.id, props.siteID!, file);
      await props.onRefresh();
    });
  }

  return (
    <div className="knowledge-layout">
      <form className="panel knowledge-form" onSubmit={submit}>
        <div className="panel-heading"><div><span className="eyebrow">{t("knowledge.controlledIngestion")}</span><h2>{t("knowledge.addRevision")}</h2></div></div>
        <label>{t("knowledge.title")}<input value={title} onChange={(event) => setTitle(event.target.value)} /></label>
        <label>{t("knowledge.assetClass")}<input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} /></label>
        <label>{t("knowledge.pdfOrText")}<input type="file" accept=".pdf,text/plain,application/pdf" onChange={(event) => setFile(event.target.files?.[0])} /></label>
        <button className="primary-button" disabled={props.busy || !props.siteID || !file}>{t("knowledge.uploadQueue")}</button>
        <p className="form-note">{t("knowledge.formNote")}</p>
      </form>
      <section className="panel">
        <div className="panel-heading"><div><span className="eyebrow">{t("knowledge.validityAware")}</span><h2>{t("knowledge.documentRevisions")}</h2></div><span className="count">{props.documents.length}</span></div>
        <div className="document-list">
          {props.documents.map((document) => (
            <article key={document.id}>
              <strong>{document.title}</strong><small>{document.document_type} · {document.authority}</small>
              {document.revisions.map((revision) => (
                <span key={revision.id} className="revision-row">
                  <span className="mono">{revision.revision}</span>
                  <span>{revision.approval_status}</span>
                  <span>{revision.ingestion_state}</span>
                </span>
              ))}
            </article>
          ))}
          {props.documents.length === 0 && <div className="empty">{t("knowledge.noDocuments")}</div>}
        </div>
      </section>
      <section className="panel search-panel">
        <div className="panel-heading"><div><span className="eyebrow">{t("knowledge.authorizationFirstRrf")}</span><h2>{t("knowledge.evidenceSearch")}</h2></div></div>
        <div className="search-row"><input value={query} onChange={(event) => setQuery(event.target.value)} /><button className="secondary-button" disabled={!props.siteID || props.busy} onClick={() => void props.onMutate(async () => setResults((await api.searchKnowledge(props.siteID!, query)).items))}>{t("knowledge.search")}</button></div>
        <EvidenceLinks evidence={results} selected={results.map((item) => item.id)} />
      </section>
    </div>
  );
}
