package domain

import (
	"testing"
	"time"
)

func TestBuildUsesReviewedLabelsAndNeverDividesByZero(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	value := Build(Counts{
		Recommendations: 4, Reviewed: 3, Accepted: 1, Corrected: 1,
		Unsafe: 1, MaterialClaims: 5, SupportedClaims: 4,
		RetrievedEvidence: 4, RelevantEvidence: 3,
		AICalls: 4, LatencyTotalMS: 400, WorkflowEvaluations: 2,
		WorkflowGatesPassed: 1,
	}, now)
	if value.ReviewCoverage != 0.75 ||
		value.RecommendationAcceptance != 1.0/3.0 ||
		value.HumanOverrideRate != 2.0/3.0 ||
		value.UnsafeRecommendationRate != 0.25 ||
		value.EvidenceCoverage != 0.8 ||
		value.RetrievalPrecision != 0.75 ||
		value.AverageLatencyMS != 100 ||
		value.WorkflowGatePassRate != 0.5 {
		t.Fatalf("summary = %+v", value)
	}
	empty := Build(Counts{}, now)
	if empty.ReviewCoverage != 0 || empty.EvidenceCoverage != 0 {
		t.Fatalf("empty summary = %+v", empty)
	}
}
