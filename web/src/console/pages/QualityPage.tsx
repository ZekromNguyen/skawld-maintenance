import { api } from "../../api";
import { useQuery } from "../useQuery";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { QualityPanel } from "../components/QualityPanel";

/**
 * QualityPage: AI quality and safety evaluation summary with real refresh.
 */
export function QualityPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const quality = useQuery(() => api.evaluationSummary());

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader
          title={t("nav.quality")}
          principal={principal}
          actions={
            <button
              className="secondary-button"
              disabled={quality.loading}
              onClick={() => void quality.refetch()}
            >
              {t("quality.refresh")}
            </button>
          }
        />
        {quality.error ? (
          <ErrorState message={quality.error} onRetry={() => void quality.refetch()} />
        ) : quality.loading && !quality.data ? (
          <div style={{ display: "grid", gap: 12 }}>
            <Skeleton height={108} />
            <Skeleton height={220} />
          </div>
        ) : (
          <QualityPanel value={quality.data} />
        )}
      </section>
    </PageTrailProvider>
  );
}
