import { Link } from "react-router";
import { Dot, ListChecks, Ruler, Warning } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { RelativeTime } from "../ui/RelativeTime";
import type { Locale } from "../../i18n/messages";
import { EmptyState } from "../ui/EmptyState";

export interface ActivityEntry {
  id: string;
  kind: "detected" | "execution" | "measurement" | "context";
  title: string;
  detail?: string;
  to?: string;
  time?: string;
}

const KIND_ICON = {
  detected: Warning,
  execution: ListChecks,
  measurement: Ruler,
  context: Dot,
} as const;

/**
 * ActivityFeed: a truthful timeline for a record. Only real events with
 * backing timestamps render as timed rows; entries without a timestamp
 * render as context rows. No invented history.
 */
export function ActivityFeed({ entries }: { entries: ActivityEntry[] }) {
  const { t, locale } = useI18n();
  if (entries.length === 0) {
    return <EmptyState title={t("incident.noActivity")} />;
  }
  return (
    <ol className="activity-feed" role="list">
      {entries.map((entry) => {
        const Icon = KIND_ICON[entry.kind];
        const body = (
          <>
            <Icon size={13} weight="fill" className={`activity-icon activity-icon--${entry.kind}`} aria-hidden="true" />
            <span className="activity-body">
              <strong>{entry.title}</strong>
              {entry.detail ? <small>{entry.detail}</small> : null}
            </span>
            {entry.time ? <ActivityTime time={entry.time} locale={locale} /> : <span className="activity-muted">—</span>}
          </>
        );
        return (
          <li key={entry.id} className="activity-row" role="listitem">
            {entry.to ? (
              <Link to={entry.to} className="activity-link">
                {body}
              </Link>
            ) : (
              body
            )}
          </li>
        );
      })}
    </ol>
  );
}

function ActivityTime({ time, locale }: { time: string; locale: Locale }) {
  return (
    <time className="activity-time" dateTime={time}>
      <RelativeTime time={time} locale={locale} />
    </time>
  );
}
