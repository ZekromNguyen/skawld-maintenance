import { useI18n } from "../../i18n/I18nProvider";
import { relativeTime } from "../../presentation";
import type { Demonstration } from "../../types";


export function DemonstrationPanel(props: {
  values: Demonstration[];
  selected?: Demonstration;
  busy: boolean;
  onSelect: (value: Demonstration) => Promise<void>;
  onComplete: (value: Demonstration, outcome: string) => Promise<void>;
  onRedact: (
    value: Demonstration,
    eventID: string,
    path: string,
    reason: string
  ) => Promise<void>;
  onReview: (
    value: Demonstration,
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED",
    reason: string
  ) => Promise<void>;
}) {
  const { t, locale } = useI18n();
  const selected = props.selected;

  function complete() {
    if (!selected) return;
    const outcome = window.prompt(
      t("demo.outcomePrompt"),
      t("demo.outcomeDefault")
    );
    if (outcome?.trim()) void props.onComplete(selected, outcome.trim());
  }

  function review(
    decision: "APPROVED" | "REJECTED" | "REDACTION_REQUIRED"
  ) {
    if (!selected) return;
    const reason = window.prompt(
      t("demo.reviewReasonPrompt"),
      decision === "APPROVED"
        ? t("demo.reviewApproveDefault")
        : t("demo.reviewRejectDefault")
    );
    if (reason?.trim()) void props.onReview(selected, decision, reason.trim());
  }

  function redact(eventID: string) {
    if (!selected) return;
    const path = window.prompt(
      t("demo.jsonPathPrompt"),
      t("demo.jsonPathDefault")
    );
    if (!path?.trim()) return;
    const reason = window.prompt(
      t("demo.redactReasonPrompt"),
      t("demo.redactReasonDefault")
    );
    if (reason?.trim()) {
      void props.onRedact(selected, eventID, path.trim(), reason.trim());
    }
  }

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
            onClick={() => void props.onSelect(value)}
          >
            <span>
              <strong>{value.workflow_key}</strong>
              <small>
                {value.subject_kind} · {t("demo.semanticEvents", { count: value.events.length })}
              </small>
            </span>
            <span className="state-badge">{value.status}</span>
            <small>
              {t("demo.review", { status: value.review_status })} · {relativeTime(value.started_at, locale)}
            </small>
          </button>
        ))}
        {props.values.length === 0 && (
          <div className="empty">
            {t("demo.startCapture")}
          </div>
        )}
      </section>

      <section className="panel demonstration-timeline">
        {!selected ? (
          <div className="empty-state">
            <strong>{t("demo.select")}</strong>
            <p>
              {t("demo.reviewGuidance")}
            </p>
          </div>
        ) : (
          <>
            <div className="panel-heading">
              <div>
                <span className="eyebrow">SDK Observation v{selected.schema_version}</span>
                <h2>{selected.workflow_key}</h2>
              </div>
              <div className="heading-actions">
                <span className="state-badge">{selected.review_status}</span>
                {selected.status === "recording" && (
                  <button
                    className="primary-button"
                    disabled={props.busy}
                    onClick={complete}
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
                      <span className="source-badge">
                        {event.trust.replaceAll("_", " ")}
                      </span>
                    </div>
                    {event.intent && <p>{event.intent}</p>}
                    <small>
                      {event.entity?.type ?? t("demo.eventFallback")} ·{" "}
                      <span className="mono">{event.entity?.id ?? event.id}</span>
                    </small>
                    <small>
                      {t("demo.actor")} <span className="mono">{event.actor_id}</span> ·{" "}
                      {new Date(event.timestamp).toLocaleString()}
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
                    </details>
                    <div className="event-provenance">
                      <span>{event.source}</span>
                      <span>{event.sensitivity}</span>
                      <span className="mono">
                        {event.domain_event_id
                          ? t("demo.domain", { id: event.domain_event_id })
                          : t("demo.manualCapture")}
                      </span>
                      <button
                        className="secondary-button"
                        disabled={props.busy}
                        onClick={() => redact(event.id)}
                      >
                        {t("demo.redactField")}
                      </button>
                    </div>
                  </div>
                </article>
              ))}
            </div>
            {selected.status === "completed" && (
              <div className="review-actions">
                <span>
                  <strong>{t("demo.humanGovernance")}</strong>
                  <small>
                    {t("demo.governanceNote")}
                  </small>
                </span>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => review("REDACTION_REQUIRED")}
                >
                  {t("demo.requestRedaction")}
                </button>
                <button
                  className="secondary-button"
                  disabled={props.busy}
                  onClick={() => review("REJECTED")}
                >
                  {t("demo.reject")}
                </button>
                <button
                  className="primary-button"
                  disabled={props.busy}
                  onClick={() => review("APPROVED")}
                >
                  {t("demo.approveTrace")}
                </button>
              </div>
            )}
          </>
        )}
      </section>
    </div>
  );
}
