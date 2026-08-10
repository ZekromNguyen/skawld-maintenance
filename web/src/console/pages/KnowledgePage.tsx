import { useState, type FormEvent, type ReactNode } from "react";
import { FormField } from "../ui/FormField";
import { api } from "../../api";
import { usePaginatedList } from "../usePaginatedList";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { Dialog } from "../feedback/Dialog";
import { KnowledgePanel } from "../components/KnowledgePanel";

/**
 * KnowledgePage: procedures and evidence document registry with controlled
 * upload inside a dialog and toast feedback on every mutation.
 */
export function KnowledgePage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteIDs = principal?.site_ids ?? [];
  const siteID = siteIDs[0];
  const documents = usePaginatedList((params) => api.documents(siteID, params), [siteID]);
  const [showForm, setShowForm] = useState(false);
  const canWrite = principal?.permissions.includes("knowledge:write") ?? false;

  const upload = useCommand(
    async (value: { site: string; title: string; assetClass: string; file: File }) => {
      const document = await api.createDocument(value.site, value.title, "SOP");
      const revision = await api.createDocumentRevision(document.id, value.site, value.assetClass);
      await api.uploadDocumentRevision(revision.id, value.site, value.file);
    },
    {
      successMessage: t("knowledge.uploadSuccess"),
      onSuccess: () => {
        setShowForm(false);
        void documents.refetch();
      },
    },
  );

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader
          title={t("nav.knowledge")}
          principal={principal}
          actions={
            canWrite ? (
              <button className="primary-button" onClick={() => setShowForm(true)}>
                {t("knowledge.newDocument")}
              </button>
            ) : undefined
          }
        />
        <KnowledgePanel
          documents={documents.items}
          loading={documents.loading}
          error={documents.error}
          onRetry={() => void documents.refetch()}
        />
        {documents.hasMore ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void documents.loadMore()}
            disabled={documents.loading}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}
          </button>
        ) : null}

        <Dialog
          open={showForm}
          onOpenChange={setShowForm}
          title={t("knowledge.newDocument")}
          footer={null}
        >
          <UploadForm
            siteIDs={siteIDs}
            pending={upload.pending}
            onCancel={() => setShowForm(false)}
            onUpload={(value) => void upload.run(value)}
          />
        </Dialog>
      </section>
    </PageTrailProvider>
  );
}

function UploadForm({
  siteIDs,
  pending,
  onCancel,
  onUpload,
}: {
  siteIDs: string[];
  pending: boolean;
  onCancel: () => void;
  onUpload: (value: { site: string; title: string; assetClass: string; file: File }) => void;
}) {
  const { t } = useI18n();
  const [site, setSite] = useState(siteIDs[0] ?? "");
  const [title, setTitle] = useState("");
  const [assetClass, setAssetClass] = useState("");
  const [file, setFile] = useState<File | undefined>(undefined);
  const [touched, setTouched] = useState(false);

  const errors = {
    site: !site ? t("form.required") : undefined,
    title: title.trim().length < 3 ? t("form.required") : undefined,
    assetClass: !assetClass.trim() ? t("form.required") : undefined,
    file: !file ? t("form.required") : undefined
  };
  const valid = !errors.site && !errors.title && !errors.assetClass && !errors.file;

  function submit(event: FormEvent) {
    event.preventDefault();
    setTouched(true);
    if (!valid || !file) return;
    onUpload({ site, title: title.trim(), assetClass: assetClass.trim(), file });
  }

  return (
    <form className="create-incident-form" onSubmit={submit} noValidate>
      <FormFieldWrapper label={t("assets.site")} error={touched ? errors.site : undefined}>
        <select value={site} onChange={(event) => setSite(event.target.value)} required>
          {siteIDs.map((id) => (
            <option key={id} value={id}>{id}</option>
          ))}
        </select>
      </FormFieldWrapper>
      <FormFieldWrapper label={t("knowledge.title")} error={touched ? errors.title : undefined}>
        <input value={title} onChange={(event) => setTitle(event.target.value)} required />
      </FormFieldWrapper>
      <FormFieldWrapper label={t("knowledge.assetClass")} error={touched ? errors.assetClass : undefined}>
        <input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} required />
      </FormFieldWrapper>
      <FormFieldWrapper label={t("knowledge.pdfOrText")} error={touched ? errors.file : undefined}>
        <input
          type="file"
          accept=".pdf,text/plain,application/pdf"
          onChange={(event) => setFile(event.target.files?.[0])}
          required
        />
      </FormFieldWrapper>
      <div className="dialog-footer">
        <button type="button" className="secondary-button" onClick={onCancel} disabled={pending}>
          {t("form.cancel")}
        </button>
        <button type="submit" className="primary-button" disabled={pending}>
          {t("knowledge.uploadQueue")}
        </button>
      </div>
    </form>
  );
}

function FormFieldWrapper({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  const id = `upload-${label}`;
  return (
    <FormField label={label} htmlFor={id} error={error} required>
      {children}
    </FormField>
  );
}
