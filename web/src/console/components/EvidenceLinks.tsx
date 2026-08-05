import { useState } from "react";
import { useI18n } from "../../i18n/I18nProvider";
import type { Evidence } from "../../types";

export function EvidenceLinks(props: {
  evidence: Evidence[];
  selected: string[];
  onView?: (evidenceID: string) => Promise<void>;
}) {
  const { evidence, selected } = props;
  const items = evidence.filter((item) => selected.includes(item.id));
  const [viewed, setViewed] = useState<Set<string>>(() => new Set());
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
