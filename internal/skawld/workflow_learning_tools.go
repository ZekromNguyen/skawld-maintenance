package skawld

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdklearning "github.com/ZekromNguyen/skawld-sdk-go/learning"
	"github.com/ZekromNguyen/skawld-sdk-go/learning/structured"
	sdkobservation "github.com/ZekromNguyen/skawld-sdk-go/observation"
	sdktools "github.com/ZekromNguyen/skawld-sdk-go/tools"
	sdkworkflow "github.com/ZekromNguyen/skawld-sdk-go/workflow"
)

const (
	guideStartTool        = "maintenance.guide.start_work"
	guideMeasurementTool  = "maintenance.guide.record_measurement"
	guideObservationTool  = "maintenance.guide.record_observation"
	guideEvidenceTool     = "maintenance.guide.inspect_evidence"
	guidePrerequisiteTool = "maintenance.guide.verify_prerequisite"
	guideDecisionTool     = "maintenance.guide.record_decision"
	guideActionTool       = "maintenance.guide.record_action"
	guideOutcomeTool      = "maintenance.guide.verify_outcome"
	guideHandoverTool     = "maintenance.guide.shift_handover"
)

var maintenanceGuideToolNames = []string{
	guideActionTool,
	guideDecisionTool,
	guideEvidenceTool,
	guideHandoverTool,
	guideMeasurementTool,
	guideObservationTool,
	guideOutcomeTool,
	guidePrerequisiteTool,
	guideStartTool,
}

type guidanceTool struct {
	name string
}

func (t guidanceTool) Name() string {
	return t.name
}

func (t guidanceTool) Description() string {
	return "Render a reviewed maintenance workflow guidance step; it does not control equipment or mutate maintenance records."
}

func (t guidanceTool) InputSchema() map[string]interface{} {
	return objectContract(map[string]interface{}{
		"semantic_action": map[string]interface{}{
			"type":      "string",
			"minLength": 1,
			"maxLength": 200,
		},
		"entity_type": map[string]interface{}{
			"type":      "string",
			"maxLength": 100,
		},
	}, "semantic_action")
}

func (t guidanceTool) Scope() sdkcore.ToolScope {
	return sdkcore.ToolScopeRead
}

func (t guidanceTool) ParallelSafe() bool {
	return true
}

func (t guidanceTool) Validate(
	raw map[string]interface{},
) (map[string]interface{}, error) {
	if len(raw) < 1 || len(raw) > 2 {
		return nil, errors.New("guidance input must contain semantic_action and optional entity_type")
	}
	action, ok := raw["semantic_action"].(string)
	action = strings.TrimSpace(action)
	if !ok || action == "" || len(action) > 200 {
		return nil, errors.New("semantic_action must be a non-empty string")
	}
	result := map[string]interface{}{"semantic_action": action}
	if value, exists := raw["entity_type"]; exists {
		entity, ok := value.(string)
		entity = strings.TrimSpace(entity)
		if !ok || len(entity) > 100 {
			return nil, errors.New("entity_type must be a string")
		}
		if entity != "" {
			result["entity_type"] = entity
		}
	}
	for key := range raw {
		if key != "semantic_action" && key != "entity_type" {
			return nil, fmt.Errorf("unknown guidance input %q", key)
		}
	}
	return result, nil
}

func (t guidanceTool) Execute(
	input map[string]interface{},
	_ sdkcore.ToolContext,
) (sdkcore.ToolResult, error) {
	validated, err := t.Validate(input)
	if err != nil {
		return sdkcore.ToolResult{}, err
	}
	return sdkcore.ToolResult{
		Content: map[string]interface{}{
			"guidance_presented": true,
			"semantic_action":    validated["semantic_action"],
		},
		Summary: "reviewed guidance step presented",
	}, nil
}

func (t guidanceTool) Summarize(input map[string]interface{}) string {
	return fmt.Sprintf("Present reviewed step %v", input["semantic_action"])
}

func (t guidanceTool) ToolDescriptor() sdkcore.ToolDescriptor {
	return sdkcore.ToolDescriptor{
		Risk:        sdkcore.RiskLow,
		SideEffect:  sdkcore.SideEffectNone,
		Idempotency: sdkcore.IdempotencyNotApplicable,
		Permissions: []string{"workflow:read"},
		OutputSchema: objectContract(map[string]interface{}{
			"guidance_presented": map[string]interface{}{"type": "boolean"},
			"semantic_action":    map[string]interface{}{"type": "string"},
		}, "guidance_presented", "semantic_action"),
	}
}

func maintenanceToolCatalog() (*structured.RegistryCatalog, error) {
	registry := sdktools.NewRegistry()
	for _, name := range maintenanceGuideToolNames {
		if err := registry.Register(guidanceTool{name: name}); err != nil {
			return nil, err
		}
	}
	trusted := make(map[string]bool, len(maintenanceGuideToolNames))
	for _, name := range maintenanceGuideToolNames {
		trusted[name] = true
	}
	return structured.NewRegistryCatalog(structured.CatalogOptions{
		Registry:            registry,
		Names:               maintenanceGuideToolNames,
		TrustedDescriptions: trusted,
	})
}

type maintenanceWorkflowExtractor struct{}

func (maintenanceWorkflowExtractor) Extract(
	_ context.Context,
	request sdklearning.ExtractionRequest,
) (sdkworkflow.Version, error) {
	if request.Analysis == nil {
		return sdkworkflow.Version{}, errors.New("maintenance workflow extraction requires multi-demonstration analysis")
	}
	actions := append(
		[]sdklearning.ActionPattern(nil),
		request.Analysis.Actions...,
	)
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].MeanPosition == actions[j].MeanPosition {
			return actionSignatureKey(actions[i].Signature) <
				actionSignatureKey(actions[j].Signature)
		}
		return actions[i].MeanPosition < actions[j].MeanPosition
	})

	steps := make([]sdkworkflow.Step, 0, len(actions))
	for _, pattern := range actions {
		if !pattern.Common {
			continue
		}
		toolName, include, err := guideToolForAction(pattern.Signature.Action)
		if err != nil {
			return sdkworkflow.Version{}, err
		}
		if !include {
			continue
		}
		evidence := evidenceForPattern(request.Demonstrations, pattern.Signature)
		if len(evidence) == 0 {
			return sdkworkflow.Version{}, fmt.Errorf(
				"semantic action %q has no demonstration evidence",
				pattern.Signature.Action,
			)
		}
		stepID := fmt.Sprintf(
			"step_%02d_%s",
			len(steps)+1,
			workflowStepSegment(pattern.Signature.Action),
		)
		step := sdkworkflow.Step{
			ID:   stepID,
			Name: humanizeAction(pattern.Signature.Action),
			Kind: sdkworkflow.StepTool,
			Tool: &sdkworkflow.ToolCall{
				Name: toolName,
				Arguments: map[string]sdkworkflow.Value{
					"semantic_action": {
						Literal: pattern.Signature.Action,
					},
					"entity_type": {
						Literal: pattern.Signature.EntityType,
					},
				},
				Reason: "Supported by reviewed semantic demonstrations",
			},
			Evidence: evidence,
		}
		if len(steps) > 0 {
			step.DependsOn = []string{steps[len(steps)-1].ID}
		}
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return sdkworkflow.Version{}, errors.New(
			"reviewed demonstrations contain no supported procedural actions",
		)
	}
	return sdkworkflow.Version{
		Workflow: sdkworkflow.Workflow{
			ID: request.WorkflowID, TenantID: request.TenantID,
			Name:        request.WorkflowName,
			Description: "Deterministically compiled from reviewed maintenance demonstrations.",
		},
		Version: request.NextVersion,
		Status:  sdkworkflow.VersionCandidate,
		Steps:   steps,
	}, nil
}

func guideToolForAction(action string) (string, bool, error) {
	switch action {
	case "demonstration.started", "copilot.recommendation.corrected":
		return "", false, nil
	case "execution.created", "execution.started", "execution.step.completed":
		return guideStartTool, true, nil
	case "measurement.recorded":
		return guideMeasurementTool, true, nil
	case "observation.recorded":
		return guideObservationTool, true, nil
	case "evidence.viewed", "copilot.recommendation.generated":
		return guideEvidenceTool, true, nil
	case "execution.prerequisite.recorded":
		return guidePrerequisiteTool, true, nil
	case "decision.recorded":
		return guideDecisionTool, true, nil
	case "maintenance.action.recorded":
		return guideActionTool, true, nil
	case "execution.completed":
		return guideOutcomeTool, true, nil
	case "handover.draft.generated", "handover.edited", "handover.submitted",
		"handover.accepted", "handover.acknowledged":
		return guideHandoverTool, true, nil
	default:
		return "", false, fmt.Errorf(
			"unsupported maintenance semantic action %q", action,
		)
	}
}

func evidenceForPattern(
	demonstrations []sdkobservation.Demonstration,
	signature sdklearning.ActionSignature,
) []sdkworkflow.EvidenceRef {
	var result []sdkworkflow.EvidenceRef
	for _, demonstration := range demonstrations {
		var eventIDs []string
		for _, event := range demonstration.Trace.Events {
			entityType := ""
			if event.Entity != nil {
				entityType = event.Entity.Type
			}
			if event.Application == signature.Application &&
				event.Action == signature.Action &&
				entityType == signature.EntityType {
				eventIDs = append(eventIDs, event.ID)
			}
		}
		if len(eventIDs) > 0 {
			result = append(result, sdkworkflow.EvidenceRef{
				DemonstrationID: demonstration.ID,
				EventIDs:        eventIDs,
			})
		}
	}
	return result
}

func actionSignatureKey(value sdklearning.ActionSignature) string {
	return value.Application + "\x00" + value.Action + "\x00" + value.EntityType
}

func workflowStepSegment(value string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(value) {
		if character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' {
			builder.WriteRune(character)
		} else if builder.Len() > 0 {
			builder.WriteByte('_')
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "action"
	}
	return result
}

func humanizeAction(value string) string {
	value = strings.ReplaceAll(value, ".", " ")
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	for index := range parts {
		parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
	}
	return strings.Join(parts, " ")
}

func objectContract(
	properties map[string]interface{},
	required ...string,
) map[string]interface{} {
	result := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}
