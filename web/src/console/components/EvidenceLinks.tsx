import { useState } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import { EmptyState } from "../ui/EmptyState";
import type { Evidence } from "../../types";

/** EvidenceLinks: collapsible evidence entries with an explicit empty state. */
export function EvidenceLinks(props: {
  evidence: Evidence[];
  selected: string[];
  emptyTitle?: string;
  onView?: (evidenceID: string) => Promise<void>;
}) {
  const { t } = useI18n();
  const { evidence, selected } = props;
  const items = evidence.filter((item) => selected.includes(item.id));
  const [viewed, setViewed] = useState<Set<string>>(() => new Set());
  if (items.length === 0) {
    return <EmptyState title={props.emptyTitle ?? t("report.noEvidence")} />;
  }
  return (
    <div className="evidence-list">
      {items.map((item) => (
        <details
          key={item.id}
          onToggle={(event) => {
            if (
              event.currentTarget.open &&
              props.onView &&
              !viewed.has(item.id)
            ) {
              const next = new Set(viewed);
              next.add(item.id);
              setViewed(next);
              void props.onView(item.id);
            }
          }}
        >
          <summary>{item.title} · {item.locator} · {item.authority}</summary>
          <p>{item.content}</p>
          <small className="mono">{item.id}</small>
        </details>
      ))}
    </div>
  );
}
