import { useI18n } from "../../i18n/I18nProvider";
import type { ShiftHandover } from "../../types";
import { EvidenceLinks } from "./EvidenceLinks";

export function HandoverPanel(props: {
  siteID?: string;
  handover?: ShiftHandover;
  busy: boolean;
  onPrepare: () => Promise<void>;
  onCapture: () => Promise<void>;
}) {
  const { t } = useI18n();
  return (
    <section className="panel handover-panel">
      <div className="panel-heading">
        <div><span className="eyebrow">{t("handover.normalWorkIntelligence")}</span><h2>{t("handover.draft")}</h2></div>
        <div className="heading-actions">
          {props.handover && (
            <button className="secondary-button" disabled={props.busy} onClick={() => void props.onCapture()}>
              {t("handover.startDemonstration")}
            </button>
          )}
          <button className="primary-button" disabled={!props.siteID || props.busy} onClick={() => void props.onPrepare()}>{t("handover.prepare")}</button>
        </div>
      </div>
      {!props.handover ? (
        <div className="empty">{t("handover.empty")}</div>
      ) : (
        <div className="handover-content">
          <div className="result-header"><p>{props.handover.structured_content.summary}</p><span className="state-badge">{props.handover.state}</span></div>
          <HandoverSection title={t("handover.openIncidents")} items={props.handover.structured_content.open_incidents} empty={t("handover.none")} />
          <HandoverSection title={t("handover.activeExecutions")} items={props.handover.structured_content.active_executions} empty={t("handover.none")} />
          <HandoverSection title={t("handover.safetyConcerns")} items={props.handover.structured_content.safety_concerns} empty={t("handover.none")} />
          <HandoverSection title={t("handover.followUp")} items={props.handover.structured_content.follow_up} empty={t("handover.none")} />
          <EvidenceLinks evidence={props.handover.evidence} selected={props.handover.structured_content.evidence_ids} />
          <small className="provenance">{props.handover.provider}/{props.handover.model} · {props.handover.prompt_version} · {t("handover.humanAcceptance")}</small>
        </div>
      )}
    </section>
  );
}

export function HandoverSection({ title, items, empty }: { title: string; items: string[]; empty: string }) {
  return <div className="handover-section"><strong>{title}</strong>{items.length ? <ul>{items.map((item, index) => <li key={`${title}-${index}`}>{item}</li>)}</ul> : <p>{empty}</p>}</div>;
}
