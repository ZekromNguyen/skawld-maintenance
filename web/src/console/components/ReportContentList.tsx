import type { Locale, MessageKey } from "../../i18n/messages";
import { useI18n } from "../../i18n/I18nProvider";
import type { ReportContentItem } from "../../types";
import { EmptyState } from "../ui/EmptyState";
import { StatusBadge, type Tone } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";

export type ReportContentKind = "measurements" | "observations" | "actions" | "unknowns";

type Translate = (key: MessageKey) => string;

const MEASUREMENT_TYPE_KEYS: Record<string, MessageKey> = {
  VIBRATION_VELOCITY: "measurement.vibrationVelocity",
  TEMPERATURE: "measurement.bearingTemperature"
};

const UNIT_KEYS: Record<string, MessageKey> = {
  MM_PER_S: "report.unit.MM_PER_S",
  IN_PER_S: "report.unit.IN_PER_S",
  DEG_C: "report.unit.DEG_C",
  DEG_F: "report.unit.DEG_F"
};

const QUALITY_META: Record<string, { key: MessageKey; tone: Tone }> = {
  GOOD: { key: "report.quality.GOOD", tone: "success" },
  QUESTIONABLE: { key: "report.quality.QUESTIONABLE", tone: "medium" },
  BAD: { key: "report.quality.BAD", tone: "critical" },
  UNKNOWN: { key: "report.quality.UNKNOWN", tone: "info" }
};

const VERIFICATION_META: Record<string, { key: MessageKey; tone: Tone }> = {
  UNVERIFIED: { key: "report.verification.UNVERIFIED", tone: "info" },
  VERIFIED: { key: "report.verification.VERIFIED", tone: "success" },
  REJECTED: { key: "report.verification.REJECTED", tone: "critical" }
};

const ACTION_TYPE_KEYS: Record<string, MessageKey> = {
  INSPECTED: "report.action.INSPECTED",
  CLEANED: "report.action.CLEANED",
  LUBRICATED: "report.action.LUBRICATED",
  ADJUSTED: "report.action.ADJUSTED",
  REPLACED: "report.action.REPLACED",
  OTHER: "report.action.OTHER"
};

const OUTCOME_META: Record<string, { key: MessageKey; tone: Tone }> = {
  COMPLETED: { key: "report.outcome.COMPLETED", tone: "success" }
};

function field(item: Record<string, unknown>, key: string): string {
  const value = item[key];
  return typeof value === "string" && value !== "" ? value : "";
}

function label(keys: Record<string, MessageKey>, raw: string, t: Translate): string {
  if (raw === "") return "";
  const key = keys[raw];
  return key ? t(key) : raw;
}

function observedTime(item: Record<string, unknown>, locale: Locale) {
  const time = field(item, "observed_at") || field(item, "performed_at");
  if (time === "") return null;
  return <RelativeTime time={time} locale={locale} />;
}

function fallbackText(item: Record<string, unknown>): string {
  const parts: string[] = [];
  for (const value of Object.values(item)) {
    if (typeof value === "string" && value !== "") parts.push(value);
    else if (typeof value === "number") parts.push(String(value));
  }
  return parts.join(" · ");
}

function StructuredItem(props: {
  kind: ReportContentKind;
  item: Record<string, unknown>;
  t: Translate;
  locale: Locale;
}) {
  const { kind, item, t, locale } = props;

  if (kind === "measurements") {
    const value = field(item, "value");
    const unit = label(UNIT_KEYS, field(item, "unit"), t);
    const type = label(MEASUREMENT_TYPE_KEYS, field(item, "type") || field(item, "measurement_type"), t);
    const quality = QUALITY_META[field(item, "quality") || field(item, "data_quality")];
    return (
      <div className="report-item">
        <div className="report-item-main">
          <span className="mono strong">{unit !== "" ? `${value} ${unit}` : value}</span>
          {type !== "" ? <span>{type}</span> : null}
          {quality ? <StatusBadge tone={quality.tone} label={t(quality.key)} /> : null}
          {observedTime(item, locale)}
        </div>
      </div>
    );
  }

  if (kind === "observations") {
    const narrative = field(item, "narrative") || fallbackText(item);
    const verification = VERIFICATION_META[field(item, "verification") || field(item, "verification_status")];
    return (
      <div className="report-item">
        <div className="report-item-main">
          <span>{narrative}</span>
          {verification ? <StatusBadge tone={verification.tone} label={t(verification.key)} /> : null}
          {observedTime(item, locale)}
        </div>
      </div>
    );
  }

  if (kind === "actions") {
    const narrative = field(item, "narrative") || fallbackText(item);
    const type = label(ACTION_TYPE_KEYS, field(item, "type") || field(item, "action_type"), t);
    const outcome = OUTCOME_META[field(item, "outcome")];
    return (
      <div className="report-item">
        <div className="report-item-main">
          <span>{narrative}</span>
          {outcome ? <StatusBadge tone={outcome.tone} label={t(outcome.key)} /> : null}
          {observedTime(item, locale)}
        </div>
        {type !== "" ? <div className="report-item-sub">{type}</div> : null}
      </div>
    );
  }

  const narrative = field(item, "narrative") || fallbackText(item);
  return (
    <div className="report-item">
      <div className="report-item-main">
        <span>{narrative}</span>
      </div>
    </div>
  );
}

export function ReportContentList(props: {
  kind: ReportContentKind;
  items: ReportContentItem[];
  emptyTitle: string;
}) {
  const { t, locale } = useI18n();
  if (props.items.length === 0) {
    return <EmptyState title={props.emptyTitle} />;
  }
  return (
    <div className="report-list">
      {props.items.map((item, index) =>
        typeof item === "string" ? (
          <div className="report-item" key={index}>
            <div className="report-item-main">
              <span>{item}</span>
            </div>
          </div>
        ) : (
          <StructuredItem key={index} kind={props.kind} item={item} t={t} locale={locale} />
        )
      )}
    </div>
  );
}
