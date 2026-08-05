import { useMemo, useState, type FormEvent } from "react";
import { useI18n } from "../../i18n/I18nProvider";


export function MeasurementForm(props: {
  busy: boolean;
  onSubmit: (type: string, value: string, unit: string) => Promise<void>;
}) {
  const { t } = useI18n();
  const [type, setType] = useState("VIBRATION_VELOCITY");
  const [value, setValue] = useState("8.1");
  const unit = useMemo(() => type === "TEMPERATURE" ? "DEG_C" : "MM_PER_S", [type]);
  function submit(event: FormEvent) {
    event.preventDefault();
    void props.onSubmit(type, value, unit);
  }
  return (
    <form className="measurement-form" onSubmit={submit}>
      <span className="eyebrow">{t("measurement.record")}</span>
      <label>{t("measurement.type")}<select value={type} onChange={(event) => setType(event.target.value)}><option value="VIBRATION_VELOCITY">{t("measurement.vibrationVelocity")}</option><option value="TEMPERATURE">{t("measurement.bearingTemperature")}</option></select></label>
      <label>{t("measurement.value")}<input inputMode="decimal" value={value} onChange={(event) => setValue(event.target.value)} /></label>
      <label>{t("measurement.unit")}<input value={unit} readOnly /></label>
      <button className="secondary-button" disabled={props.busy}>{t("measurement.recordButton")}</button>
    </form>
  );
}
