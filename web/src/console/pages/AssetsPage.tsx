import { useState } from "react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { AssetsTable } from "../components/AssetsTable";
import { CreateAssetForm } from "../components/CreateAssetForm";

/**
 * AssetsPage: asset registry with native creation (permission-gated).
 */
export function AssetsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const assets = useApi(() => api.assets());
  const [showForm, setShowForm] = useState(false);
  const [busy, setBusy] = useState(false);
  const canCreate = principal?.permissions.includes("asset:create") ?? false;

  const createAsset = async (value: { site_id: string; tag: string; name: string; class: string }) => {
    setBusy(true);
    try {
      await api.createAsset(value);
      setShowForm(false);
      await assets.refetch();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section>
      <Topbar title={t("nav.assets")} principal={principal} />
      {assets.error && (
        <div className="toast-error" role="alert">{assets.error}</div>
      )}
      {showForm && (
        <>
          <div className="modal-backdrop" onClick={() => setShowForm(false)} />
          <CreateAssetForm
            siteID={principal?.site_ids[0]}
            busy={busy}
            onCancel={() => setShowForm(false)}
            onCreate={createAsset}
          />
        </>
      )}
      {assets.loading && !assets.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <AssetsTable
          assets={assets.data?.items ?? []}
          onCreate={() => (canCreate ? setShowForm(true) : undefined)}
        />
      )}
    </section>
  );
}
