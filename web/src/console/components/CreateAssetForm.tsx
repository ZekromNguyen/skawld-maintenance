import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import { FormField } from "../ui/FormField";

/**
 * CreateAssetForm: native asset creation inside a dialog. No demo defaults,
 * required validation, explicit site picker for multi-site principals.
 */
export function CreateAssetForm(props: {
  siteIDs: string[];
  pending: boolean;
  onCancel: () => void;
  onCreate: (value: { site_id: string; tag: string; name: string; class: string }) => void;
}) {
  const { t } = useI18n();
  const [siteID, setSiteID] = useState(props.siteIDs[0] ?? "");
  const [tag, setTag] = useState("");
  const [name, setName] = useState("");
  const [assetClass, setAssetClass] = useState("");
  const [touched, setTouched] = useState(false);

  const errors = {
    site: !siteID ? t("form.required") : undefined,
    tag: !tag.trim() ? t("form.required") : undefined,
    name: name.trim().length < 2 ? t("form.required") : undefined,
    class: !assetClass.trim() ? t("form.required") : undefined
  };
  const valid = !errors.site && !errors.tag && !errors.name && !errors.class;

  function submit(event: FormEvent) {
    event.preventDefault();
    setTouched(true);
    if (!valid) return;
    props.onCreate({ site_id: siteID, tag: tag.trim(), name: name.trim(), class: assetClass.trim() });
  }

  return (
    <form className="create-incident-form" onSubmit={submit} noValidate>
      <FormField label={t("assets.site")} htmlFor="asset-site" required error={touched ? errors.site : undefined}>
        <select id="asset-site" value={siteID} onChange={(event) => setSiteID(event.target.value)} required>
          {props.siteIDs.map((id) => (
            <option key={id} value={id}>
              {id}
            </option>
          ))}
        </select>
      </FormField>
      <FormField label={t("form.tag")} htmlFor="asset-tag" required error={touched ? errors.tag : undefined}>
        <input id="asset-tag" value={tag} onChange={(event) => setTag(event.target.value)} required />
      </FormField>
      <FormField label={t("form.name")} htmlFor="asset-name" required error={touched ? errors.name : undefined}>
        <input id="asset-name" value={name} onChange={(event) => setName(event.target.value)} required />
      </FormField>
      <FormField label={t("form.classLabel")} htmlFor="asset-class" required error={touched ? errors.class : undefined}>
        <input id="asset-class" value={assetClass} onChange={(event) => setAssetClass(event.target.value)} required />
      </FormField>
      <div className="dialog-footer">
        <button type="button" className="secondary-button" onClick={props.onCancel} disabled={props.pending}>
          {t("form.cancel")}
        </button>
        <button type="submit" className="primary-button" disabled={props.pending}>
          {t("form.create")}
        </button>
      </div>
    </form>
  );
}
