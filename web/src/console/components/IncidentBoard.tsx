import { Link } from "react-router";
import { Info, Warning, WarningDiamond, WarningOctagon } from "@phosphor-icons/react";
import type { Locale } from "../../i18n/messages";
import { useI18n } from "../../i18n/I18nProvider";
import type { Incident } from "../../types";
import { RelativeTime } from "../ui/RelativeTime";
import { Skeleton } from "../ui/Skeleton";
import { severityLabelKey } from "../labels";

const SEVERITY_ICONS = {
  CRITICAL: { Icon: WarningOctagon, className: "sev-critical" },
  HIGH: { Icon: Warning, className: "sev-high" },
  MEDIUM: { Icon: WarningDiamond, className: "sev-medium" },
  LOW: { Icon: Info, className: "sev-low" },
} as const;

const COLUMNS: Array<{ state: Incident["state"]; labelKey: "incident.state.open" | "incident.state.inProgress" | "incident.state.resolved" }> = [
  { state: "OPEN", labelKey: "incident.state.open" },
  { state: "IN_PROGRESS", labelKey: "incident.state.inProgress" },
  { state: "RESOLVED", labelKey: "incident.state.resolved" },
];

export function IncidentBoard(props: {
  incidents: Incident[];
  loading: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
  emptyTitle: string;
}) {
  const { t, locale } = useI18n();
  return (
    <div className="board" role="list" aria-label={t("incidents.view.board")}>
      {COLUMNS.map((column) => {
        const items = props.incidents.filter((incident) => incident.state === column.state);
        return (
          <div className="board-column" key={column.state} role="listitem">
            <div className="board-column-heading">
              <span className="board-column-title">{t(column.labelKey)}</span>
              <span className="count">{items.length}</span>
            </div>
            <div className="board-column-body">
              {props.loading ? (
                <Skeleton height={96} />
              ) : items.length > 0 ? (
                items.map((incident) => <BoardCard key={incident.id} incident={incident} locale={locale} t={t} />)
              ) : (
                <div className="board-empty">{props.emptyTitle}</div>
              )}
            </div>
          </div>
        );
      })}
      {props.hasMore && !props.loading ? (
        <button type="button" className="secondary-button board-load-more" onClick={props.onLoadMore}>
          {t("common.loadMore")}
        </button>
      ) : null}
    </div>
  );
}

function BoardCard(props: {
  incident: Incident;
  locale: Locale;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const { incident, locale, t } = props;
  const severity = SEVERITY_ICONS[incident.severity] ?? SEVERITY_ICONS.LOW;
  const SeverityIcon = severity.Icon;
  return (
    <Link
      to={`/incidents/${incident.id}`}
      className={`board-card board-card--${incident.severity.toLowerCase()}`}
    >
      <div className="board-card-top">
        <SeverityIcon size={14} weight="fill" className={severity.className} aria-label={t(severityLabelKey(incident.severity))} />
        <span className="mono">{incident.number}</span>
      </div>
      <span className="board-card-summary">{incident.summary}</span>
      <div className="board-card-meta">
        {incident.asset_tag ? <span className="board-card-asset">{incident.asset_tag}</span> : null}
        <RelativeTime time={incident.detected_at} locale={locale} />
      </div>
    </Link>
  );
}
