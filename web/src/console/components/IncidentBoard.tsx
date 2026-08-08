import { Link } from "react-router";
import { Info, Warning, WarningDiamond, WarningOctagon } from "@phosphor-icons/react";
import type { Locale } from "../../i18n/messages";
import { useI18n } from "../../i18n/I18nProvider";
import type { Incident } from "../../types";
import { RelativeTime } from "../ui/RelativeTime";
import { Skeleton } from "../ui/Skeleton";
import { priorityLabelKey } from "../labels";
import { initials, avatarColor } from "../ui/avatar";

const SEVERITY_ICONS = {
  CRITICAL: { Icon: WarningOctagon, className: "sev-critical" },
  HIGH: { Icon: Warning, className: "sev-high" },
  MEDIUM: { Icon: WarningDiamond, className: "sev-medium" },
  LOW: { Icon: Info, className: "sev-low" },
} as const;

const COLUMNS: Array<{ state: Incident["status"]; labelKey: "incident.status.open" | "incident.status.inProgress" | "incident.status.resolved" | "incident.status.closed" }> = [
  { state: "OPEN", labelKey: "incident.status.open" },
  { state: "IN_PROGRESS", labelKey: "incident.status.inProgress" },
  { state: "RESOLVED", labelKey: "incident.status.resolved" },
  { state: "CLOSED", labelKey: "incident.status.closed" },
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
        const items = props.incidents.filter((incident) => incident.status === column.state);
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
  const severity = SEVERITY_ICONS[incident.priority] ?? SEVERITY_ICONS.LOW;
  const SeverityIcon = severity.Icon;
  return (
    <Link
      to={`/incidents/${incident.id}`}
      className={`board-card board-card--${incident.priority.toLowerCase()}`}
    >
      <div className="board-card-top">
        <SeverityIcon size={14} weight="fill" className={severity.className} aria-label={t(priorityLabelKey(incident.priority))} />
        <span className="mono">{incident.number}</span>
      </div>
      <span className="board-card-summary">{incident.summary}</span>
      <div className="board-card-meta">
        {incident.asset_tag ? <span className="board-card-asset">{incident.asset_tag}</span> : null}
        {incident.assignee_name ? (
          <span
            className="board-avatar"
            style={{ backgroundColor: avatarColor(incident.assignee_name) }}
            title={incident.assignee_name}
            aria-label={incident.assignee_name}
          >
            {initials(incident.assignee_name)}
          </span>
        ) : null}
        <RelativeTime time={incident.detected_at} locale={locale} />
      </div>
    </Link>
  );
}
