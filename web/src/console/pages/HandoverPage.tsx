import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { HandoverPanel } from "../components/HandoverPanel";

/**
 * HandoverPage: shift handover capture and review.
 */
export function HandoverPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0];
  const handovers = useApi(() => api.handovers());
  const [busy, setBusy] = useState(false);

  const latest = handovers.data?.items[0];

  const mutate = async (action: () => Promise<void>) => {
    setBusy(true);
    try {
      await action();
      await handovers.refetch();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section>
      <Topbar title={t("nav.handover")} principal={principal} />
      {handovers.error && (
        <div className="toast-error" role="alert">{handovers.error}</div>
      )}
      {handovers.loading && !handovers.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <HandoverPanel
          siteID={siteID}
          handover={latest}
          busy={busy}
          onPrepare={() =>
            siteID
              ? mutate(async () => {
                  await api.prepareHandover(siteID);
                })
              : Promise.resolve()
          }
          onCapture={() =>
            latest
              ? mutate(async () => {
                  await api.startDemonstration("HANDOVER", latest.id);
                  navigate("/demonstrations");
                })
              : Promise.resolve()
          }
        />
      )}
    </section>
  );
}
