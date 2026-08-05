import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * CreateAssetForm: lightweight native asset creation.
 */
export function CreateAssetForm(props: {
  siteID?: string;
  busy: boolean;
  onCancel: () => void;
  onCreate: (value: {
    site_id: string;
    tag: string;
    name: string;
    class: string;
  }) => Promise<void>;
}) {
  const { t } = useI18n();
  const [tag, setTag] = useState("P-302");
  const [name, setName] = useState("Process Pump P-302");
  const [assetClass, setAssetClass] = useState("CENTRIFUGAL_PUMP");
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!props.siteID) return;
    void props.onCreate({
      site_id: props.siteID,
      tag,
      name,
      class: assetClass
    });
  }
  return (
    <form className="command-form panel" onSubmit={submit}>
      <div><span className="eyebrow">{t("form.lightweightNative")}</span><h2>{t("form.createAsset")}</h2></div>
      <label>{t("form.tag")}<input value={tag} onChange={(event) => setTag(event.target.value)} required /></label>
      <label>{t("form.name")}<input value={name} onChange={(event) => setName(event.target.value)} required /></label>
      <label>{t("form.classLabel")}<input value={assetClass} onChange={(event) => setAssetClass(event.target.value)} required /></label>
      <div className="form-actions"><button type="button" className="secondary-button" onClick={props.onCancel}>{t("form.cancel")}</button><button className="primary-button" disabled={props.busy || !props.siteID}>{t("form.create")}</button></div>
    </form>
  );
}
