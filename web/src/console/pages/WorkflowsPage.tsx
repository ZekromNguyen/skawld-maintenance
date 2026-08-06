import { useState } from "react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useSite } from "../state/SiteContext";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { WorkflowLearningPanel } from "../components/WorkflowLearningPanel";
import type { WorkflowApplicability, WorkflowVersion } from "../../types";

/**
 * WorkflowsPage: learned workflow compilation, review, publish, retire.
 * Site-scoped demos, toast feedback, names resolved from the workflow data.
 */
export function WorkflowsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const { siteId } = useSite();
  const workflows = useQuery(() => api.workflows().then((list) => list.items));
  const demonstrations = useQuery(
    () => api.demonstrations(siteId).then((list) => list.items),
    [siteId],
  );
  const assets = useQuery(() => api.assets().then((list) => list.items));
  const [selected, setSelected] = useState<WorkflowVersion | undefined>(undefined);

  const compile = useCommand(
    (name: string, ids: string[]) => api.compileWorkflow(name, ids),
    {
      successMessage: t("workflow.compileSuccess"),
      onSuccess: (value) => setSelected(value),
    },
  );
  const review = useCommand(
    (value: WorkflowVersion, decision: "APPROVED" | "REJECTED" | "REVIEW_REQUIRED", reason: string) =>
      api.reviewWorkflow(value, decision, reason, applicabilityFor(value)),
    {
      successMessage: t("workflow.reviewSuccess"),
      onSuccess: (value) => setSelected(value),
    },
  );
  const publish = useCommand(
    (value: WorkflowVersion, reason: string) => api.publishWorkflow(value, reason),
    {
      successMessage: t("workflow.publishSuccess"),
      onSuccess: (value) => setSelected(value),
    },
  );
  const retire = useCommand(
    (value: WorkflowVersion, reason: string) => api.retireWorkflow(value, reason),
    {
      successMessage: t("workflow.retireSuccess"),
      onSuccess: (value) => setSelected(value),
    },
  );

  function applicabilityFor(value: WorkflowVersion): WorkflowApplicability[] {
    const asset = (assets.data ?? []).find(
      (item) => item.site_id === value.site_id && item.class === value.asset_class
    );
    return [
      {
        site_id: value.site_id,
        asset_id: asset?.id,
        asset_class: value.asset_class,
        manufacturer: asset?.manufacturer,
        model: asset?.model,
        validation_status: "VALIDATED"
      }
    ];
  }

  const busy = compile.pending || review.pending || publish.pending || retire.pending;
  const perms = principal?.permissions ?? [];
  const canReview = perms.includes("workflow:review");
  const canPublish = perms.includes("workflow:publish");

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.workflows")} principal={principal} />
        <WorkflowLearningPanel
          values={workflows.data ?? []}
          demonstrations={demonstrations.data ?? []}
          selected={selected}
          busy={busy}
          canReview={canReview}
          canPublish={canPublish}
          onSelect={(value) => setSelected(value)}
          onCompile={(ids) => {
            const source = demonstrations.data?.find((demo) => demo.id === ids[0]);
            void compile.run(source?.workflow_key ?? "compiled workflow", ids);
          }}
          onReview={(value, decision, reason) => void review.run(value, decision, reason)}
          onPublish={(value, reason) => void publish.run(value, reason)}
          onRetire={(value, reason) => void retire.run(value, reason)}
        />
      </section>
    </PageTrailProvider>
  );
}
