import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import type { Asset } from "../../types";
import { EmptyRow } from "./EmptyRow";

/**
 * AssetsTable: source-aware asset registry with links to asset detail.
 */
export function AssetsTable({ assets, onCreate }: { assets: Asset[]; onCreate: () => void }) {
  const { t } = useI18n();
  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("assets.sourceAwareRegistry")}</span>
          <h2>{t("assets.title")}</h2>
        </div>
        <div className="heading-actions">
          <span className="count">{t("assets.count", { count: assets.length })}</span>
          <button className="secondary-button" onClick={onCreate}>{t("assets.add")}</button>
        </div>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{t("assets.tag")}</th>
              <th>{t("assets.asset")}</th>
              <th>{t("assets.class")}</th>
              <th>{t("assets.criticality")}</th>
              <th>{t("assets.authority")}</th>
              <th>{t("assets.status")}</th>
            </tr>
          </thead>
          <tbody>
            {assets.map((asset) => (
              <tr key={asset.id}>
                <td className="mono strong">
                  <Link to={`/assets/${asset.id}`} style={{ color: "inherit" }}>
                    {asset.tag}
                  </Link>
                </td>
                <td>
                  <strong>{asset.name}</strong>
                  <small>{[asset.manufacturer, asset.model].filter(Boolean).join(" · ") || t("assets.noOem")}</small>
                </td>
                <td>{asset.class}</td>
                <td>
                  <span className={`criticality rating-${asset.criticality?.rating ?? "none"}`}>
                    {asset.criticality?.rating ?? "-"}
                  </span>
                </td>
                <td>
                  <span className="source-badge">
                    {asset.source_of_truth === "EXTERNAL_REFERENCE"
                      ? t("assets.externalProjection")
                      : t("assets.skawldNative")}
                  </span>
                </td>
                <td><span className="status-dot" />{asset.status}</td>
              </tr>
            ))}
            {assets.length === 0 && <EmptyRow columns={6} labelKey="assets.empty" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}
