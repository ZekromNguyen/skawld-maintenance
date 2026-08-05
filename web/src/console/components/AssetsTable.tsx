import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { assetStatusTone, assetStatusLabelKey } from "../labels";
import type { Asset } from "../../types";

/**
 * AssetsTable: source-aware asset registry with tone-mapped status,
 * clickable rows, and links to asset detail.
 */
export function AssetsTable({
  assets,
  loading,
  error,
  onRetry,
  onRowClick
}: {
  assets: Asset[];
  loading: boolean;
  error?: string;
  onRetry: () => void;
  onRowClick: (asset: Asset) => void;
}) {
  const { t } = useI18n();
  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("assets.sourceAwareRegistry")}</span>
          <h2>{t("assets.title")}</h2>
        </div>
        <span className="count">{t("assets.count", { count: assets.length })}</span>
      </div>
      <DataTable<Asset>
        columns={[
          {
            key: "tag",
            header: t("assets.tag"),
            render: (asset) => (
              <Link to={`/assets/${asset.id}`} className="mono strong" onClick={(event) => event.stopPropagation()}>
                {asset.tag}
              </Link>
            ),
            sortValue: (a) => a.tag
          },
          {
            key: "name",
            header: t("assets.asset"),
            render: (asset) => (
              <>
                <strong>{asset.name}</strong>
                <small>{[asset.manufacturer, asset.model].filter(Boolean).join(" · ") || t("assets.noOem")}</small>
              </>
            ),
            sortValue: (a) => a.name
          },
          { key: "class", header: t("assets.class"), render: (asset) => asset.class, sortValue: (a) => a.class },
          {
            key: "criticality",
            header: t("assets.criticality"),
            render: (asset) => (
              <span className={`criticality rating-${asset.criticality?.rating ?? "none"}`}>
                {asset.criticality?.rating ?? "-"}
              </span>
            ),
            sortValue: (a) => a.criticality?.rating ?? ""
          },
          {
            key: "authority",
            header: t("assets.authority"),
            render: (asset) => (
              <span className="source-badge">
                {asset.source_of_truth === "EXTERNAL_REFERENCE"
                  ? t("assets.externalProjection")
                  : t("assets.skawldNative")}
              </span>
            )
          },
          {
            key: "status",
            header: t("assets.status"),
            render: (asset) => (
              <StatusBadge tone={assetStatusTone(asset.status)} label={assetStatusLabelKey(asset.status) ? t(assetStatusLabelKey(asset.status)!) : asset.status} />
            ),
            sortValue: (a) => a.status
          }
        ]}
        rows={assets}
        rowKey={(asset) => asset.id}
        onRowClick={onRowClick}
        emptyTitle={t("assets.empty")}
        loading={loading}
        error={error}
        onRetry={onRetry}
      />
    </section>
  );
}
