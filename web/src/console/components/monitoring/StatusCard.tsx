import { StatusBadge, type Tone } from "../../ui/StatusBadge";
import { Sparkline } from "./Sparkline";
import { useI18n } from "../../../i18n/I18nProvider";
import type { MonitoringMetric, MonitoringSummaryEntry } from "../../../types";

const STATUS_TONE: Record<MonitoringSummaryEntry["status"], Tone> = {
  PASS: "success",
  WARN: "high",
  CRIT: "critical",
};

const STATUS_LABEL_KEY: Record<MonitoringSummaryEntry["status"], string> = {
  PASS: "monitoring.statusPass",
  WARN: "monitoring.statusWarn",
  CRIT: "monitoring.statusCrit",
};

/**
 * StatusCard: one monitor with its latest value, derived status badge, a
 * 30-point sparkline, and the open-alert count.
 */
export function StatusCard({
  entry,
  history,
}: {
  entry: MonitoringSummaryEntry;
  history: MonitoringMetric[];
}) {
  const { t } = useI18n();
  const values = history.map((row) => row.value);
  return (
    <div className="panel status-card">
      <div className="panel-heading">
        <span className="eyebrow">{t(`monitoring.group.${entry.group}` as never)}</span>
        <StatusBadge tone={STATUS_TONE[entry.status]} label={t(STATUS_LABEL_KEY[entry.status] as never)} />
      </div>
      <div className="status-card-body">
        <div>
          <strong>{entry.metric_key}</strong>
          <span className="status-value">
            {entry.value} {entry.unit}
          </span>
        </div>
        <Sparkline values={values} />
      </div>
      <div className="status-card-foot">
        <span>
          {entry.open_alerts > 0
            ? t("monitoring.alerts") + ": " + entry.open_alerts
            : t("monitoring.noAlerts")}
        </span>
      </div>
    </div>
  );
}
