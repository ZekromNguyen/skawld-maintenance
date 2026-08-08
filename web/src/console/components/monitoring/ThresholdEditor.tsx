import { useState, type FormEvent } from "react";
import { Dialog } from "../../feedback/Dialog";
import { FormField } from "../../ui/FormField";
import { useI18n } from "../../../i18n/I18nProvider";
import type { MonitorThreshold } from "../../../types";

const COMPARATORS = ["GT", "GTE", "LT", "LTE"] as const;

/**
 * ThresholdEditor: dialog for configuring one monitor threshold. Site scope
 * is org-wide (empty) in Phase 1; the edit affordance is gated by the
 * monitoring:configure permission at the call site.
 */
export function ThresholdEditor({
  open,
  metrics,
  initial,
  pending,
  onClose,
  onSave,
}: {
  open: boolean;
  metrics: string[];
  initial?: MonitorThreshold;
  pending: boolean;
  onClose: () => void;
  onSave: (value: MonitorThreshold) => void;
}) {
  const { t } = useI18n();
  const [metricKey, setMetricKey] = useState(initial?.metric_key ?? metrics[0] ?? "");
  const [comparator, setComparator] = useState<(typeof COMPARATORS)[number]>(
    initial?.comparator ?? "GT",
  );
  const [warnValue, setWarnValue] = useState(String(initial?.warn_value ?? 0));
  const [critValue, setCritValue] = useState(String(initial?.crit_value ?? 0));
  const [enabled, setEnabled] = useState(initial?.enabled ?? true);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    onSave({
      metric_key: metricKey,
      comparator,
      warn_value: Number(warnValue),
      crit_value: Number(critValue),
      enabled,
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("monitoring.configure")}
    >
      <form onSubmit={submit} className="form-stack">
        <FormField label={t("monitoring.metric")} htmlFor="threshold-metric">
          <select
            id="threshold-metric"
            value={metricKey}
            onChange={(event) => setMetricKey(event.target.value)}
            required
          >
            {metrics.map((key) => (
              <option key={key} value={key}>
                {key}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("monitoring.comparator")} htmlFor="threshold-comparator">
          <select
            id="threshold-comparator"
            value={comparator}
            onChange={(event) => setComparator(event.target.value as (typeof COMPARATORS)[number])}
          >
            {COMPARATORS.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("monitoring.warnValue")} htmlFor="threshold-warn">
          <input
            id="threshold-warn"
            type="number"
            step="any"
            value={warnValue}
            onChange={(event) => setWarnValue(event.target.value)}
            required
          />
        </FormField>
        <FormField label={t("monitoring.critValue")} htmlFor="threshold-crit">
          <input
            id="threshold-crit"
            type="number"
            step="any"
            value={critValue}
            onChange={(event) => setCritValue(event.target.value)}
            required
          />
        </FormField>
        <label className="checkbox-row">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(event) => setEnabled(event.target.checked)}
          />
          {t("monitoring.enabled")}
        </label>
        <div className="dialog-actions">
          <button type="button" className="secondary-button" onClick={onClose}>
            {t("monitoring.cancel")}
          </button>
          <button type="submit" className="primary-button" disabled={pending}>
            {t("monitoring.save")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
