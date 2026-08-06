import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useI18n } from "../../i18n/I18nProvider";
import { PromptDialog } from "../feedback/PromptDialog";
import { Dialog } from "../feedback/Dialog";
import { FormField } from "../ui/FormField";
import { StatusBadge } from "../ui/StatusBadge";
import type { Demonstration } from "../../types";

type PromptState =
  | { kind: "complete" }
  | { kind: "review"; decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED" }
  | { kind: "redact"; eventID: string }
  | null;

const REVIEWED: ReadonlyArray<string> = ["APPROVED", "REJECTED"];

export function DemonstrationPanel(props: {
  values: Demonstration[];
  selected?: Demonstration;
  busy: boolean;
  onSelect: (value: Demonstration) => void;
  onComplete: (value: Demonstration, outcome: string) => void;
  onRedact: (
    value: Demonstration,
    eventID: string,
    path: string,
    reason: string
  ) => void;
  onReview: (
    value: Demonstration,
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED",
    reason: string
  ) => void;
}) {
  const { t, locale } = useI18n();
  const selected = props.selected;
  const [prompt, setPrompt] = useState<PromptState>(null);

  const localeTag = locale === "vi" ? "vi-VN" : "en-US";
  const canReview =
    selected?.status === "completed" && !REVIEWED.includes(selected.review_status ?? "");

  return (
    <div className="demonstration-layout">
      <section className="panel demonstration-list">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">{t("demo.organizationalMemory")}</span>
            <h2>{t("demo.semanticDemonstrations")}</h2>
          </div>
          <span className="count">{props.values.length}</span>
        </div>
        {props.values.map((value) => (
          <button
            key={value.id}
            className={
              selected?.id === value.id
                ? "demonstration-item selected"
                : "demonstration-item"
            }
            aria-current={selected?.id === value.id ? "true" : undefined}
            onClick={() => props.onSelect(value)}
          >
            <span>
              <strong>{value.workflow_key}</strong>
              <small>
                {value.subject_kind} · {t("demo.semanticEvents", { count: value.events.length })}
              </small>
            </span>
            <StatusBadge tone={value.review_status === "APPROVED" ? "success" : value.review_status === "REJECTED" ? "critical" : "info"} label={value.review_status ?? value.status} />
            <small>
              {value.status} · {new Date(value.started_at).toLocaleString(localeTag)}
            </small>
          </button>
        ))}
        {props.values.length === 0 && (
          <div className="empty-state">
            <strong>{t("demo.noCaptures")}</strong>
            <p>{t("demo.startCaptureGuidance")}</p>
            <Link to="/handovers" className="secondary-button">{t("nav.handover")}</Link>
          </div>
        )}
      </section>

      <section className="panel demonstration-timeline">
        {!selected ? (
          <div className="empty-state">
            <strong>{t("demo.select")}</strong>
            <p>{t("demo.reviewGuidance")}</p>
          </div>
        ) : (
          <>
            <div className="panel-heading">
              <div>
                <span className="eyebrow">SDK Observation v{selected.schema_version}</span>
                <h2>{selected.workflow_key}</h2>
              </div>
              <div className="heading-actions">
                <StatusBadge tone={selected.review_status === "APPROVED" ? "success" : selected.review_status === "REJECTED" ? "critical" : "info"} label={selected.review_status ?? selected.status} />
                {selected.status === "recording" && (
                  <button
                    className="primary-button"
                    disabled={props.busy}
                    onClick={() => setPrompt({ kind: "complete" })}
                  >
                    {t("demo.completeCapture")}
                  </button>
                )}
              </div>
            </div>
            <div className="capture-health">
              <span>
                <strong>{selected.capture.applied}</strong> {t("demo.applied")}
              </span>
              <span>
                <strong>{selected.capture.pending}</strong> {t("demo.pending")}
              </span>
              <span className={selected.capture.failed ? "capture-failed" : ""}>
                <strong>{selected.capture.failed}</strong> {t("demo.failed")}
              </span>
              <small className="mono">{t("demo.session", { id: selected.session_id })}</small>
            </div>
            {selected.capture.last_error && (
              <div className="notice">{selected.capture.last_error}</div>
            )}
            <div className="semantic-timeline">
              {selected.events.map((event) => (
                <article
                  key={event.id}
                  className={
                    event.correction_of
                      ? "semantic-event correction-event"
                      : "semantic-event"
                  }
                >
                  <span className="timeline-ordinal">{event.ordinal}</span>
                  <div>
                    <div className="event-heading">
                      <strong>{event.action}</strong>
                      <span className="source-badge">{event.trust.replaceAll("_", " ")}</span>
                      {event.redactions && event.redactions.length > 0 ? (
                        <StatusBadge tone="critical" label={t("demo.redactedCount", { count: event.redactions.length })} />
                      ) : null}
                    </div>
                    {event.intent && <p>{event.intent}</p>}
                    <small>
                      {event.entity?.type ?? t("demo.eventFallback")} ·{" "}
                      <span className="mono">{event.entity?.id ?? event.id}</span>
                    </small>
                    <small>
                      {t("demo.actor")} <span className="mono">{event.actor_id}</span> ·{" "}
                      {new Date(event.timestamp).toLocaleString(localeTag)}
                    </small>
                    {event.correction_of && (
                      <span className="correction-link">
                        {t("demo.correctsEvent", { id: event.correction_of })}
                      </span>
                    )}
                    <details>
                      <summary>{t("demo.structuredPayload")}</summary>
                      <pre>
                        {JSON.stringify(
                          {
                            output: event.output,
                            decision: event.decision,
                            result: event.result,
                            context: event.context
                          },
                          null,
                          2
                        )}
                      </pre>
                      <CopyButton payload={JSON.stringify({
                        output: event.output,
                        decision: event.decision,
                        result: event.result,
                        context: event.context
                      })} label={t("demo.copyPayload")} copiedLabel={t("demo.copied")} />
                    </details>
                    <div className="event-provenance">
                      <span>{event.source}</span>
                      <span>{event.sensitivity}</span>
                      <span className="mono">
                        {event.domain_event_id
                          ? t("demo.domain", { id: event.domain_event_id })
                          : t("demo.manualCapture")}
                      </span>
                      {canReview ? (
                        <button
                          className="secondary-button"
                          disabled={props.busy}
                          onClick={() => setPrompt({ kind: "redact", eventID: event.id })}
                        >
                          {t("demo.redactField")}
                        </button>
                      ) : null}
                    </div>
                  </div>
                </article>
              ))}
            </div>
            {canReview && (
              <div className="review-actions">
                <span>
                  <strong>{t("demo.humanGovernance")}</strong>
                  <small>{t("demo.governanceNote")}</small>
                </span>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => setPrompt({ kind: "review", decision: "REDACTION_REQUIRED" })}
                >
                  {t("demo.requestRedaction")}
                </button>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => setPrompt({ kind: "review", decision: "REJECTED" })}
                >
                  {t("demo.reject")}
                </button>
                <button
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() => setPrompt({ kind: "review", decision: "APPROVED" })}
                >
                  {t("demo.approveTrace")}
                </button>
              </div>
            )}
          </>
        )}
      </section>

      {selected && (
        <>
          <PromptDialog
            open={prompt?.kind === "complete"}
            onOpenChange={(open) => {
              if (!open) setPrompt(null);
            }}
            title={t("demo.completeCapture")}
            label={t("demo.outcomeLabel")}
            defaultValue={t("demo.outcomeDefault")}
            confirmLabel={t("demo.completeCapture")}
            pending={props.busy}
            onConfirm={(outcome) => {
              props.onComplete(selected, outcome);
            }}
          />
          <PromptDialog
            open={prompt?.kind === "review"}
            onOpenChange={(open) => {
              if (!open) setPrompt(null);
            }}
            title={prompt?.kind === "review" && prompt.decision === "APPROVED" ? t("demo.approveTrace") : prompt?.kind === "review" && prompt.decision === "REJECTED" ? t("demo.reject") : t("demo.requestRedaction")}
            label={t("demo.reviewReasonLabel")}
            defaultValue={prompt?.kind === "review" && prompt.decision === "APPROVED" ? t("demo.reviewApproveDefault") : t("demo.reviewRejectDefault")}
            confirmLabel={prompt?.kind === "review" && prompt.decision === "APPROVED" ? t("demo.approveTrace") : prompt?.kind === "review" && prompt.decision === "REJECTED" ? t("demo.reject") : t("demo.requestRedaction")}
            pending={props.busy}
            onConfirm={(reason) => {
              if (prompt?.kind === "review") props.onReview(selected, prompt.decision, reason);
            }}
          />
          {prompt?.kind === "redact" && (
            <RedactDialog
              open
              onClose={() => setPrompt(null)}
              pending={props.busy}
              onRedact={(path, reason) => {
                if (prompt.kind === "redact") props.onRedact(selected, prompt.eventID, path, reason);
              }}
            />
          )}
        </>
      )}
    </div>
  );
}

function RedactDialog({
  open,
  onClose,
  pending,
  onRedact,
}: {
  open: boolean;
  onClose: () => void;
  pending: boolean;
  onRedact: (path: string, reason: string) => void;
}) {
  const { t } = useI18n();
  const [path, setPath] = useState(t("demo.jsonPathDefault"));
  const [reason, setReason] = useState(t("demo.redactReasonDefault"));

  useEffect(() => {
    if (open) {
      setPath(t("demo.jsonPathDefault"));
      setReason(t("demo.redactReasonDefault"));
    }
  }, [open, t]);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("demo.redactField")}
      footer={
        <>
          <button className="secondary-button" onClick={onClose} disabled={pending}>
            {t("demo.cancel")}
          </button>
          <button
            className="primary-button"
            disabled={pending || !path.trim() || !reason.trim()}
            onClick={() => onRedact(path.trim(), reason.trim())}
          >
            {t("demo.redactField")}
          </button>
        </>
      }
    >
      <div style={{ display: "grid", gap: 14 }}>
        <FormField label={t("demo.jsonPathLabel")} htmlFor="redact-path">
          <input
            id="redact-path"
            className="mono"
            value={path}
            onChange={(event) => setPath(event.target.value)}
          />
        </FormField>
        <FormField label={t("demo.redactReasonLabel")} htmlFor="redact-reason">
          <textarea
            id="redact-reason"
            rows={3}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
        </FormField>
      </div>
    </Dialog>
  );
}

function CopyButton({
  payload,
  label,
  copiedLabel,
}: {
  payload: string;
  label: string;
  copiedLabel: string;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      className="secondary-button"
      onClick={() => {
        void navigator.clipboard.writeText(payload).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        });
      }}
    >
      {copied ? copiedLabel : label}
    </button>
  );
}
