import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { usePaginatedList } from "../usePaginatedList";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { Dialog } from "../feedback/Dialog";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";
import { Tabs } from "../ui/Tabs";
import { CreateIncidentForm } from "../components/CreateIncidentForm";
import {
  severityTone,
  severityLabelKey,
  incidentStateTone,
  incidentStateLabelKey,
} from "../labels";
import type { Incident } from "../../types";

type StateTab = "OPEN" | "IN_PROGRESS" | "RESOLVED" | "ALL";
const STATE_TABS: StateTab[] = ["OPEN", "IN_PROGRESS", "RESOLVED", "ALL"];

/**
 * IncidentsPage: filterable incident queue with native creation inside a
 * proper dialog. Server-side state/severity filtering; search applies to the
 * loaded page; creation navigates to the detail.
 */
export function IncidentsPage() {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids?.[0];
  const [tab, setTab] = useState<StateTab>("OPEN");
  const [severity, setSeverity] = useState("ALL");
  const incidents = usePaginatedList(
    (params) =>
      api.incidents({
        ...params,
        site_id: siteID,
        state: tab === "ALL" ? undefined : [tab],
        severity: severity === "ALL" ? undefined : severity,
      }),
    [tab, severity, siteID],
  );
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const summary = useQuery(() => api.summary(siteID), [siteID]);
  const [query, setQuery] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [params] = useSearchParams();
  useEffect(() => {
    if (params.get("create") === "1") setShowForm(true);
    const fromParams = params.get("state");
    if (fromParams && STATE_TABS.includes(fromParams as StateTab)) {
      setTab(fromParams as StateTab);
    }
  }, [params]);
  const canCreate = principal?.permissions.includes("incident:create") ?? false;

  const create = useCommand(
    (value: { site_id: string; asset_id: string; summary: string; severity: string }) =>
      api.createIncident(value),
    {
      successMessage: t("incidents.create.success"),
      onSuccess: (incident) => navigate(`/incidents/${incident.id}`),
    },
  );

  const rows = useMemo(() => {
    const items = incidents.items;
    if (!query.trim()) return items;
    const q = query.trim().toLowerCase();
    return items.filter(
      (incident) =>
        incident.number.toLowerCase().includes(q) ||
        incident.summary.toLowerCase().includes(q) ||
        (incident.asset_tag ?? "").toLowerCase().includes(q),
    );
  }, [incidents.items, query]);

  const counts = {
    OPEN: summary.data?.open_incidents ?? 0,
    IN_PROGRESS: summary.data?.in_progress_incidents ?? 0,
    RESOLVED: summary.data?.resolved_incidents ?? 0,
    ALL: summary.data?.total_incidents ?? 0,
  } as Record<StateTab, number>;

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
          <Tabs
            label={t("nav.incidents")}
            tabs={STATE_TABS.map((state) => ({
              id: state,
              label: state === "ALL" ? t("incidents.tabs.all") : t(incidentStateLabelKey(state)),
              count: counts[state],
            }))}
            active={tab}
            onChange={(id) => setTab(id as StateTab)}
          />
          <input
            aria-label={t("incidents.filter.search")}
            placeholder={t("incidents.filter.search")}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="incident-search"
          />
          <select
            aria-label={t("incidents.filter.severity")}
            value={severity}
            onChange={(event) => setSeverity(event.target.value)}
            className="incident-severity-filter"
          >
            <option value="ALL">{t("incidents.tabs.all")}</option>
            <option value="LOW">{t(severityLabelKey("LOW"))}</option>
            <option value="MEDIUM">{t(severityLabelKey("MEDIUM"))}</option>
            <option value="HIGH">{t(severityLabelKey("HIGH"))}</option>
            <option value="CRITICAL">{t(severityLabelKey("CRITICAL"))}</option>
          </select>
        </div>
        <p className="section-lead">{t("incidents.filter.searchHint")}</p>
        <DataTable<Incident>
          columns={[
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
                <StatusBadge tone={severityTone(incident.severity)} label={t(severityLabelKey(incident.severity))} />
              ),
              sortValue: (i) => i.severity,
            },
            {
              key: "state",
              header: t("dashboard.table.state"),
              render: (incident) => (
                <StatusBadge tone={incidentStateTone(incident.state)} label={t(incidentStateLabelKey(incident.state))} />
              ),
              sortValue: (i) => i.state,
            },
            {
              key: "age",
              header: t("dashboard.table.age"),
              render: (incident) => <RelativeTime time={incident.detected_at} locale={locale} />,
            },
          ]}
          rows={rows}
          rowKey={(incident) => incident.id}
          onRowClick={(incident) => navigate(`/incidents/${incident.id}`)}
          emptyTitle={t("incidents.noResults")}
          loading={incidents.loading}
          error={incidents.error}
          onRetry={() => void incidents.refetch()}
        />
        {incidents.hasMore ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void incidents.loadMore()}
            disabled={incidents.loading}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}
            {incidents.items.length > 0 && summary.data
              ? ` · ${incidents.items.length} / ${summary.data.total_incidents}`
              : ""}
          </button>
        ) : null}
        <Dialog
          open={showForm}
          onOpenChange={setShowForm}
          title={t("form.createIncident")}
          footer={null}
        >
          <CreateIncidentForm
            assets={assets.data ?? []}
            pending={create.pending}
            onCancel={() => setShowForm(false)}
            onCreate={(value) => void create.run(value)}
          />
        </Dialog>
      </section>
    </PageTrailProvider>
  );
}
