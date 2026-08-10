package domain

import (
	"errors"
	"strings"
	"time"
)

type ExecutionState string
type StepState string

const (
	ExecutionAssigned   ExecutionState = "ASSIGNED"
	ExecutionInProgress ExecutionState = "IN_PROGRESS"
	ExecutionCompleted  ExecutionState = "COMPLETED"

	StepPending   StepState = "PENDING"
	StepCompleted StepState = "COMPLETED"
	StepBlocked   StepState = "BLOCKED"
)

type Step struct {
	ID                   string
	Key                  string
	Sequence             int
	Title                string
	State                StepState
	RiskLevel            string
	RequiredPrerequisite string
	BlockedReason        string
	Version              int64
}

type Verification struct {
	Type              string
	Status            string
	ExternalReference string
	VerifiedBy        string
	VerifiedAt        time.Time
	ValidUntil        *time.Time
}

func (v Verification) Current(at time.Time) bool {
	if v.Status != "VERIFIED" || v.VerifiedAt.After(at) {
		return false
	}
	return v.ValidUntil == nil || at.Before(*v.ValidUntil)
}

type Execution struct {
	ID             string
	OrganizationID string
	SiteID         string
	IncidentID     string
	AssetID        string
	Purpose        string
	State          ExecutionState
	AssignedTo     string
	Version        int64
	StartedAt      *time.Time
	CompletedAt    *time.Time
	OutcomeSummary string
	Steps          []Step
	Verifications  []Verification
}

func NewExecution(value Execution) (Execution, error) {
	value.Purpose = strings.TrimSpace(value.Purpose)
	if value.ID == "" || value.OrganizationID == "" || value.SiteID == "" ||
		value.AssetID == "" || value.Purpose == "" {
		return Execution{}, errors.New("execution identity, scope, asset, and purpose are required")
	}
	if len(value.Steps) == 0 {
		return Execution{}, errors.New("execution requires at least one step")
	}
	value.State = ExecutionAssigned
	value.Version = 1
	for index := range value.Steps {
		if value.Steps[index].ID == "" || value.Steps[index].Key == "" ||
			value.Steps[index].Title == "" || value.Steps[index].Sequence <= 0 {
			return Execution{}, errors.New("execution step identity, key, sequence, and title are required")
		}
		value.Steps[index].State = StepPending
		value.Steps[index].Version = 1
	}
	return value, nil
}

func (e *Execution) Start(at time.Time) error {
	if e.State != ExecutionAssigned || at.IsZero() {
		return errors.New("only an assigned execution can start")
	}
	started := at.UTC()
	e.State = ExecutionInProgress
	e.StartedAt = &started
	e.Version++
	return nil
}

func (e *Execution) CompleteStep(stepID string, expectedVersion int64, at time.Time) error {
	if e.State != ExecutionInProgress {
		return errors.New("execution must be in progress")
	}
	for index := range e.Steps {
		step := &e.Steps[index]
		if step.ID != stepID {
			continue
		}
		if step.Version != expectedVersion {
			return errors.New("step version conflict")
		}
		if step.State == StepCompleted {
			return errors.New("step is already completed")
		}
		if step.RequiredPrerequisite != "" && !e.hasCurrentPrerequisite(step.RequiredPrerequisite, at) {
			step.State = StepBlocked
			step.BlockedReason = step.RequiredPrerequisite + " has not been verified"
			return errors.New(step.BlockedReason)
		}
		step.State = StepCompleted
		step.BlockedReason = ""
		step.Version++
		e.Version++
		return nil
	}
	return errors.New("step not found")
}

func (e *Execution) Finish(outcome string, at time.Time) error {
	if e.State != ExecutionInProgress {
		return errors.New("execution must be in progress")
	}
	for _, step := range e.Steps {
		if step.State != StepCompleted {
			return errors.New("all execution steps must be completed")
		}
	}
	outcome = strings.TrimSpace(outcome)
	if outcome == "" || at.IsZero() {
		return errors.New("execution outcome and completion time are required")
	}
	completed := at.UTC()
	e.State = ExecutionCompleted
	e.OutcomeSummary = outcome
	e.CompletedAt = &completed
	e.Version++
	return nil
}

func (e Execution) hasCurrentPrerequisite(prerequisite string, at time.Time) bool {
	for _, verification := range e.Verifications {
		if verification.Type == prerequisite && verification.Current(at) {
			return true
		}
	}
	return false
}

func PumpInspectionSteps(ids []string) []Step {
	if len(ids) != 4 {
		return nil
	}
	return []Step{
		{ID: ids[0], Key: "review_context", Sequence: 1, Title: "Review incident and asset history", RiskLevel: "INFORMATIONAL"},
		{ID: ids[1], Key: "external_visual", Sequence: 2, Title: "Perform external visual inspection", RiskLevel: "ADVISORY"},
		{ID: ids[2], Key: "baseline_measurements", Sequence: 3, Title: "Record vibration and bearing temperature", RiskLevel: "ADVISORY"},
		{ID: ids[3], Key: "intrusive_bearing_inspection", Sequence: 4, Title: "Inspect bearing and lubrication condition", RiskLevel: "SAFETY_SIGNIFICANT", RequiredPrerequisite: "ENERGY_ISOLATION"},
	}
}
