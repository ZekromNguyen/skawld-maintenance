import { useState } from "react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useSite } from "../state/SiteContext";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { DemonstrationPanel } from "../components/DemonstrationPanel";

/**
 * DemonstrationsPage: expert demonstration capture and review. Site-scoped
 * fetch (no mount race), dialog-based inputs, toast feedback on actions.
 */
export function DemonstrationsPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const { siteId } = useSite();
  const demonstrations = useQuery(
    () => api.demonstrations(siteId).then((list) => list.items),
    [siteId],
  );
  const [selected, setSelected] = useState<string | undefined>(undefined);

  const complete = useCommand(
    (id: string, outcome: string) => api.completeDemonstration(id, outcome),
    { successMessage: t("demo.completeSuccess"), onSuccess: () => void demonstrations.refetch() },
  );
  const redact = useCommand(
    (id: string, eventID: string, path: string, reason: string) =>
      api.redactDemonstrationEvent(id, eventID, path, reason),
    { successMessage: t("demo.redactSuccess"), onSuccess: () => void demonstrations.refetch() },
  );
  const review = useCommand(
    (id: string, decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED", reason: string) =>
      api.reviewDemonstration(id, decision, reason),
    { successMessage: t("demo.reviewSuccess"), onSuccess: () => void demonstrations.refetch() },
  );

  const values = demonstrations.data ?? [];
  const selectedValue = values.find((value) => value.id === selected);
  const busy = complete.pending || redact.pending || review.pending;

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.demonstrations")} principal={principal} />
        <DemonstrationPanel
          values={values}
          selected={selectedValue}
          busy={busy}
          onSelect={(value) => setSelected(value.id)}
          onComplete={(value, outcome) => void complete.run(value.id, outcome)}
          onRedact={(value, eventID, path, reason) => void redact.run(value.id, eventID, path, reason)}
          onReview={(value, decision, reason) => void review.run(value.id, decision, reason)}
        />
      </section>
    </PageTrailProvider>
  );
}
