import { useState } from "react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";
import { QualityPanel } from "../components/QualityPanel";

/**
 * QualityPage: AI quality and safety evaluation summary.
 */
export function QualityPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const quality = useApi(() => api.evaluationSummary());
  const [busy, setBusy] = useState(false);

  const refresh = async () => {
    setBusy(true);
    try {
      await quality.refetch();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section>
      <Topbar title={t("nav.quality")} principal={principal} />
      {quality.error && (
        <div className="toast-error" role="alert">{quality.error}</div>
      )}
      {quality.loading && !quality.data ? (
        <div style={{ display: "grid", gap: 12 }}>
          <div className="skeleton" style={{ height: 108 }} />
          <div className="skeleton" style={{ height: 220 }} />
        </div>
      ) : (
        <QualityPanel value={quality.data} busy={busy} onRefresh={refresh} />
      )}
    </section>
  );
}
