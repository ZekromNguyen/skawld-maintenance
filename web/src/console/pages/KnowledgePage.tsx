import { useState } from "react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { KnowledgePanel } from "../components/KnowledgePanel";

/**
 * KnowledgePage: procedures and evidence document registry.
 */
export function KnowledgePage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0];
  const documents = useApi(() => api.documents(siteID));
  const [busy, setBusy] = useState(false);

  const refresh = async () => {
    await documents.refetch();
  };

  const mutate = async (action: () => Promise<void>) => {
    setBusy(true);
    try {
      await action();
      await refresh();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section>
      <Topbar title={t("nav.knowledge")} principal={principal} />
      {documents.error && (
        <div className="toast-error" role="alert">{documents.error}</div>
      )}
      {documents.loading && !documents.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <KnowledgePanel
          siteID={siteID}
          documents={documents.data?.items ?? []}
          busy={busy}
          onRefresh={refresh}
          onMutate={mutate}
        />
      )}
    </section>
  );
}
