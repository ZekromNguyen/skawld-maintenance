import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { HandoverPanel } from "../components/HandoverPanel";

/**
 * HandoverPage: shift handover capture, transitions, and review.
 * DRAFT -> Submit (handover:write); SUBMITTED -> Accept (handover:accept);
 * ACCEPTED -> Acknowledge (handover:accept).
 */
export function HandoverPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0];
  const handovers = useApi(() => api.handovers());
  const [busy, setBusy] = useState(false);

  const latest = handovers.data?.items[0];
  const perms = principal?.permissions ?? [];

  const mutate = async (action: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await action();
      await handovers.refetch();
    } finally {
      setBusy(false);
    }
  };

  const transitions: Array<{ label: string; run: () => void }> = [];
  if (latest) {
    if (latest.state === "DRAFT" && perms.includes("handover:write")) {
      transitions.push({
        label: t("handover.submit"),
        run: () => void mutate(() => api.submitHandover(latest.id))
      });
    }
    if (latest.state === "SUBMITTED" && perms.includes("handover:accept")) {
      transitions.push({
        label: t("handover.accept"),
        run: () => void mutate(() => api.acceptHandover(latest.id))
      });
    }
    if (latest.state === "ACCEPTED" && perms.includes("handover:accept")) {
      transitions.push({
        label: t("handover.acknowledge"),
        run: () => void mutate(() => api.acknowledgeHandover(latest.id))
      });
    }
  }

  return (
    <section>
      <Topbar title={t("nav.handover")} principal={principal} />
      {handovers.error && (
        <div className="toast-error" role="alert">{handovers.error}</div>
      )}
      {handovers.loading && !handovers.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <>
          {transitions.length > 0 && (
            <div style={{ display: "flex", gap: 10, marginBottom: 14 }}>
              {transitions.map((transition) => (
                <button
                  key={transition.label}
                  className="primary-button"
                  disabled={busy}
                  onClick={transition.run}
                >
                  {transition.label}
                </button>
              ))}
            </div>
          )}
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
        </>
      )}
    </section>
  );
}

