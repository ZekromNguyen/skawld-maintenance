import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { useQuery } from "../useQuery";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { Overview } from "../components/Overview";
import { focusRole } from "../permissions";
import { FOCUS_CONFIG } from "../dashboardFocus";

/**
 * DashboardPage: role-aware landing. Metric cards come from the /summary
 * endpoint (truthful counts regardless of page size); "my queue" and the
 * active-incident table use capped list queries.
 */
export function DashboardPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids?.[0];
  const summary = useQuery(() => api.summary(siteID), [siteID]);
  const incidents = useQuery(() =>
    api.incidents({ site_id: siteID, page_size: 25 }).then((list) => list.items),
    [siteID],
  );
  const executions = useQuery(() =>
    api.listExecutions({ site_id: siteID, page_size: 25 }).then((list) => list.items),
    [siteID],
  );
  const handovers = useQuery(() => api.pendingHandovers(), []);

  const loading =
    summary.loading || incidents.loading || executions.loading || handovers.loading;
  const error =
    summary.error ?? incidents.error ?? executions.error ?? handovers.error;
  const onRetry = () => {
    void summary.refetch();
    void incidents.refetch();
    void executions.refetch();
    void handovers.refetch();
  };

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.overview")} principal={principal} />
        <Overview
          focus={FOCUS_CONFIG[focusRole(principal)]}
          summary={summary.data}
          incidents={incidents.data ?? []}
          executions={executions.data ?? []}
          pendingHandoverCount={handovers.data?.length ?? 0}
          loading={loading}
          error={error}
          onRetry={onRetry}
        />
      </section>
    </PageTrailProvider>
  );
}
