import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useSite } from "../state/SiteContext";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { HandoverPanel } from "../components/HandoverPanel";
import type { ShiftHandover } from "../../types";

/**
 * HandoverPage: shift handover capture, transitions, and review. The latest
 * handover per site scope, sorted by shift start; prior handovers as history.
 */
export function HandoverPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const { siteId } = useSite();
  const firstPage = useQuery(
    () =>
      api.handovers(
        siteId ? { site_id: siteId, page_size: 25 } : { page_size: 25 },
      ),
    [siteId],
  );
  const [history, setHistory] = useState<ShiftHandover[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);

  useEffect(() => {
    setHistory([]);
    setNextCursor(firstPage.data?.next_cursor ?? null);
  }, [firstPage.data, siteId]);

  const items = useMemo(
    () => [...(firstPage.data?.items ?? []), ...history],
    [firstPage.data, history],
  );
  const latest = items[0];
  const historyList = items.slice(1);
  const loadMore = async () => {
    if (!nextCursor) return;
    const page = await api.handovers(
      siteId
        ? { site_id: siteId, cursor: nextCursor, page_size: 25 }
        : { cursor: nextCursor, page_size: 25 },
    );
    setHistory((prev) => [...prev, ...page.items]);
    setNextCursor(page.next_cursor);
  };
  const perms = principal?.permissions ?? [];

  const submit = useCommand(
    (id: string) => api.submitHandover(id),
    { successMessage: t("handover.submitSuccess"), onSuccess: () => void firstPage.refetch() },
  );
  const accept = useCommand(
    (id: string) => api.acceptHandover(id),
    { successMessage: t("handover.acceptSuccess"), onSuccess: () => void firstPage.refetch() },
  );
  const acknowledge = useCommand(
    (id: string) => api.acknowledgeHandover(id),
    { successMessage: t("handover.acknowledgeSuccess"), onSuccess: () => void firstPage.refetch() },
  );
  const prepare = useCommand(
    (site: string) => api.prepareHandover(site),
    { successMessage: t("handover.prepareSuccess"), onSuccess: () => void firstPage.refetch() },
  );
  const capture = useCommand(
    (id: string) => api.startDemonstration("HANDOVER", id),
    { successMessage: t("handover.captureStarted"), onSuccess: () => navigate("/demonstrations") },
  );

  const transitions: Array<{ key: string; label: string; pending: boolean; run: () => void }> = [];
  if (latest) {
    if (latest.state === "DRAFT" && perms.includes("handover:write")) {
      transitions.push({ key: "submit", label: t("handover.submit"), pending: submit.pending, run: () => void submit.run(latest.id) });
    }
    if (latest.state === "SUBMITTED" && perms.includes("handover:accept")) {
      transitions.push({ key: "accept", label: t("handover.accept"), pending: accept.pending, run: () => void accept.run(latest.id) });
    }
    if (latest.state === "ACCEPTED" && perms.includes("handover:accept")) {
      transitions.push({ key: "acknowledge", label: t("handover.acknowledge"), pending: acknowledge.pending, run: () => void acknowledge.run(latest.id) });
    }
  }

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.handover")} principal={principal} />
        {firstPage.error ? (
          <div className="error-state" role="alert" style={{ marginBottom: 16 }}>
            <strong>Something went wrong</strong>
            <p>{firstPage.error}</p>
            <div className="error-actions">
              <button className="secondary-button" onClick={() => void firstPage.refetch()}>Retry</button>
            </div>
          </div>
        ) : null}
        {firstPage.loading && !firstPage.data ? (
          <div className="skeleton" style={{ height: 300 }} />
        ) : (
          <>
            <div style={{ display: "flex", gap: 10, marginBottom: 14, flexWrap: "wrap" }}>
              {transitions.map((transition) => (
                <button key={transition.key} className="primary-button" disabled={transition.pending} onClick={transition.run}>
                  {transition.label}
                </button>
              ))}
            </div>
            <HandoverPanel
              handover={latest}
              pending={capture.pending}
              canPrepare={perms.includes("handover:write") && Boolean(siteId)}
              canCapture={Boolean(latest) && perms.includes("handover:write")}
              onPrepare={() => {
                if (siteId) void prepare.run(siteId);
              }}
              onCapture={() => {
                if (latest) void capture.run(latest.id);
              }}
            />
            <section className="panel" style={{ marginTop: 16 }}>
              <div className="panel-heading"><h2>{t("handover.history")}</h2></div>
              {historyList.length > 0 ? (
                <div>
                  {historyList.map((handover) => (
                    <div key={handover.id} className="handover-history-row">
                      <span className="mono">{handover.shift_start.slice(0, 10)}</span>
                      <span className="state-badge">{handover.state.replace("_", " ")}</span>
                      <span className="muted">{handover.structured_content.summary.slice(0, 80)}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="empty">{t("handover.noHistory")}</div>
              )}
            </section>
            {nextCursor ? (
              <button
                type="button"
                className="secondary-button"
                onClick={() => void loadMore()}
                style={{ marginTop: 12 }}
              >
                {t("common.loadMore")}
              </button>
            ) : null}
          </>
        )}
      </section>
    </PageTrailProvider>
  );
}
