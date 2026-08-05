import { useState } from "react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { DemonstrationPanel } from "../components/DemonstrationPanel";

/**
 * DemonstrationsPage: expert demonstration capture and review.
 */
export function DemonstrationsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0];
  const demonstrations = useApi(() => api.demonstrations(siteID));
  const [selected, setSelected] = useState<string | undefined>(undefined);
  const [busy, setBusy] = useState(false);

  const mutate = async (action: () => Promise<void>) => {
    setBusy(true);
    try {
      await action();
      await demonstrations.refetch();
    } finally {
      setBusy(false);
    }
  };

  const values = demonstrations.data?.items ?? [];
  const selectedValue = values.find((value) => value.id === selected);

  return (
    <section>
      <Topbar title={t("nav.demonstrations")} principal={principal} />
      {demonstrations.error && (
        <div className="toast-error" role="alert">{demonstrations.error}</div>
      )}
      {demonstrations.loading && !demonstrations.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <DemonstrationPanel
          values={values}
          selected={selectedValue}
          busy={busy}
          onSelect={(value) =>
            mutate(async () => {
              const detail = await api.demonstration(value.id);
              setSelected(detail.id);
              demonstrations.refetch();
            })
          }
          onComplete={(value, outcome) =>
            mutate(async () => {
              const detail = await api.completeDemonstration(value.id, outcome);
              setSelected(detail.id);
            })
          }
          onRedact={(value, eventID, path, reason) =>
            mutate(async () => {
              await api.redactDemonstrationEvent(value.id, eventID, path, reason);
              setSelected(value.id);
            })
          }
          onReview={(value, decision, reason) =>
            mutate(async () => {
              await api.reviewDemonstration(value.id, decision, reason);
              setSelected(value.id);
            })
          }
        />
      )}
    </section>
  );
}
