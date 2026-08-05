import { useState } from "react";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import type { WorkflowVersion } from "../../types";
import { Topbar } from "../layout/Topbar";
import { WorkflowLearningPanel } from "../components/WorkflowLearningPanel";

/**
 * WorkflowsPage: learned workflow compilation, review, publish, retire.
 */
export function WorkflowsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0];
  const workflows = useApi(() => api.workflows());
  const demonstrations = useApi(() => api.demonstrations(siteID));
  const assets = useApi(() => api.assets());
  const [selected, setSelected] = useState<WorkflowVersion | undefined>(undefined);
  const [busy, setBusy] = useState(false);

  const mutate = async (action: () => Promise<void>) => {
    setBusy(true);
    try {
      await action();
      await workflows.refetch();
    } finally {
      setBusy(false);
    }
  };

  const values = workflows.data?.items ?? [];
  const demos = demonstrations.data?.items ?? [];

  return (
    <section>
      <Topbar title={t("nav.workflows")} principal={principal} />
      {workflows.error && (
        <div className="toast-error" role="alert">{workflows.error}</div>
      )}
      {workflows.loading && !workflows.data ? (
        <div className="skeleton" style={{ height: 300 }} />
      ) : (
        <WorkflowLearningPanel
          values={values}
          demonstrations={demos}
          selected={selected}
          busy={busy}
          onSelect={(value) =>
            mutate(async () => {
              const detail = await api.workflow(value.workflow_id, value.version);
              setSelected(detail);
            })
          }
          onCompile={(demonstrationIDs) =>
            mutate(async () => {
              const source = demos.find((demonstration) =>
                demonstrationIDs.includes(demonstration.id),
              );
              const name =
                source?.workflow_key === "maintenance.shift_handover"
                  ? "Shift handover workflow"
                  : "High vibration pump inspection";
              const value = await api.compileWorkflow(name, demonstrationIDs);
              setSelected(value);
            })
          }
          onReview={(value, decision, reason) =>
            mutate(async () => {
              const asset = (assets.data?.items ?? []).find(
                (item) =>
                  item.site_id === value.site_id &&
                  item.class === value.asset_class,
              );
              const applicability = [
                {
                  site_id: value.site_id,
                  asset_id: asset?.id,
                  asset_class: value.asset_class,
                  manufacturer: asset?.manufacturer,
                  model: asset?.model,
                  validation_status: "VALIDATED" as const
                }
              ];
              const detail = await api.reviewWorkflow(
                value,
                decision,
                reason,
                applicability,
              );
              setSelected(detail);
            })
          }
          onPublish={(value, reason) =>
            mutate(async () => {
              const detail = await api.publishWorkflow(value, reason);
              setSelected(detail);
            })
          }
          onRetire={(value, reason) =>
            mutate(async () => {
              const detail = await api.retireWorkflow(value, reason);
              setSelected(detail);
            })
          }
        />
      )}
    </section>
  );
}
