import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { useQuery } from "../useQuery";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { Overview } from "../components/Overview";
import { ForYouTabs } from "../components/ForYouTabs";
import { ShortcutCard } from "../components/ShortcutCard";
import { focusRole } from "../permissions";
import { FOCUS_CONFIG } from "../dashboardFocus";

/**
 * DashboardPage: role-aware landing. The annunciator strip and role focus
 * panel come from Overview; the Jira-style tabbed "For you" queue below
 * them aggregates the same live data into status-driven queues.
 */
export function DashboardPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const incidents = useQuery(() => api.incidents().then((list) => list.items));
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const executions = useQuery(() => api.listExecutions().then((list) => list.items));
  const handovers = useQuery(() => api.pendingHandovers());

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
  const pendingHandovers = handovers.data?.length ?? 0;
  const focus = FOCUS_CONFIG[focusRole(principal)];

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.overview")} principal={principal} />
        <Overview
          focus={focus}
          incidents={incidents.data ?? []}
          executions={executions.data ?? []}
          assets={assets.data ?? []}
          pendingHandoverCount={pendingHandovers}
          loading={loading}
          error={error}
          onRetry={onRetry}
        />
        <ForYouTabs
          incidents={incidents.data ?? []}
          executions={executions.data ?? []}
          assets={assets.data ?? []}
          pendingHandoverCount={pendingHandovers}
          loading={loading}
          error={error}
          onRetry={onRetry}
        />
        <ShortcutCard
          label={t("shortcuts.title")}
          detail=""
          to="/"
          variant="shortcuts"
        />
      </section>
    </PageTrailProvider>
  );
}
