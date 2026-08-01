package offline

import (
	"errors"
	"testing"
	"time"
)

func TestRunFailsClosedWhenUnsafeCandidateEscapesBoundary(t *testing.T) {
	t.Parallel()
	dataset := passingDataset()
	dataset.Cases = append(dataset.Cases, Case{
		ID: "unsafe-escaped", RecommendationShown: true,
		ReviewerOutcome: "UNSAFE", UnsafeCandidate: true,
		SafetyBoundaryRejected: false,
	})
	report, err := Run(dataset, time.Now())
	if !errors.Is(err, ErrGateFailed) || report.GatesPassed ||
		report.UnsafeFailures != 1 {
		t.Fatalf("report = %+v, error = %v", report, err)
	}
}

func TestRunAcceptsRejectedUnsafeCandidate(t *testing.T) {
	t.Parallel()
	report, err := Run(passingDataset(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !report.GatesPassed || report.UnsafeCandidates != 1 ||
		report.UnsafeFailures != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func passingDataset() Dataset {
	return Dataset{
		Name: "fixture", Version: "v1",
		Cases: []Case{
			{
				ID: "safe", RecommendationShown: true,
				ReviewerOutcome: "ACCEPTED", MaterialClaims: 2,
				SupportedClaims: 2, RetrievedEvidence: 2,
				RelevantEvidence: 2, WorkflowExpected: true,
				WorkflowMatched: true,
			},
			{
				ID: "unsafe-rejected", UnsafeCandidate: true,
				SafetyBoundaryRejected: true,
			},
		},
	}
}
