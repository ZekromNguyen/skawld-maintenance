import { useI18n } from "../../i18n/I18nProvider";
import type { HandoverListItem, ShiftHandover } from "../../types";
import { EvidenceLinks } from "./EvidenceLinks";
import { StatusBadge } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";
import { handoverStateTone } from "../labels";

const STATE_LABEL: Record<string, "handover.state.draft" | "handover.state.submitted" | "handover.state.accepted" | "handover.state.acknowledged"> = {
  DRAFT: "handover.state.draft",
  SUBMITTED: "handover.state.submitted",
  ACCEPTED: "handover.state.accepted",
  ACKNOWLEDGED: "handover.state.acknowledged"
};

export function HandoverPanel(props: {
  handover?: ShiftHandover;
  pending: boolean;
  canPrepare: boolean;
  canCapture: boolean;
  onPrepare: () => void;
  onCapture: () => void;
}) {
  const { t, locale } = useI18n();
  const handover = props.handover;
  const titleKey = handover ? STATE_LABEL[handover.state] ?? "handover.state.draft" : "handover.state.draft";
  const requiresReview = handover?.structured_content.requires_human_review === true;
  const unknowns = handover?.structured_content.unknowns ?? [];

  return (
    <section className="panel handover-panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("handover.normalWorkIntelligence")}</span>
          <h2>{t(titleKey)}</h2>
        </div>
        <div className="heading-actions">
          {handover && (
            <button
              className="secondary-button"
              disabled={!props.canCapture || props.pending}
              title={!props.canCapture ? t("handover.humanAcceptance") : undefined}
              onClick={props.onCapture}
            >
              {t("handover.startDemonstration")}
            </button>
          )}
          <button
            className="primary-button"
            disabled={!props.canPrepare || props.pending}
            onClick={props.onPrepare}
          >
            {t("handover.prepare")}
          </button>
        </div>
      </div>
      {!handover ? (
        <div className="empty">{t("handover.empty")}</div>
      ) : (
        <div className="handover-content">
          {requiresReview ? (
            <div className="notice" role="alert">{t("handover.requiresHumanReview")}</div>
          ) : null}
          <div className="result-header">
            <p>{handover.structured_content.summary}</p>
            <StatusBadge tone={handoverStateTone(handover.state)} label={t(titleKey)} />
          </div>
          <div className="handover-meta">
            <span className="eyebrow">{t("handover.shiftWindow")}</span>
            <span className="mono">
              <RelativeTime time={handover.shift_start} locale={locale} /> → <RelativeTime time={handover.shift_end} locale={locale} />
            </span>
          </div>
          <HandoverSection title={t("handover.openIncidents")} items={handover.structured_content.open_incidents} empty={t("handover.none")} />
          <HandoverSection title={t("handover.activeExecutions")} items={handover.structured_content.active_executions} empty={t("handover.none")} />
          <HandoverSection title={t("handover.safetyConcerns")} items={handover.structured_content.safety_concerns} empty={t("handover.none")} />
          <HandoverSection title={t("handover.followUp")} items={handover.structured_content.follow_up} empty={t("handover.none")} />
          {unknowns.length > 0 ? (
            <HandoverSection title={t("handover.unknowns")} items={unknowns} empty={t("handover.none")} />
          ) : null}
          <EvidenceLinks evidence={handover.evidence} selected={handover.structured_content.evidence_ids} />
          <small className="provenance">
            {handover.provider}/{handover.model} · {handover.prompt_version} · {t("handover.humanAcceptance")}
          </small>
        </div>
      )}
    </section>
  );
}

export function HandoverSection({ title, items, empty }: { title: string; items: HandoverListItem[]; empty: string }) {
  return (
    <div className="handover-section">
      <h3>{title}</h3>
      {items.length ? (
        <ul>
          {items.map((item, index) => (
            <li key={`${title}-${item.title}-${index}`}>
              {item.title}
              {item.detail ? <span className="handover-detail"> · {item.detail}</span> : null}
            </li>
          ))}
        </ul>
      ) : (
        <p>{empty}</p>
      )}
    </div>
  );
}
