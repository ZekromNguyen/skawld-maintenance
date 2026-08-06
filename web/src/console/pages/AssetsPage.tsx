import { useState } from "react";
import { useNavigate } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { Dialog } from "../feedback/Dialog";
import { AssetsTable } from "../components/AssetsTable";
import { CreateAssetForm } from "../components/CreateAssetForm";

/**
 * AssetsPage: asset registry with native creation inside a proper dialog.
 * The create button is hidden without asset:create; site is explicit.
 */
export function AssetsPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const [showForm, setShowForm] = useState(false);
  const canCreate = principal?.permissions.includes("asset:create") ?? false;
  const siteIDs = principal?.site_ids ?? [];

  const create = useCommand(
    (value: { site_id: string; tag: string; name: string; class: string }) => api.createAsset(value),
    {
      successMessage: t("assets.create.success"),
      onSuccess: () => {
        setShowForm(false);
        void assets.refetch();
      },
    },
  );

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader
          title={t("nav.assets")}
          principal={principal}
          actions={
            canCreate ? (
              <button className="primary-button" onClick={() => setShowForm(true)}>
                {t("assets.add")}
              </button>
            ) : undefined
          }
        />
        <AssetsTable
          assets={assets.data ?? []}
          loading={assets.loading}
          error={assets.error}
          onRetry={() => void assets.refetch()}
          onRowClick={(asset) => navigate(`/assets/${asset.id}`)}
        />
        <Dialog
          open={showForm}
          onOpenChange={setShowForm}
          title={t("form.createAsset")}
          footer={null}
        >
          <CreateAssetForm
            siteIDs={siteIDs}
            pending={create.pending}
            onCancel={() => setShowForm(false)}
            onCreate={(value) => void create.run(value)}
          />
        </Dialog>
      </section>
    </PageTrailProvider>
  );
}
