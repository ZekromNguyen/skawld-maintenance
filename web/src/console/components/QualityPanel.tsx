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
  return (
    <>
      <section className="metrics" aria-label={t("quality.pilotEvaluation")}>
        <MetricCard label={t("quality.reviewCoverage")} value={percent(value.review_coverage)} detail={t("quality.reviewedDetail", { reviewed: value.reviewed, total: value.recommendations })} tone={value.review_coverage > 0.9 ? "success" : "medium"} />
        <MetricCard label={t("quality.evidenceCoverage")} value={percent(value.evidence_coverage)} detail={t("quality.labeledSupported")} />
        <MetricCard label={t("quality.unsafeRate")} value={percent(value.unsafe_recommendation_rate)} detail={t("quality.mustRemainZero")} tone={unsafe ? "critical" : "success"} />
        <MetricCard label={t("quality.workflowGates")} value={percent(value.workflow_gate_pass_rate)} detail={t("quality.evaluatedVersions", { count: value.workflow_evaluations })} />
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
          <div><dt>{t("quality.acceptance")}</dt><dd>{percent(value.recommendation_acceptance)}</dd></div>
          <div><dt>{t("quality.humanOverride")}</dt><dd>{percent(value.human_override_rate)}</dd></div>
          <div><dt>{t("quality.unsupported")}</dt><dd>{percent(value.unsupported_recommendation_rate)}</dd></div>
          <div><dt>{t("quality.incorrectNextStep")}</dt><dd>{percent(value.incorrect_next_step_rate)}</dd></div>
          <div><dt>{t("quality.retrievalPrecision")}</dt><dd>{percent(value.retrieval_precision)}</dd></div>
          <div><dt>{t("quality.llmCalls")}</dt><dd>{number(value.llm_calls)}</dd></div>
          <div><dt>{t("quality.avgLatency")}</dt><dd>{number(value.average_latency_ms)} ms</dd></div>
          <div><dt>{t("quality.tokens")}</dt><dd>{number(value.tokens_in + value.tokens_out)}</dd></div>
          <div><dt>{t("quality.estimatedCost")}</dt><dd>{cost(value.estimated_cost_micros)}</dd></div>
        </dl>
        <p className="muted">{t("quality.muted")}</p>
      </section>
    </>
  );
}
