import { useState } from "react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { IncidentQueue } from "../components/IncidentQueue";
import { CreateIncidentForm } from "../components/CreateIncidentForm";

/**
 * IncidentsPage: incident queue with native creation (permission-gated).
 */
export function IncidentsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const incidents = useApi(() => api.incidents());
  const assets = useApi(() => api.assets());
  const [showForm, setShowForm] = useState(false);
  const [busy, setBusy] = useState(false);
  const canCreate = principal?.permissions.includes("incident:create") ?? false;

  const createIncident = async (value: {
    site_id: string;
    asset_id: string;
    summary: string;
    severity: string;
  }) => {
    setBusy(true);
    try {
      await api.createIncident(value);
      setShowForm(false);
      await incidents.refetch();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section>
      <Topbar title={t("nav.incidents")} principal={principal} />
      {incidents.error && (
        <div className="toast-error" role="alert">{incidents.error}</div>
      )}
      {showForm && (
        <>
          <div className="modal-backdrop" onClick={() => setShowForm(false)} />
          <CreateIncidentForm
            assets={assets.data?.items ?? []}
            busy={busy}
            onCancel={() => setShowForm(false)}
            onCreate={createIncident}
          />
        </>
      )}
      {incidents.loading && !incidents.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <IncidentQueue
          incidents={incidents.data?.items ?? []}
          onCreate={() => (canCreate ? setShowForm(true) : undefined)}
        />
      )}
    </section>
  );
}
