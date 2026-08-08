import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Kanban, MagnifyingGlass, Rows } from "@phosphor-icons/react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { usePaginatedList } from "../usePaginatedList";
import { useCommand } from "../useCommand";
import { useListKeyboard } from "../useListKeyboard";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { Dialog } from "../feedback/Dialog";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";
import { initials, avatarColor } from "../ui/avatar";
import { CreateIncidentForm, type CreateIncidentValue } from "../components/CreateIncidentForm";
import { IncidentBoard } from "../components/IncidentBoard";
import { SavedViews, type SavedView } from "../components/SavedViews";
import {
  priorityLabelKey,
  incidentStatusTone,
  incidentStatusLabelKey,
} from "../labels";
import type { Incident } from "../../types";

type StateTab = "OPEN" | "IN_PROGRESS" | "RESOLVED" | "CLOSED" | "ALL";
const STATE_TABS: StateTab[] = ["OPEN", "IN_PROGRESS", "RESOLVED", "CLOSED", "ALL"];

type ViewMode = "list" | "board";
const VIEW_STORAGE_KEY = "skawld.incidents.view";
const SAVED_VIEWS_KEY = "skawld.incidents.savedViews";

function initialView(): ViewMode {
  try {
    return localStorage.getItem(VIEW_STORAGE_KEY) === "board" ? "board" : "list";
  } catch {
    return "list";
  }
}

function readSavedViews(seed: SavedView[]): SavedView[] {
  try {
    const raw = localStorage.getItem(SAVED_VIEWS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as SavedView[];
      if (Array.isArray(parsed)) return parsed;
    }
  } catch {
    // storage unavailable or corrupt; fall through to the seed
  }
  return seed;
}

function writeSavedViews(views: SavedView[]) {
  try {
    localStorage.setItem(SAVED_VIEWS_KEY, JSON.stringify(views));
  } catch {
    // storage unavailable — session-only views
  }
}

/**
 * IncidentsPage: filterable incident queue with native creation inside a
 * proper dialog. Client-side filter/sort; creation navigates to the detail.
 */
export function IncidentsPage() {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const incidents = usePaginatedList((params) => api.incidents(params), []);
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const teams = useQuery(() => api.teams().then((list) => list.items));
  const people = useQuery(() => api.people().then((list) => list.items));
  const [tab, setTab] = useState<StateTab>("OPEN");
  const [priority, setPriority] = useState("ALL");
  const [query, setQuery] = useState("");
  const [view, setView] = useState<ViewMode>(initialView);
  const [showForm, setShowForm] = useState(false);
  const [activeViewId, setActiveViewId] = useState<string | null>(null);
  const [savedViews, setSavedViews] = useState<SavedView[]>(() =>
    readSavedViews([
      {
        id: "my-critical",
        name: t("incidents.views.myCritical"),
        priority: "CRITICAL",
        query: "",
      },
    ]),
  );
  const [params, setParams] = useSearchParams();
  useEffect(() => {
    if (params.get("create") === "1") {
      setShowForm(true);
      params.delete("create");
      setParams(params, { replace: true });
    }
  }, [params, setParams]);
  const canCreate = principal?.permissions.includes("incident:create") ?? false;

  function switchView(next: ViewMode) {
    setView(next);
    try {
      localStorage.setItem(VIEW_STORAGE_KEY, next);
    } catch {
      // storage unavailable — session-only preference
    }
  }

  function saveCurrentView() {
    const next: SavedView = {
      id: typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : String(Date.now()),
      name: t("incidents.views.defaultName"),
      priority,
      query,
    };
    const updated = [...savedViews, next];
    setSavedViews(updated);
    writeSavedViews(updated);
    setActiveViewId(next.id);
  }

  function applyView(id: string) {
    const view = savedViews.find((item) => item.id === id);
    if (!view) return;
    setPriority(view.priority);
    setQuery(view.query);
    setActiveViewId(id);
  }

  function deleteView(id: string) {
    const updated = savedViews.filter((item) => item.id !== id);
    setSavedViews(updated);
    writeSavedViews(updated);
    if (activeViewId === id) setActiveViewId(null);
  }

  const create = useCommand(
    async (value: CreateIncidentValue) => {
      const incident = await api.createIncident(value);
      for (const file of value.files) {
        await api.uploadIncidentAttachment(incident.id, value.site_id, file);
      }
      return incident;
    },
    {
      successMessage: t("incidents.create.success"),
      onSuccess: (incident) => navigate(`/incidents/${incident.id}`),
    },
  );

  const filtered = useMemo(() => {
    const items = incidents.items;
    return items
      .filter((incident) => priority === "ALL" || incident.priority === priority)
      .filter((incident) => {
        if (!query.trim()) return true;
        const q = query.trim().toLowerCase();
        return (
          incident.number.toLowerCase().includes(q) ||
          incident.summary.toLowerCase().includes(q) ||
          (incident.asset_tag ?? "").toLowerCase().includes(q)
        );
      });
  }, [incidents.items, priority, query]);

  const rows = useMemo(
    () => filtered.filter((incident) => tab === "ALL" || incident.status === tab),
    [filtered, tab],
  );

  const { selectedIndex, setSelectedIndex, onKeyDown } = useListKeyboard<Incident>({
    rows,
    onOpen: (incident) => navigate(`/incidents/${incident.id}`),
  });
  const selectedIncident = selectedIndex >= 0 ? rows[selectedIndex] : undefined;

  const counts = useMemo(() => {
    const items = filtered;
    return {
      OPEN: items.filter((i) => i.status === "OPEN").length,
      IN_PROGRESS: items.filter((i) => i.status === "IN_PROGRESS").length,
      RESOLVED: items.filter((i) => i.status === "RESOLVED").length,
      CLOSED: items.filter((i) => i.status === "CLOSED").length,
      ALL: items.length,
    } as Record<StateTab, number>;
  }, [filtered]);

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader
          title={t("nav.incidents")}
          principal={principal}
          actions={
            canCreate ? (
              <button className="primary-button" onClick={() => setShowForm(true)}>
                {t("form.createIncident")}
              </button>
            ) : undefined
          }
        />
        <div className="incident-toolbar">
          <SavedViews
            views={savedViews}
            activeId={activeViewId}
            onSave={saveCurrentView}
            onApply={applyView}
            onDelete={deleteView}
          />
          <div className="incident-filter-row">
          {view === "list" && (
            <div className="incident-tabs" role="tablist" aria-label={t("nav.incidents")}>
              {STATE_TABS.map((state) => (
                <button
                  key={state}
                  role="tab"
                  aria-selected={tab === state}
                  className={`tab${tab === state ? " active" : ""}`}
                  onClick={() => setTab(state)}
                >
                  {state === "ALL" ? t("incidents.tabs.all") : t(incidentStatusLabelKey(state))}
                  <span className="count">{counts[state]}</span>
                </button>
              ))}
            </div>
          )}
          <div className="search-field">
            <MagnifyingGlass size={13} className="search-icon" aria-hidden="true" />
            <input
              aria-label={t("incidents.filter.search")}
              placeholder={t("incidents.filter.search")}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="incident-search"
            />
          </div>
          <select
            aria-label={t("incidents.filter.severity")}
            value={priority}
            onChange={(event) => setPriority(event.target.value)}
            className="incident-severity-filter"
          >
            <option value="ALL">{t("incidents.tabs.all")}</option>
            <option value="LOW">{t(priorityLabelKey("LOW"))}</option>
            <option value="MEDIUM">{t(priorityLabelKey("MEDIUM"))}</option>
            <option value="HIGH">{t(priorityLabelKey("HIGH"))}</option>
            <option value="CRITICAL">{t(priorityLabelKey("CRITICAL"))}</option>
          </select>
          <div className="view-toggle" role="group" aria-label={t("incidents.view.label")}>
            <button
              type="button"
              className={view === "list" ? "view-toggle-button active" : "view-toggle-button"}
              aria-pressed={view === "list"}
              title={t("incidents.view.list")}
              onClick={() => switchView("list")}
            >
              <Rows size={14} weight="bold" />
              {t("incidents.view.list")}
            </button>
            <button
              type="button"
              className={view === "board" ? "view-toggle-button active" : "view-toggle-button"}
              aria-pressed={view === "board"}
              title={t("incidents.view.board")}
              onClick={() => switchView("board")}
            >
              <Kanban size={14} weight="bold" />
              {t("incidents.view.board")}
            </button>
          </div>
          </div>
        </div>
        {view === "board" ? (
          <IncidentBoard
            incidents={filtered}
            loading={incidents.loading}
            hasMore={incidents.hasMore}
            onLoadMore={() => void incidents.loadMore()}
            emptyTitle={t("incidents.noResults")}
          />
        ) : (
        <DataTable<Incident>
          columns={[
            {
              key: "edge",
              header: "",
              render: () => <span className="alarm-edge" aria-hidden="true" />,
            },
            {
              key: "number",
              header: t("dashboard.table.incident"),
              render: (incident) => (
                <Link to={`/incidents/${incident.id}`} className="strong">
                  {incident.number}
                </Link>
              ),
              sortValue: (i) => i.number,
            },
            {
              key: "summary",
              header: t("form.summary"),
              render: (incident) => <span className="summary-cell">{incident.summary}</span>,
            },
            {
              key: "asset",
              header: t("dashboard.table.asset"),
              render: (incident) => incident.asset_tag ?? "—",
            },
            {
              key: "severity",
              header: t("dashboard.table.severity"),
              render: (incident) => (
                <span className="priority-cell">
                  <span
                    className={`priority-dot priority-dot--${incident.priority.toLowerCase()}`}
                    aria-hidden="true"
                  />
                  {t(priorityLabelKey(incident.priority))}
                </span>
              ),
              sortValue: (i) => i.priority,
            },
            {
              key: "state",
              header: t("dashboard.table.state"),
              render: (incident) => (
                <StatusBadge tone={incidentStatusTone(incident.status)} label={t(incidentStatusLabelKey(incident.status))} />
              ),
              sortValue: (i) => i.status,
            },
            {
              key: "assignee",
              header: t("incident.assignee"),
              render: (incident) =>
                incident.assignee_name ? (
                  <span className="assignee-cell">
                    <span
                      className="assignee-avatar"
                      style={{ backgroundColor: avatarColor(incident.assignee_name) }}
                      aria-hidden="true"
                    >
                      {initials(incident.assignee_name)}
                    </span>
                    {incident.assignee_name}
                  </span>
                ) : (
                  "—"
                ),
              sortValue: (i) => i.assignee_name ?? "",
            },
            {
              key: "team",
              header: t("incident.team"),
              render: (incident) => incident.team_name ?? "—",
              sortValue: (i) => i.team_name ?? "",
            },
            {
              key: "age",
              header: t("incident.date"),
              render: (incident) => (
                <RelativeTime time={incident.occurred_at ?? incident.detected_at} locale={locale} />
              ),
            },
          ]}
          rows={rows}
          rowKey={(incident) => incident.id}
          rowClassName={(incident) => `alarm-row alarm-row--${incident.priority.toLowerCase()}`}
          onRowClick={(incident) => navigate(`/incidents/${incident.id}`)}
          onTableKeyDown={(event) => {
            if (view === "list") onKeyDown(event);
          }}
          selectedKey={selectedIncident?.id}
          emptyTitle={t("incidents.noResults")}
          loading={incidents.loading}
          error={incidents.error}
          onRetry={() => void incidents.refetch()}
        />
        )}
        {view === "list" && incidents.hasMore ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void incidents.loadMore()}
            disabled={incidents.loading}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}
          </button>
        ) : null}
        <Dialog
          open={showForm}
          onOpenChange={setShowForm}
          title={t("form.createIncident")}
          footer={null}
          wide
        >
          <CreateIncidentForm
            assets={assets.data ?? []}
            people={people.data ?? []}
            teams={teams.data ?? []}
            principal={principal}
            pending={create.pending}
            onCancel={() => setShowForm(false)}
            onCreate={(value) => void create.run(value)}
          />
        </Dialog>
      </section>
    </PageTrailProvider>
  );
}
