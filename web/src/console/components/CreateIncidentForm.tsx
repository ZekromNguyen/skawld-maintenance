import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import { FormField } from "../ui/FormField";
import { severityLabelKey } from "../labels";
import type { Asset } from "../../types";

const SEVERITIES = ["LOW", "MEDIUM", "HIGH", "CRITICAL"] as const;

/**
 * CreateIncidentForm: native incident creation inside a dialog. No demo
 * defaults, required validation on every field, i18n severity options.
 */
export function CreateIncidentForm(props: {
  assets: Asset[];
  pending: boolean;
  onCancel: () => void;
  onCreate: (value: {
    site_id: string;
    asset_id: string;
    summary: string;
    severity: string;
  }) => void;
}) {
  const { t } = useI18n();
  const [assetID, setAssetID] = useState("");
  const [summary, setSummary] = useState("");
  const [severity, setSeverity] = useState("");
  const [touched, setTouched] = useState(false);

  const asset = props.assets.find((item) => item.id === assetID);
  const errors = {
    asset: !assetID ? t("form.required") : undefined,
    summary: summary.trim().length < 3 ? t("form.required") : undefined,
    severity: !severity ? t("form.required") : undefined,
  };
  const valid = !errors.asset && !errors.summary && !errors.severity;

  function submit(event: FormEvent) {
    event.preventDefault();
    setTouched(true);
    if (!valid || !asset) return;
    props.onCreate({
      site_id: asset.site_id,
      asset_id: asset.id,
      summary: summary.trim(),
      severity,
    });
  }

  return (
    <form className="create-incident-form" onSubmit={submit}>
      <FormField label={t("form.asset")} htmlFor="incident-asset" required error={touched ? errors.asset : undefined}>
        <select
          id="incident-asset"
          value={assetID}
          onChange={(event) => setAssetID(event.target.value)}
          required
        >
          <option value="" disabled>
            {t("form.selectAsset")}
          </option>
          {props.assets.map((item) => (
            <option key={item.id} value={item.id}>
              {item.tag} · {item.name}
            </option>
          ))}
        </select>
      </FormField>
      <FormField label={t("form.summary")} htmlFor="incident-summary" required error={touched ? errors.summary : undefined}>
        <input
          id="incident-summary"
          value={summary}
          onChange={(event) => setSummary(event.target.value)}
          required
        />
      </FormField>
      <FormField label={t("form.severity")} htmlFor="incident-severity" required error={touched ? errors.severity : undefined}>
        <select
          id="incident-severity"
          value={severity}
          onChange={(event) => setSeverity(event.target.value)}
          required
        >
          <option value="" disabled>
            {t("incidents.tabs.all")}
          </option>
          {SEVERITIES.map((level) => (
            <option key={level} value={level}>
              {t(severityLabelKey(level))}
            </option>
          ))}
        </select>
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
