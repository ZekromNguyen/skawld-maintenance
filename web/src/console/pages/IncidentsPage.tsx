import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { Dialog } from "../feedback/Dialog";
import { DataTable } from "../ui/DataTable";
import { StatusBadge } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";
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
 * proper dialog. Client-side filter/sort; creation navigates to the detail.
 */
export function IncidentsPage() {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const incidents = useQuery(() => api.incidents().then((list) => list.items));
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const [tab, setTab] = useState<StateTab>("OPEN");
  const [severity, setSeverity] = useState("ALL");
  const [query, setQuery] = useState("");
  const [showForm, setShowForm] = useState(false);
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
    const items = incidents.data ?? [];
    return items
      .filter((incident) => tab === "ALL" || incident.state === tab)
      .filter((incident) => severity === "ALL" || incident.severity === severity)
      .filter((incident) => {
        if (!query.trim()) return true;
        const q = query.trim().toLowerCase();
        return (
          incident.number.toLowerCase().includes(q) ||
          incident.summary.toLowerCase().includes(q) ||
          (incident.asset_tag ?? "").toLowerCase().includes(q)
        );
      });
  }, [incidents.data, tab, severity, query]);

  const counts = useMemo(() => {
    const items = incidents.data ?? [];
    return {
      OPEN: items.filter((i) => i.state === "OPEN").length,
      IN_PROGRESS: items.filter((i) => i.state === "IN_PROGRESS").length,
      RESOLVED: items.filter((i) => i.state === "RESOLVED").length,
      ALL: items.length,
    } as Record<StateTab, number>;
  }, [incidents.data]);

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
          <div className="incident-tabs" role="tablist" aria-label={t("nav.incidents")}>
            {STATE_TABS.map((state) => (
              <button
                key={state}
                role="tab"
                aria-selected={tab === state}
                className={`tab${tab === state ? " active" : ""}`}
                onClick={() => setTab(state)}
              >
                {state === "ALL" ? t("incidents.tabs.all") : t(incidentStateLabelKey(state))}
                <span className="count">{counts[state]}</span>
              </button>
            ))}
          </div>
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
