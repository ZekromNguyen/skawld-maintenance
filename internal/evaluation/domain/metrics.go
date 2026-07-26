package domain

import "time"

type Counts struct {
	Recommendations     int64
	Reviewed            int64
	Accepted            int64
	Corrected           int64
	Rejected            int64
	Unsafe              int64
	Unsupported         int64
	IncorrectNextStep   int64
	MaterialClaims      int64
	SupportedClaims     int64
	RetrievedEvidence   int64
	RelevantEvidence    int64
	AICalls             int64
	LatencyTotalMS      int64
	TokensIn            int64
	TokensOut           int64
	EstimatedCostMicros int64
	WorkflowEvaluations int64
	WorkflowGatesPassed int64
}

type Summary struct {
	Recommendations               int64     `json:"recommendations"`
	Reviewed                      int64     `json:"reviewed"`
	ReviewCoverage                float64   `json:"review_coverage"`
	RecommendationAcceptance      float64   `json:"recommendation_acceptance"`
	HumanOverrideRate             float64   `json:"human_override_rate"`
	UnsafeRecommendationRate      float64   `json:"unsafe_recommendation_rate"`
	UnsupportedRecommendationRate float64   `json:"unsupported_recommendation_rate"`
	IncorrectNextStepRate         float64   `json:"incorrect_next_step_rate"`
	EvidenceCoverage              float64   `json:"evidence_coverage"`
	RetrievalPrecision            float64   `json:"retrieval_precision"`
	LLMCalls                      int64     `json:"llm_calls"`
	AverageLatencyMS              float64   `json:"average_latency_ms"`
	TokensIn                      int64     `json:"tokens_in"`
	TokensOut                     int64     `json:"tokens_out"`
	EstimatedCostMicros           int64     `json:"estimated_cost_micros"`
	WorkflowEvaluations           int64     `json:"workflow_evaluations"`
	WorkflowGatePassRate          float64   `json:"workflow_gate_pass_rate"`
	GeneratedAt                   time.Time `json:"generated_at"`
}

func Build(counts Counts, generatedAt time.Time) Summary {
	return Summary{
		Recommendations:          counts.Recommendations,
		Reviewed:                 counts.Reviewed,
		ReviewCoverage:           ratio(counts.Reviewed, counts.Recommendations),
		RecommendationAcceptance: ratio(counts.Accepted, counts.Reviewed),
		HumanOverrideRate: ratio(
			counts.Corrected+counts.Rejected+counts.Unsafe+
				counts.Unsupported+counts.IncorrectNextStep,
			counts.Reviewed,
		),
		UnsafeRecommendationRate: ratio(
			counts.Unsafe, counts.Recommendations,
		),
		UnsupportedRecommendationRate: ratio(
			counts.Unsupported, counts.Recommendations,
		),
		IncorrectNextStepRate: ratio(
			counts.IncorrectNextStep, counts.Recommendations,
		),
		EvidenceCoverage: ratio(
			counts.SupportedClaims, counts.MaterialClaims,
		),
		RetrievalPrecision: ratio(
			counts.RelevantEvidence, counts.RetrievedEvidence,
		),
		LLMCalls: counts.AICalls,
		AverageLatencyMS: ratio(
			counts.LatencyTotalMS, counts.AICalls,
		),
		TokensIn:            counts.TokensIn,
		TokensOut:           counts.TokensOut,
		EstimatedCostMicros: counts.EstimatedCostMicros,
		WorkflowEvaluations: counts.WorkflowEvaluations,
		WorkflowGatePassRate: ratio(
			counts.WorkflowGatesPassed, counts.WorkflowEvaluations,
		),
		GeneratedAt: generatedAt.UTC(),
	}
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
