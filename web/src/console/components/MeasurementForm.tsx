import { useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import { FormField } from "../ui/FormField";

const TYPES: Array<{
  value: string;
  labelKey: "measurement.vibrationVelocity" | "measurement.bearingTemperature";
  unit: string;
}> = [
  { value: "VIBRATION_VELOCITY", labelKey: "measurement.vibrationVelocity", unit: "MM_PER_S" },
  { value: "TEMPERATURE", labelKey: "measurement.bearingTemperature", unit: "DEG_C" }
];

/**
 * MeasurementForm: validated, no demo defaults, unit shown as a suffix
 * (not a read-only input), form resets after a successful submit.
 */
export function MeasurementForm(props: {
  pending: boolean;
  onSubmit: (type: string, value: string, unit: string) => void;
}) {
  const { t } = useI18n();
  const [type, setType] = useState(TYPES[0].value);
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | undefined>(undefined);
  const selected = TYPES.find((item) => item.value === type) ?? TYPES[0];

  function submit(event: FormEvent) {
    event.preventDefault();
    const numeric = Number(value);
    if (!value.trim() || !Number.isFinite(numeric)) {
      setError(t("workbench.invalidMeasurement"));
      return;
    }
    setError(undefined);
    props.onSubmit(type, value.trim(), selected.unit);
    setValue("");
  }

  return (
    <form className="measurement-form" onSubmit={submit} noValidate>
      <span className="eyebrow">{t("measurement.record")}</span>
      <FormField label={t("measurement.type")} htmlFor="measurement-type">
        <select id="measurement-type" value={type} onChange={(event) => setType(event.target.value)}>
          {TYPES.map((item) => (
            <option key={item.value} value={item.value}>
              {t(item.labelKey)}
            </option>
          ))}
        </select>
      </FormField>
      <FormField
        label={t("measurement.value")}
        htmlFor="measurement-value"
        error={error}
        required
      >
        <input
          id="measurement-value"
          inputMode="decimal"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          required
        />
      </FormField>
      <FormField label={t("measurement.unit")} htmlFor="measurement-unit">
        <input id="measurement-unit" value={selected.unit} readOnly aria-readonly="true" />
      </FormField>
      <button className="secondary-button" disabled={props.pending}>
        {t("measurement.recordButton")}
      </button>
    </form>
  );
}
