import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import type { Asset } from "../../types";

/**
 * CreateIncidentForm: lightweight native incident creation.
 */
export function CreateIncidentForm(props: {
  assets: Asset[];
  busy: boolean;
  onCancel: () => void;
  onCreate: (value: {
    site_id: string;
    asset_id: string;
    summary: string;
    severity: string;
  }) => Promise<void>;
}) {
  const { t } = useI18n();
  const [assetID, setAssetID] = useState(props.assets[0]?.id ?? "");
  const [summary, setSummary] = useState("High vibration");
  const [severity, setSeverity] = useState("HIGH");
  const asset = props.assets.find((item) => item.id === assetID);
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!asset) return;
    void props.onCreate({
      site_id: asset.site_id,
      asset_id: asset.id,
      summary,
      severity
    });
  }
  return (
    <form className="command-form panel incident-command" onSubmit={submit}>
      <div><span className="eyebrow">{t("form.abnormalCondition")}</span><h2>{t("form.createIncident")}</h2></div>
      <label>{t("form.asset")}<select value={assetID} onChange={(event) => setAssetID(event.target.value)} required><option value="" disabled>{t("form.selectAsset")}</option>{props.assets.map((item) => <option key={item.id} value={item.id}>{item.tag} · {item.name}</option>)}</select></label>
      <label>{t("form.summary")}<input value={summary} onChange={(event) => setSummary(event.target.value)} required /></label>
      <label>{t("form.severity")}<select value={severity} onChange={(event) => setSeverity(event.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option><option>CRITICAL</option></select></label>
      <div className="form-actions"><button type="button" className="secondary-button" onClick={props.onCancel}>{t("form.cancel")}</button><button className="primary-button" disabled={props.busy || !asset}>{t("form.create")}</button></div>
    </form>
  );
}
