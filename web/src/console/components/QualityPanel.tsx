import { useI18n } from "../../i18n/I18nProvider";
import { MetricCard } from "../ui/MetricCard";
import { RelativeTime } from "../ui/RelativeTime";
import type { EvaluationSummary } from "../../types";

/**
 * QualityPanel: evaluation KPIs with honest formatting (no zero-padded
 * percents, readable currency/tokens) and a last-updated timestamp.
 */
export function QualityPanel({ value }: { value?: EvaluationSummary }) {
  const { t, locale } = useI18n();
  if (!value) {
    return <section className="panel empty-state"><strong>{t("quality.noSummary")}</strong></section>;
  }
  const percent = (v: number) => `${Math.round(v * 100)}%`;
  const number = (v: number) => new Intl.NumberFormat(locale === "vi" ? "vi-VN" : "en-US").format(v);
  const cost = (micros: number) => `$${(micros / 1_000_000).toFixed(2)}`;
  const unsafe = value.unsafe_recommendation_rate > 0;
  // Per-metric gating: a zero with no sample behind it is "no data", not a
  // real measurement. Recommendation-derived rates gate on the recommendation
  // count; usage/cost figures gate on LLM calls; workflow gates on evaluations.
  const hasRecommendations = value.recommendations > 0;
  const hasWorkflow = value.workflow_evaluations > 0;
  const hasCost = value.llm_calls > 0;
  const noDataText = t("quality.noData");
  const ratio = (v: number, present: boolean) => (present ? percent(v) : noDataText);
  const count = (v: number, present: boolean) => (present ? number(v) : noDataText);
  return (
    <>
      <section className="metrics" aria-label={t("quality.pilotEvaluation")}>
        <MetricCard label={t("quality.reviewCoverage")} value={hasRecommendations ? `${percent(value.review_coverage)} (${value.reviewed}/${value.recommendations})` : noDataText} detail={t("quality.reviewedDetail", { reviewed: value.reviewed, total: value.recommendations })} tone={value.review_coverage > 0.9 ? "success" : "medium"} />
        <MetricCard label={t("quality.evidenceCoverage")} value={ratio(value.evidence_coverage, hasRecommendations)} detail={t("quality.labeledSupported")} />
        <MetricCard label={t("quality.unsafeRate")} value={ratio(value.unsafe_recommendation_rate, hasRecommendations)} detail={t("quality.mustRemainZero")} tone={unsafe ? "critical" : "success"} />
        <MetricCard label={t("quality.workflowGates")} value={ratio(value.workflow_gate_pass_rate, hasWorkflow)} detail={t("quality.evaluatedVersions", { count: value.workflow_evaluations })} />
      </section>
      <section className="panel quality-summary">
        <div className="panel-heading">
          <div>
            <span className="eyebrow">{t("quality.pilotEvaluation")}</span>
            <h2>{t("quality.humanReviewedSignals")}</h2>
          </div>
          {value.generated_at ? (
            <small className="muted">
              {t("quality.lastUpdated")} <RelativeTime time={value.generated_at} locale={locale} />
            </small>
          ) : null}
        </div>
        <dl>
          <div><dt>{t("quality.acceptance")}</dt><dd>{ratio(value.recommendation_acceptance, hasRecommendations)}</dd></div>
          <div><dt>{t("quality.humanOverride")}</dt><dd>{ratio(value.human_override_rate, hasRecommendations)}</dd></div>
          <div><dt>{t("quality.unsupported")}</dt><dd>{ratio(value.unsupported_recommendation_rate, hasRecommendations)}</dd></div>
          <div><dt>{t("quality.incorrectNextStep")}</dt><dd>{ratio(value.incorrect_next_step_rate, hasRecommendations)}</dd></div>
          <div><dt>{t("quality.retrievalPrecision")}</dt><dd>{ratio(value.retrieval_precision, hasRecommendations)}</dd></div>
          <div><dt>{t("quality.llmCalls")}</dt><dd>{count(value.llm_calls, hasCost)}</dd></div>
          <div><dt>{t("quality.avgLatency")}</dt><dd>{hasCost ? `${number(value.average_latency_ms)} ms` : noDataText}</dd></div>
          <div><dt>{t("quality.tokens")}</dt><dd>{count(value.tokens_in + value.tokens_out, hasCost)}</dd></div>
          <div><dt>{t("quality.estimatedCost")}</dt><dd>{hasCost ? cost(value.estimated_cost_micros) : noDataText}</dd></div>
        </dl>
        <p className="muted">{t("quality.muted")}</p>
      </section>
    </>
  );
}
