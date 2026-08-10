package sdkcontract

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integration "github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	sdkaudit "github.com/ZekromNguyen/skawld-sdk-go/audit"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	"github.com/ZekromNguyen/skawld-sdk-go/evaluation"
	"github.com/ZekromNguyen/skawld-sdk-go/learning"
	"github.com/ZekromNguyen/skawld-sdk-go/learning/structured"
	"github.com/ZekromNguyen/skawld-sdk-go/observation"
	"github.com/ZekromNguyen/skawld-sdk-go/policy"
	"github.com/ZekromNguyen/skawld-sdk-go/tools"
	"github.com/ZekromNguyen/skawld-sdk-go/workflow"
)

const recordObservationToolName = "maintenance.record_observation"

func TestPinnedSDKModuleContract(t *testing.T) {
	t.Parallel()
	if integration.IntegrationStatus != "pinned_contracts_verified" {
		t.Fatalf("integration status = %q", integration.IntegrationStatus)
	}
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve SDK contract test path")
	}
	moduleFile := filepath.Clean(
		filepath.Join(filepath.Dir(current), "..", "..", "..", "go.mod"),
	)
	raw, err := os.ReadFile(moduleFile)
	if err != nil {
		t.Fatal(err)
	}
	module := string(raw)
	expected := "github.com/ZekromNguyen/skawld-sdk-go " +
		integration.SDKVersion
	if !strings.Contains(module, expected) {
		t.Fatalf("go.mod does not pin %q", expected)
	}
	for _, line := range strings.Split(module, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "replace ") &&
			strings.Contains(line, "skawld-sdk-go") {
			t.Fatalf("release module must not replace SDK: %s", line)
		}
	}
}

func TestMaintenanceToolWorkflowApprovalAndAuditContract(t *testing.T) {
	t.Parallel()
	tool := &recordObservationTool{callsByKey: make(map[string]int)}
	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatal(err)
	}
	rolePolicy, err := policy.NewRolePolicy(policy.RolePolicyOptions{
		RoleCapabilities: integration.RoleCapabilities(),
	})
	if err != nil {
		t.Fatal(err)
	}
	approvalAuthorizer, err := policy.NewApprovalRolePolicy(
		policy.ApprovalRolePolicyOptions{
			RoleCapabilities: map[string][]string{
				integration.RoleName(identitydomain.RoleMaintenanceSupervisor): {
					"approval.grant",
					"approval.reject",
					"approval.cancel",
				},
			},
			RequireDistinctApprover: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	approvals, err := policy.NewAuthorizedApprovalStore(
		policy.NewMemoryApprovalStore(),
		approvalAuthorizer,
	)
	if err != nil {
		t.Fatal(err)
	}
	auditLog := &sdkaudit.MemoryStore{}
	executions := workflow.NewMemoryExecutionStore()
	executor, err := workflow.NewExecutor(workflow.ExecutorOptions{
		Tools:      workflow.RegistryRunner{Registry: registry},
		Policy:     rolePolicy,
		Approvals:  approvals,
		Audit:      auditLog,
		Executions: executions,
	})
	if err != nil {
		t.Fatal(err)
	}
	version := publishedWorkflowVersion()
	technicianContext, technician, err := integration.AuthenticatedContext(
		context.Background(),
		identitydomain.Principal{
			ID:             "technician-1",
			OrganizationID: "organization-1",
			Roles:          []identitydomain.Role{identitydomain.RoleSeniorTechnician},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := executor.Execute(
		technicianContext,
		version,
		map[string]interface{}{
			"observation":     "Motor-side bearing is hotter than normal.",
			"client_event_id": "event-1",
		},
		map[string]interface{}{"site_id": "site-1"},
		technician,
	)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Status != workflow.ExecutionAwaitingApproval ||
		checkpoint.PendingApprovalID == "" {
		t.Fatalf("approval checkpoint = %+v", checkpoint)
	}
	if calls := tool.callCount("event-1"); calls != 0 {
		t.Fatalf("tool calls before approval = %d", calls)
	}

	if _, err := approvals.Decide(
		technicianContext,
		checkpoint.PendingApprovalID,
		policy.ApprovalGranted,
		technician,
		"self approval",
	); err == nil {
		t.Fatal("requester must not approve their own high-risk action")
	}
	supervisorContext, supervisor, err := integration.AuthenticatedContext(
		context.Background(),
		identitydomain.Principal{
			ID:             "supervisor-1",
			OrganizationID: "organization-1",
			Roles: []identitydomain.Role{
				identitydomain.RoleMaintenanceSupervisor,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approvals.Decide(
		supervisorContext,
		checkpoint.PendingApprovalID,
		policy.ApprovalGranted,
		supervisor,
		"reviewed maintenance evidence",
	); err != nil {
		t.Fatal(err)
	}
	completed, err := executor.Resume(
		technicianContext,
		version,
		checkpoint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != workflow.ExecutionCompleted {
		t.Fatalf("execution status = %q, error=%+v", completed.Status, completed.Error)
	}
	if calls := tool.callCount("event-1"); calls != 1 {
		t.Fatalf("idempotent tool calls = %d, want 1", calls)
	}
	if _, err := executor.Resume(
		technicianContext,
		version,
		completed,
	); err == nil {
		t.Fatal("completed execution must not resume or duplicate its tool call")
	}
	if calls := tool.callCount("event-1"); calls != 1 {
		t.Fatalf("tool was duplicated after terminal resume: %d", calls)
	}
	events, err := auditLog.List(technicianContext, completed.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertAuditTypes(t, events,
		sdkaudit.EventExecutionStarted,
		sdkaudit.EventApprovalRequested,
		sdkaudit.EventApprovalDecided,
		sdkaudit.EventToolCalled,
		sdkaudit.EventToolCompleted,
		sdkaudit.EventExecutionEnded,
	)
}

func TestSemanticDemonstrationCompilationReviewAndPublicationContract(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	if err := registry.Register(
		&recordObservationTool{callsByKey: make(map[string]int)},
	); err != nil {
		t.Fatal(err)
	}
	catalog, err := structured.NewRegistryCatalog(structured.CatalogOptions{
		Registry: registry,
		Names:    []string{recordObservationToolName},
		TrustedDescriptions: map[string]bool{
			recordObservationToolName: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	maintenancePrincipal := identitydomain.Principal{
		ID:             "senior-technician-1",
		OrganizationID: "organization-1",
		Roles:          []identitydomain.Role{identitydomain.RoleSeniorTechnician},
	}
	ctx, principal, err := integration.AuthenticatedContext(
		context.Background(),
		maintenancePrincipal,
	)
	if err != nil {
		t.Fatal(err)
	}
	demonstrations := captureDemonstrations(t, ctx, principal)
	workflows := workflow.NewMemoryStore()
	compiler := learning.Compiler{
		Extractor: fixtureExtractor{},
		Tools:     catalog,
		Store:     workflows,
		InputSchema: objectSchema(map[string]interface{}{
			"observation":     map[string]interface{}{"type": "string"},
			"client_event_id": map[string]interface{}{"type": "string"},
		}, "observation", "client_event_id"),
		ContextSchema: objectSchema(map[string]interface{}{
			"site_id": map[string]interface{}{"type": "string"},
		}, "site_id"),
		Now: func() time.Time {
			return time.Date(2026, 7, 26, 5, 0, 0, 0, time.UTC)
		},
	}
	compilation, err := compiler.CompileMultiple(
		ctx,
		"high-vibration-pump",
		"High vibration pump inspection",
		demonstrations,
		learning.MultiDemoOptions{
			MinimumDemonstrations:         2,
			MinimumSequenceConsistency:    1,
			MinimumEvidenceDemonstrations: 2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	candidate := compilation.Candidate
	if candidate.Status != workflow.VersionCandidate ||
		candidate.ToolCatalogDigest == "" ||
		candidate.Learning == nil ||
		!candidate.Learning.RequiresHumanReview ||
		candidate.Learning.DemonstrationCount != 2 ||
		candidate.Learning.StepEvidenceCoverage != 1 {
		t.Fatalf("compiled candidate = %+v", candidate)
	}
	if published, exists, err := workflows.Published(
		ctx,
		candidate.Workflow.ID,
	); err != nil || exists || published.Status != "" {
		t.Fatalf("candidate published before review: %+v exists=%v err=%v", published, exists, err)
	}

	reviews := workflow.NewMemoryReviewStore()
	review, err := workflow.NewReview(
		candidate,
		workflow.ReviewApproved,
		principal,
		"evidence and maintenance tool mapping reviewed",
		time.Date(2026, 7, 26, 5, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := reviews.Save(ctx, review); err != nil {
		t.Fatal(err)
	}
	reports := evaluation.NewMemoryStore()
	suite := evaluation.Suite{
		Name: "maintenance-contract",
		Scenarios: []evaluation.Scenario{{
			ID: "verified-observation",
			Input: map[string]interface{}{
				"observation":     "Grease looks dark.",
				"client_event_id": "evaluation-event-1",
			},
			Context: map[string]interface{}{"site_id": "site-1"},
			Tools: map[string]evaluation.ToolFixture{
				recordObservationToolName: {
					Descriptor: recordObservationDescriptor(),
					Responses: []evaluation.ToolResponse{{
						Output: map[string]interface{}{
							"observation_id": "observation-evaluation-1",
							"recorded":       true,
						},
					}},
				},
			},
			Approvals: map[string]policy.ApprovalStatus{
				"record_observation": policy.ApprovalGranted,
			},
			Expected: evaluation.ExpectedOutcome{
				Status: workflow.ExecutionCompleted,
				ToolCalls: []evaluation.ExpectedToolCall{{
					Name: recordObservationToolName,
					Arguments: map[string]interface{}{
						"observation": "Grease looks dark.",
					},
				}},
				StepStatuses: map[string]workflow.StepStatus{
					"record_observation": workflow.StepCompleted,
				},
			},
		}},
		Gates: []evaluation.Gate{
			{
				Metric:   evaluation.MetricTaskSuccessRate,
				Operator: evaluation.GateAtLeast,
				Value:    1,
			},
			{
				Metric:   evaluation.MetricUnsafeActionRate,
				Operator: evaluation.GateAtMost,
				Value:    0,
			},
		},
	}
	report, err := evaluation.NewRunner(evaluation.RunnerOptions{
		Store: reports,
	}).Run(ctx, suite, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Gates.Passed {
		t.Fatalf("evaluation gates = %+v", report.Gates)
	}
	publicationAudit := &sdkaudit.MemoryStore{}
	publisher, err := evaluation.NewPublisher(evaluation.PublisherOptions{
		Workflows:     workflows,
		Reports:       reports,
		Reviews:       reviews,
		ToolCatalog:   catalog,
		Audit:         publicationAudit,
		RequiredSuite: suite.Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := publisher.Publish(
		ctx,
		candidate.Workflow.ID,
		candidate.Version,
		principal,
	)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != workflow.VersionPublished ||
		published.PublishedBy != principal.ActorID {
		t.Fatalf("published workflow = %+v", published)
	}
	events, err := publicationAudit.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	assertAuditTypes(t, events, sdkaudit.EventWorkflowPublished)
}

func TestSDKProviderStreamingContract(t *testing.T) {
	t.Parallel()
	provider := fixtureProvider{}
	var generic sdkcore.Provider = provider
	if !sdkcore.SupportsStreamingProvider(generic) {
		t.Fatal("fixture provider must satisfy the tagged streaming contract")
	}
	stream, err := sdkcore.StreamProvider(
		context.Background(),
		generic,
		sdkcore.ProviderRequest{Model: "fixture-model"},
	)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := <-stream
	if !ok || result.Err != nil ||
		result.Event.Type != "message_end" ||
		result.Event.StopReason != sdkcore.StopEndTurn {
		t.Fatalf("provider stream result = %+v, ok=%v", result, ok)
	}
	if _, open := <-stream; open {
		t.Fatal("provider stream did not close")
	}
}

type recordObservationTool struct {
	mu         sync.Mutex
	callsByKey map[string]int
}

var (
	_ sdkcore.Tool           = (*recordObservationTool)(nil)
	_ sdkcore.DescribedTool  = (*recordObservationTool)(nil)
	_ sdkcore.IdempotentTool = (*recordObservationTool)(nil)
)

func (*recordObservationTool) Name() string {
	return recordObservationToolName
}

func (*recordObservationTool) Description() string {
	return "Record one human-authored maintenance observation."
}

func (*recordObservationTool) InputSchema() map[string]interface{} {
	return objectSchema(map[string]interface{}{
		"observation": map[string]interface{}{"type": "string"},
	}, "observation")
}

func (*recordObservationTool) Scope() sdkcore.ToolScope {
	return sdkcore.ToolScopeWrite
}

func (*recordObservationTool) ParallelSafe() bool {
	return true
}

func (*recordObservationTool) Validate(
	raw map[string]interface{},
) (map[string]interface{}, error) {
	if len(raw) != 1 {
		return nil, errors.New("record observation input contains unknown fields")
	}
	value, ok := raw["observation"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len(value) > 4000 {
		return nil, errors.New("observation must be a bounded non-empty string")
	}
	return map[string]interface{}{"observation": value}, nil
}

func (*recordObservationTool) Execute(
	map[string]interface{},
	sdkcore.ToolContext,
) (sdkcore.ToolResult, error) {
	return sdkcore.ToolResult{}, errors.New("idempotent execution is required")
}

func (t *recordObservationTool) ExecuteIdempotent(
	input map[string]interface{},
	idempotencyKey string,
	ctx sdkcore.ToolContext,
) (sdkcore.ToolResult, error) {
	if err := ctx.Context.Err(); err != nil {
		return sdkcore.ToolResult{}, err
	}
	if idempotencyKey == "" {
		return sdkcore.ToolResult{}, errors.New("idempotency key is required")
	}
	t.mu.Lock()
	t.callsByKey[idempotencyKey]++
	call := t.callsByKey[idempotencyKey]
	t.mu.Unlock()
	return sdkcore.ToolResult{
		Content: map[string]interface{}{
			"observation_id": fmt.Sprintf("observation-%d", call),
			"recorded":       true,
		},
		Summary: "maintenance observation recorded",
	}, nil
}

func (*recordObservationTool) Summarize(
	map[string]interface{},
) string {
	return "record maintenance observation"
}

func (*recordObservationTool) ToolDescriptor() sdkcore.ToolDescriptor {
	return recordObservationDescriptor()
}

func (t *recordObservationTool) callCount(key string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.callsByKey[key]
}

func recordObservationDescriptor() sdkcore.ToolDescriptor {
	return sdkcore.ToolDescriptor{
		Risk:        sdkcore.RiskHigh,
		SideEffect:  sdkcore.SideEffectIdempotent,
		Idempotency: sdkcore.IdempotencyRequired,
		Timeout:     5 * time.Second,
		Permissions: []string{
			string(identitydomain.PermissionExecutionWrite),
		},
		OutputSchema: objectSchema(map[string]interface{}{
			"observation_id": map[string]interface{}{"type": "string"},
			"recorded":       map[string]interface{}{"type": "boolean"},
		}, "observation_id", "recorded"),
	}
}

func publishedWorkflowVersion() workflow.Version {
	key := workflow.Value{Ref: "input.client_event_id"}
	return workflow.Version{
		SchemaVersion: workflow.SchemaVersion,
		Workflow: workflow.Workflow{
			ID:       "high-vibration-pump",
			TenantID: "organization-1",
			Name:     "High vibration pump inspection",
		},
		Version: 1,
		Status:  workflow.VersionPublished,
		InputSchema: objectSchema(map[string]interface{}{
			"observation":     map[string]interface{}{"type": "string"},
			"client_event_id": map[string]interface{}{"type": "string"},
		}, "observation", "client_event_id"),
		ContextSchema: objectSchema(map[string]interface{}{
			"site_id": map[string]interface{}{"type": "string"},
		}, "site_id"),
		Steps: []workflow.Step{{
			ID:   "record_observation",
			Kind: workflow.StepTool,
			Tool: &workflow.ToolCall{
				Name: recordObservationToolName,
				Arguments: map[string]workflow.Value{
					"observation": {Ref: "input.observation"},
				},
				IdempotencyKey: &key,
				Reason:         "record technician evidence",
			},
		}},
		CreatedAt: time.Date(2026, 7, 26, 4, 0, 0, 0, time.UTC),
		CreatedBy: "workflow-reviewer-1",
	}
}

type fixtureExtractor struct{}

func (fixtureExtractor) Extract(
	_ context.Context,
	request learning.ExtractionRequest,
) (workflow.Version, error) {
	evidence := make([]workflow.EvidenceRef, 0, len(request.Demonstrations))
	for _, demonstration := range request.Demonstrations {
		events := demonstration.Trace.Events
		evidence = append(evidence, workflow.EvidenceRef{
			DemonstrationID: demonstration.ID,
			EventIDs:        []string{events[len(events)-1].ID},
		})
	}
	key := workflow.Value{Ref: "input.client_event_id"}
	return workflow.Version{Steps: []workflow.Step{{
		ID:       "record_observation",
		Kind:     workflow.StepTool,
		Evidence: evidence,
		Tool: &workflow.ToolCall{
			Name: recordObservationToolName,
			Arguments: map[string]workflow.Value{
				"observation": {Ref: "input.observation"},
			},
			IdempotencyKey: &key,
			Reason:         "learned from reviewed technician demonstrations",
		},
	}}}, nil
}

func captureDemonstrations(
	t *testing.T,
	ctx context.Context,
	principal sdkcore.Principal,
) []observation.Demonstration {
	t.Helper()
	store := observation.NewMemoryStore()
	recorder, err := observation.NewRecorder(store)
	if err != nil {
		t.Fatal(err)
	}
	output := make([]observation.Demonstration, 0, 2)
	for index := 1; index <= 2; index++ {
		demonstration, err := recorder.Start(
			ctx,
			"high-vibration-pump",
			principal,
			map[string]interface{}{
				"site_id":     "site-1",
				"asset_class": "centrifugal_pump",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		events := []observation.Event{
			{
				Principal:   principal,
				Source:      observation.SourceAPI,
				Trust:       observation.TrustApplicationEvent,
				Sensitivity: observation.SensitivityInternal,
				Application: "skawld-maintenance",
				Action:      "maintenance_history_inspected",
				Entity:      &observation.Entity{Type: "asset", ID: "P-302"},
			},
			{
				Principal:   principal,
				Source:      observation.SourceAPI,
				Trust:       observation.TrustApplicationEvent,
				Sensitivity: observation.SensitivityInternal,
				Application: "skawld-maintenance",
				Action:      "measurement_recorded",
				Entity:      &observation.Entity{Type: "asset", ID: "P-302"},
				Input: map[string]interface{}{
					"measurement_type": "vibration_velocity",
					"unit":             "mm/s",
				},
			},
			{
				Principal:   principal,
				Source:      observation.SourceAPI,
				Trust:       observation.TrustHumanInstruction,
				Sensitivity: observation.SensitivityInternal,
				Application: "skawld-maintenance",
				Action:      "observation_recorded",
				Entity:      &observation.Entity{Type: "component", ID: "bearing-de"},
				Context: map[string]interface{}{
					"verification": "technician_verified",
				},
			},
		}
		for _, event := range events {
			if _, err := recorder.Capture(ctx, demonstration.ID, event); err != nil {
				t.Fatal(err)
			}
		}
		completed, err := recorder.Complete(
			ctx,
			demonstration.ID,
			map[string]interface{}{"outcome": "inspection_completed"},
		)
		if err != nil {
			t.Fatal(err)
		}
		output = append(output, completed)
	}
	return output
}

type fixtureProvider struct{}

var _ sdkcore.StreamingProvider = fixtureProvider{}

func (fixtureProvider) ID() string {
	return "maintenance-fixture"
}

func (fixtureProvider) ContextWindow(sdkcore.ModelID) int {
	return 4096
}

func (fixtureProvider) Stream(
	ctx context.Context,
	_ sdkcore.ProviderRequest,
) sdkcore.ProviderStream {
	output := make(chan sdkcore.ProviderStreamResult, 1)
	select {
	case output <- sdkcore.ProviderStreamResult{
		Event: sdkcore.ProviderStreamEvent{
			Type:       "message_end",
			Model:      "fixture-model",
			StopReason: sdkcore.StopEndTurn,
		},
	}:
	case <-ctx.Done():
	}
	close(output)
	return output
}

func assertAuditTypes(
	t *testing.T,
	events []sdkaudit.Event,
	expected ...sdkaudit.EventType,
) {
	t.Helper()
	present := make(map[sdkaudit.EventType]bool, len(events))
	for _, event := range events {
		present[event.Type] = true
	}
	for _, eventType := range expected {
		if !present[eventType] {
			t.Fatalf("audit event %q missing from %+v", eventType, events)
		}
	}
}

func objectSchema(
	properties map[string]interface{},
	required ...string,
) map[string]interface{} {
	values := make([]interface{}, len(required))
	for index, value := range required {
		values[index] = value
	}
	return map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"required":             values,
		"additionalProperties": false,
	}
}
