import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { useQuery } from "../useQuery";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { Overview } from "../components/Overview";

/**
 * DashboardPage: role-aware landing. Real KPIs, a "my queue" of in-progress
 * executions, and the active-incident table. No fabricated metrics.
 */
export function DashboardPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const incidents = useQuery(() => api.incidents().then((list) => list.items));
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const executions = useQuery(() => api.listExecutions().then((list) => list.items));
  const handovers = useQuery(() => api.handovers());

  const loading =
    incidents.loading || assets.loading || executions.loading || handovers.loading;
  const error =
    incidents.error ?? assets.error ?? executions.error ?? handovers.error;
  const onRetry = () => {
    void incidents.refetch();
    void assets.refetch();
    void executions.refetch();
    void handovers.refetch();
  };
  const pendingHandovers =
    handovers.data?.items.filter((handover) => handover.state !== "ACKNOWLEDGED").length ?? 0;

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.overview")} principal={principal} />
        <Overview
          incidents={incidents.data ?? []}
          executions={executions.data ?? []}
          assets={assets.data ?? []}
          pendingHandoverCount={pendingHandovers}
          loading={loading}
          error={error}
          onRetry={onRetry}
        />
      </section>
    </PageTrailProvider>
  );
}
