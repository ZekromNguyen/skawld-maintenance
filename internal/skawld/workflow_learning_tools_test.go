package skawld

import (
	"context"
	"strings"
	"testing"
	"time"

	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdklearning "github.com/ZekromNguyen/skawld-sdk-go/learning"
	sdkobservation "github.com/ZekromNguyen/skawld-sdk-go/observation"
	sdkworkflow "github.com/ZekromNguyen/skawld-sdk-go/workflow"
)

func TestMaintenanceGuidanceCatalogContainsOnlyReadOnlyClassifiedTools(
	t *testing.T,
) {
	t.Parallel()
	catalog, err := maintenanceToolCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range maintenanceGuideToolNames {
		descriptor, exists, err := catalog.Describe(
			context.Background(), name,
		)
		if err != nil {
			t.Fatal(err)
		}
		if !exists || descriptor.Risk != sdkcore.RiskLow ||
			descriptor.SideEffect != sdkcore.SideEffectNone ||
			descriptor.Idempotency != sdkcore.IdempotencyNotApplicable ||
			descriptor.NetworkAccess || descriptor.HandlesSecrets {
			t.Fatalf("unsafe or incomplete descriptor %s = %+v", name, descriptor)
		}
	}
	if _, exists, err := catalog.Describe(
		context.Background(), "maintenance.shutdown_equipment",
	); err != nil || exists {
		t.Fatalf("critical tool leaked into catalog, exists=%v err=%v", exists, err)
	}
}

func TestMaintenanceExtractorRejectsUnclassifiedSemanticAction(t *testing.T) {
	t.Parallel()
	demonstrations := workflowLearningUnitDemonstrations("plc.write_setpoint")
	analysis, err := sdklearning.Analyze(
		demonstrations, sdklearning.AnalyzerOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = (maintenanceWorkflowExtractor{}).Extract(
		context.Background(), sdklearning.ExtractionRequest{
			WorkflowID: "workflow-1", WorkflowName: "Unsafe",
			TenantID: "organization-1", NextVersion: 1,
			Demonstrations: demonstrations, Analysis: &analysis,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unclassified action error = %v", err)
	}
}

func TestCompilerRejectsFabricatedDemonstrationEvidence(t *testing.T) {
	t.Parallel()
	demonstrations := workflowLearningUnitDemonstrations("measurement.recorded")
	catalog, err := maintenanceToolCatalog()
	if err != nil {
		t.Fatal(err)
	}
	principal := sdkcore.Principal{
		TenantID: "organization-1", ActorID: "reviewer-1",
		Roles: []string{"senior_technician"},
	}
	compiler := sdklearning.Compiler{
		Extractor: fabricatedEvidenceExtractor{},
		Tools:     catalog,
		Store:     sdkworkflow.NewMemoryStore(),
	}
	_, err = compiler.CompileMultiple(
		sdkcore.WithPrincipal(context.Background(), principal),
		"workflow-1", "Fabricated", demonstrations,
		sdklearning.MultiDemoOptions{
			MinimumDemonstrations:         2,
			MinimumEvidenceDemonstrations: 2,
			MinimumSequenceConsistency:    1,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown event") {
		t.Fatalf("fabricated evidence error = %v", err)
	}
}

type fabricatedEvidenceExtractor struct{}

func (fabricatedEvidenceExtractor) Extract(
	context.Context,
	sdklearning.ExtractionRequest,
) (sdkworkflow.Version, error) {
	return sdkworkflow.Version{Steps: []sdkworkflow.Step{{
		ID: "fabricated", Kind: sdkworkflow.StepTool,
		Tool: &sdkworkflow.ToolCall{
			Name: guideMeasurementTool,
			Arguments: map[string]sdkworkflow.Value{
				"semantic_action": {Literal: "measurement.recorded"},
			},
		},
		Evidence: []sdkworkflow.EvidenceRef{
			{DemonstrationID: "demo-1", EventIDs: []string{"invented-event"}},
			{DemonstrationID: "demo-2", EventIDs: []string{"invented-event"}},
		},
	}}}, nil
}

func workflowLearningUnitDemonstrations(
	action string,
) []sdkobservation.Demonstration {
	principal := sdkcore.Principal{
		TenantID: "organization-1", ActorID: "technician-1",
		Roles: []string{"senior_technician"},
	}
	base := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	result := make([]sdkobservation.Demonstration, 0, 2)
	for index := 1; index <= 2; index++ {
		demonstrationID := "demo-" + string(rune('0'+index))
		sessionID := "session-" + string(rune('0'+index))
		eventID := "event-" + string(rune('0'+index))
		result = append(result, sdkobservation.Demonstration{
			ID: demonstrationID, WorkflowKey: "maintenance.execution.pump",
			Principal: principal, Status: sdkobservation.DemonstrationCompleted,
			StartedAt: base, CompletedAt: base.Add(time.Minute),
			Trace: sdkobservation.WorkflowTrace{
				SchemaVersion: sdkobservation.SchemaVersion,
				SessionID:     sessionID,
				Events: []sdkobservation.Event{{
					SchemaVersion: sdkobservation.SchemaVersion,
					ID:            eventID, SessionID: sessionID, Principal: principal,
					Timestamp: base, Source: sdkobservation.SourceDatabase,
					Trust:       sdkobservation.TrustApplicationEvent,
					Sensitivity: sdkobservation.SensitivityInternal,
					Application: "skawld-maintenance", Action: action,
					Entity: &sdkobservation.Entity{Type: "measurement"},
				}},
			},
		})
	}
	return result
}
