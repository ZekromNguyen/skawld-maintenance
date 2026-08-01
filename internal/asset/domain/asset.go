package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

type Status string

const (
	StatusActive       Status = "ACTIVE"
	StatusOutOfService Status = "OUT_OF_SERVICE"
	StatusRetired      Status = "RETIRED"
)

type ExternalReference struct {
	System  string
	ID      string
	Version string
}

type Asset struct {
	ID                string
	OrganizationID    string
	SiteID            string
	Tag               string
	Name              string
	Class             string
	Manufacturer      string
	Model             string
	Status            Status
	SourceOfTruth     integrationdomain.SourceOfTruth
	ExternalReference *ExternalReference
	Version           int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type NewAsset struct {
	ID                string
	OrganizationID    string
	SiteID            string
	Tag               string
	Name              string
	Class             string
	Manufacturer      string
	Model             string
	SourceOfTruth     integrationdomain.SourceOfTruth
	ExternalReference *ExternalReference
	Now               time.Time
}

func Create(input NewAsset) (Asset, error) {
	input.Tag = strings.TrimSpace(input.Tag)
	input.Name = strings.TrimSpace(input.Name)
	input.Class = strings.TrimSpace(input.Class)
	if input.ID == "" || input.OrganizationID == "" || input.SiteID == "" {
		return Asset{}, errors.New("asset ID, organization, and site are required")
	}
	if input.Tag == "" || input.Name == "" || input.Class == "" {
		return Asset{}, errors.New("asset tag, name, and class are required")
	}
	hasExternal := input.ExternalReference != nil &&
		strings.TrimSpace(input.ExternalReference.System) != "" &&
		strings.TrimSpace(input.ExternalReference.ID) != ""
	if err := input.SourceOfTruth.Validate(hasExternal); err != nil {
		return Asset{}, err
	}
	if input.Now.IsZero() {
		return Asset{}, errors.New("asset creation time is required")
	}
	return Asset{
		ID:                input.ID,
		OrganizationID:    input.OrganizationID,
		SiteID:            input.SiteID,
		Tag:               input.Tag,
		Name:              input.Name,
		Class:             input.Class,
		Manufacturer:      strings.TrimSpace(input.Manufacturer),
		Model:             strings.TrimSpace(input.Model),
		Status:            StatusActive,
		SourceOfTruth:     input.SourceOfTruth,
		ExternalReference: input.ExternalReference,
		Version:           1,
		CreatedAt:         input.Now.UTC(),
		UpdatedAt:         input.Now.UTC(),
	}, nil
}

func (a Asset) RequireNativeMutation() error {
	if a.SourceOfTruth != integrationdomain.OwnedBySkawld {
		return errors.New("external asset projection cannot be mutated as a Skawld-owned asset")
	}
	return nil
}

type Component struct {
	ID             string
	OrganizationID string
	AssetID        string
	Code           string
	Name           string
	Type           string
}

func NewComponent(id, organizationID, assetID, code, name, componentType string) (Component, error) {
	component := Component{
		ID:             id,
		OrganizationID: organizationID,
		AssetID:        assetID,
		Code:           strings.TrimSpace(code),
		Name:           strings.TrimSpace(name),
		Type:           strings.TrimSpace(componentType),
	}
	if component.ID == "" || component.OrganizationID == "" || component.AssetID == "" ||
		component.Code == "" || component.Name == "" || component.Type == "" {
		return Component{}, errors.New("component ID, scope, code, name, and type are required")
	}
	return component, nil
}

type Containment struct {
	ParentID string
	ChildID  string
}

func ValidateContainment(newEdge Containment, existing []Containment) error {
	if newEdge.ParentID == "" || newEdge.ChildID == "" {
		return errors.New("containment parent and child are required")
	}
	if newEdge.ParentID == newEdge.ChildID {
		return errors.New("asset cannot contain itself")
	}
	children := make(map[string][]string)
	for _, edge := range existing {
		children[edge.ParentID] = append(children[edge.ParentID], edge.ChildID)
	}
	children[newEdge.ParentID] = append(children[newEdge.ParentID], newEdge.ChildID)
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) error
	visit = func(node string) error {
		if visiting[node] {
			return fmt.Errorf("containment cycle detected at asset %s", node)
		}
		if visited[node] {
			return nil
		}
		visiting[node] = true
		for _, child := range children[node] {
			if err := visit(child); err != nil {
				return err
			}
		}
		visiting[node] = false
		visited[node] = true
		return nil
	}
	for node := range children {
		if err := visit(node); err != nil {
			return err
		}
	}
	return nil
}
