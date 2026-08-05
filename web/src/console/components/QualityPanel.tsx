import { useI18n } from "../../i18n/I18nProvider";
import type { EvaluationSummary } from "../../types";
import { Metric } from "./Metric";


export function QualityPanel(props: {
  value?: EvaluationSummary;
  busy: boolean;
  onRefresh: () => Promise<void>;
}) {
  const { t } = useI18n();
  if (!props.value) {
    return <section className="panel empty-state"><strong>{t("quality.noSummary")}</strong></section>;
  }
  const percent = (value: number) => `${Math.round(value * 100)}%`;
  return (
    <>
      <section className="metrics" aria-label={t("quality.pilotEvaluation")}>
        <Metric label={t("quality.reviewCoverage")} value={Math.round(props.value.review_coverage * 100)} detail={t("quality.reviewedDetail", { reviewed: props.value.reviewed, total: props.value.recommendations })} />
        <Metric label={t("quality.evidenceCoverage")} value={Math.round(props.value.evidence_coverage * 100)} detail={t("quality.labeledSupported")} />
        <Metric label={t("quality.unsafeRate")} value={Math.round(props.value.unsafe_recommendation_rate * 100)} detail={t("quality.mustRemainZero")} accent={props.value.unsafe_recommendation_rate > 0} safe={props.value.unsafe_recommendation_rate === 0} />
        <Metric label={t("quality.workflowGates")} value={Math.round(props.value.workflow_gate_pass_rate * 100)} detail={t("quality.evaluatedVersions", { count: props.value.workflow_evaluations })} />
      </section>
      <section className="panel quality-summary">
        <div className="panel-heading">
          <div><span className="eyebrow">{t("quality.pilotEvaluation")}</span><h2>{t("quality.humanReviewedSignals")}</h2></div>
          <button className="secondary-button" disabled={props.busy} onClick={() => void props.onRefresh()}>{t("quality.refresh")}</button>
        </div>
        <dl>
          <div><dt>{t("quality.acceptance")}</dt><dd>{percent(props.value.recommendation_acceptance)}</dd></div>
          <div><dt>{t("quality.humanOverride")}</dt><dd>{percent(props.value.human_override_rate)}</dd></div>
          <div><dt>{t("quality.unsupported")}</dt><dd>{percent(props.value.unsupported_recommendation_rate)}</dd></div>
          <div><dt>{t("quality.incorrectNextStep")}</dt><dd>{percent(props.value.incorrect_next_step_rate)}</dd></div>
          <div><dt>{t("quality.retrievalPrecision")}</dt><dd>{percent(props.value.retrieval_precision)}</dd></div>
          <div><dt>{t("quality.llmCalls")}</dt><dd>{props.value.llm_calls}</dd></div>
          <div><dt>{t("quality.avgLatency")}</dt><dd>{Math.round(props.value.average_latency_ms)} ms</dd></div>
          <div><dt>{t("quality.tokens")}</dt><dd>{props.value.tokens_in + props.value.tokens_out}</dd></div>
          <div><dt>{t("quality.estimatedCost")}</dt><dd>{props.value.estimated_cost_micros} μ</dd></div>
        </dl>
        <p className="muted">{t("quality.muted")}</p>
      </section>
    </>
  );
}
