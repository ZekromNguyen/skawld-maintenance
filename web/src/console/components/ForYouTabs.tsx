import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";
import { Info, ListChecks, Warning, WarningDiamond, WarningOctagon } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { RelativeTime } from "../ui/RelativeTime";
import { StatusBadge, type Tone } from "../ui/StatusBadge";
import { severityTone, severityLabelKey, incidentStateTone, incidentStateLabelKey } from "../labels";
import type { Asset, Execution, Incident } from "../../types";

type ForYouTab = "recommended" | "assigned" | "starred" | "viewed";
const TABS: ForYouTab[] = ["recommended", "assigned", "starred", "viewed"];

const TAB_KEY: Record<ForYouTab, "dashboard.forYou.recommended" | "dashboard.forYou.assigned" | "dashboard.forYou.starred" | "dashboard.forYou.viewed"> = {
  recommended: "dashboard.forYou.recommended",
  assigned: "dashboard.forYou.assigned",
  starred: "dashboard.forYou.starred",
  viewed: "dashboard.forYou.viewed",
};

const TAB_STORAGE_KEY = "skawld.dashboard.foryouTab";
const RECENT_STORAGE_KEY = "skawld.dashboard.recent";
const VIEWS_STORAGE_KEY = "skawld.incidents.savedViews";

interface RecentView {
  kind: "incident" | "execution";
  id: string;
  key: string;
  title: string;
  at: number;
}

interface SavedViewShape {
  id: string;
  name: string;
}

const SEVERITY_ICONS = {
  CRITICAL: WarningOctagon,
  HIGH: Warning,
  MEDIUM: WarningDiamond,
  LOW: Info,
} as const;

function readStorage<T>(key: string): T[] {
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as T[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function writeStorage<T>(key: string, value: T[]) {
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // storage unavailable; preference is session-only
  }
}

function initialTab(): ForYouTab {
  try {
    const stored = window.localStorage.getItem(TAB_STORAGE_KEY);
    if (stored && TABS.includes(stored as ForYouTab)) return stored as ForYouTab;
  } catch {
    // storage unavailable; default to Recommended
  }
  return "recommended";
}

/** Record a detail-page visit so the Viewed tab reflects real history. */
export function recordRecentView(entry: RecentView) {
  const recent = readStorage<RecentView>(RECENT_STORAGE_KEY);
  const next = [entry, ...recent.filter((item) => !(item.kind === entry.kind && item.id === entry.id))].slice(0, 8);
  writeStorage(RECENT_STORAGE_KEY, next);
}

/**
 * ForYouTabs: the dashboard "For you" queue in Jira's pattern. Tabs are
 * status-driven queues over real data: Recommended (open incidents),
 * Assigned to me (in-progress executions), Starred (saved views), Viewed
 * (recently viewed detail pages). Counts come from the data, never
 * hardcoded; the active tab persists as a saved-view preference.
 */
export function ForYouTabs({
  incidents,
  executions,
  assets,
  pendingHandoverCount,
  loading,
  error,
  onRetry,
}: {
  incidents: Incident[];
  executions: Execution[];
  assets: Asset[];
  pendingHandoverCount: number;
  loading: boolean;
  error?: string;
  onRetry: () => void;
}) {
  const { t, locale } = useI18n();
  const [tab, setTab] = useState<ForYouTab>(initialTab);

  useEffect(() => {
    try {
      window.localStorage.setItem(TAB_STORAGE_KEY, tab);
    } catch {
      // storage unavailable; preference is session-only
    }
  }, [tab]);

  const open = useMemo(
    () => incidents.filter((incident) => incident.state !== "RESOLVED"),
    [incidents],
  );
  const inProgress = useMemo(
    () =>
      executions.filter(
        (execution) => execution.state === "ASSIGNED" || execution.state === "IN_PROGRESS",
      ),
    [executions],
  );
  const savedViews = useMemo(() => readStorage<SavedViewShape>(VIEWS_STORAGE_KEY), []);
  const recent = useMemo(() => readStorage<RecentView>(RECENT_STORAGE_KEY), []);

  const counts: Record<ForYouTab, number> = {
    recommended: open.length,
    assigned: inProgress.length,
    starred: savedViews.length,
    viewed: recent.length,
  };

  const openRow = (entry: RecentView) => {
    recordRecentView(entry);
  };

  return (
    <section aria-label={t("dashboard.recentlyViewed")}>
      <div className="for-you-tabs incident-tabs" role="tablist" aria-label={t("dashboard.recentlyViewed")}>
        {TABS.map((key) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            className={`tab${tab === key ? " active" : ""}`}
            onClick={() => setTab(key)}
          >
            {t(TAB_KEY[key])}
            <span className="count" aria-live="polite">
              {counts[key]}
            </span>
          </button>
        ))}
      </div>

      {error ? (
        <ErrorState message={error} onRetry={onRetry} />
      ) : loading ? (
        <Skeleton height={180} />
      ) : tab === "recommended" ? (
        <RecommendedQueue incidents={open} t={t} locale={locale} onOpen={openRow} />
      ) : tab === "assigned" ? (
        <AssignedQueue executions={inProgress} t={t} locale={locale} onOpen={openRow} />
      ) : tab === "starred" ? (
        <StarredQueue views={savedViews} t={t} />
      ) : (
        <ViewedQueue recent={recent} t={t} />
      )}
    </section>
  );
}

function RecommendedQueue({
  incidents,
  t,
  locale,
  onOpen,
}: {
  incidents: Incident[];
  t: ReturnType<typeof useI18n>["t"];
  locale: ReturnType<typeof useI18n>["locale"];
  onOpen: (entry: RecentView) => void;
}) {
  if (incidents.length === 0) {
    return (
      <div className="queue-empty">
        <strong>{t("dashboard.forYou.recommendedEmpty")}</strong>
        <Link to="/incidents" className="secondary-button">
          {t("dashboard.viewAll")}
        </Link>
      </div>
    );
  }
  return (
    <ul className="queue-list" role="list">
      {incidents.map((incident) => {
        const Icon = SEVERITY_ICONS[incident.severity] ?? Info;
        return (
          <li key={incident.id} role="listitem">
            <Link
              to={`/incidents/${incident.id}`}
              className="queue-row"
              onClick={() =>
                onOpen({
                  kind: "incident",
                  id: incident.id,
                  key: incident.number,
                  title: incident.summary,
                  at: Date.now(),
                })
              }
            >
              <span className={`alarm-edge alarm-edge--${incident.severity.toLowerCase()}`} aria-hidden="true" />
              <Icon size={15} weight="fill" className={`sev-${incident.severity.toLowerCase()}`} aria-hidden="true" />
              <span className="queue-row-key mono">{incident.number}</span>
              <span className="queue-row-main">
                <strong>{incident.summary}</strong>
                <small>
                  {incident.asset_tag ?? t("dashboard.table.asset")}
                  <span className="queue-row-breadcrumb">
                    <StatusBadge tone={severityTone(incident.severity)} label={t(severityLabelKey(incident.severity))} />
                    <StatusBadge tone={incidentStateTone(incident.state)} label={t(incidentStateLabelKey(incident.state))} />
                  </span>
                </small>
              </span>
              <RelativeTime time={incident.detected_at} locale={locale} />
            </Link>
          </li>
        );
      })}
    </ul>
  );
}

function AssignedQueue({
  executions,
  t,
  locale,
  onOpen,
}: {
  executions: Execution[];
  t: ReturnType<typeof useI18n>["t"];
  locale: ReturnType<typeof useI18n>["locale"];
  onOpen: (entry: RecentView) => void;
}) {
  if (executions.length === 0) {
    return (
      <div className="queue-empty">
        <strong>{t("dashboard.myQueueEmpty")}</strong>
        <Link to="/executions" className="secondary-button">
          {t("dashboard.viewAll")}
        </Link>
      </div>
    );
  }
  return (
    <ul className="queue-list" role="list">
      {executions.map((execution) => (
        <li key={execution.id} role="listitem">
          <Link
            to={`/executions/${execution.id}`}
            className="queue-row"
            onClick={() =>
              onOpen({
                kind: "execution",
                id: execution.id,
                key: execution.purpose,
                title: execution.purpose,
                at: Date.now(),
              })
            }
          >
            <span className="alarm-edge alarm-edge--info" aria-hidden="true" />
            <ListChecks size={15} weight="fill" className="sev-low" aria-hidden="true" />
            <span className="queue-row-key mono">EX-{execution.id.slice(0, 4).toUpperCase()}</span>
            <span className="queue-row-main">
              <strong>{execution.purpose}</strong>
              <small>
                {execution.asset_tag ?? t("executions.asset")}
                <span className="queue-row-breadcrumb">
                  <StatusBadge tone={executionTone(execution.state)} label={execution.state.replace("_", " ")} />
                </span>
              </small>
            </span>
            <span className="queue-row-relative muted">{t("executions.state.inProgress")}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function StarredQueue({
  views,
  t,
}: {
  views: SavedViewShape[];
  t: ReturnType<typeof useI18n>["t"];
}) {
  if (views.length === 0) {
    return (
      <div className="queue-empty">
        <strong>{t("dashboard.forYou.starredEmpty")}</strong>
        <Link to="/incidents" className="secondary-button">
          {t("incidents.views.label")}
        </Link>
      </div>
    );
  }
  return (
    <ul className="queue-list" role="list">
      {views.map((view) => (
        <li key={view.id} role="listitem">
          <Link to="/incidents" className="queue-row">
            <span className="alarm-edge alarm-edge--info" aria-hidden="true" />
            <span className="queue-row-key mono">★</span>
            <span className="queue-row-main">
              <strong>{view.name}</strong>
              <small>{t("incidents.views.label")}</small>
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function ViewedQueue({
  recent,
  t,
}: {
  recent: RecentView[];
  t: ReturnType<typeof useI18n>["t"];
}) {
  if (recent.length === 0) {
    return (
      <div className="queue-empty">
        <strong>{t("dashboard.forYou.viewedEmpty")}</strong>
      </div>
    );
  }
  return (
    <ul className="queue-list" role="list">
      {recent.map((entry) => (
        <li key={`${entry.kind}-${entry.id}`} role="listitem">
          <Link to={entry.kind === "incident" ? `/incidents/${entry.id}` : `/executions/${entry.id}`} className="queue-row">
            <span className="alarm-edge alarm-edge--info" aria-hidden="true" />
            <span className="queue-row-key mono">{entry.kind === "incident" ? entry.key : "EX"}</span>
            <span className="queue-row-main">
              <strong>{entry.title}</strong>
              <small>{t("dashboard.recentlyViewed")}</small>
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function executionTone(state: string): Tone {
  switch (state) {
    case "IN_PROGRESS":
      return "medium";
    case "COMPLETED":
      return "success";
    default:
      return "info";
  }
}
