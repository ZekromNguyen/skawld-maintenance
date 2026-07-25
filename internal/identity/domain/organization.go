package domain

import (
	"fmt"
	"strings"
	"time"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

type Organization struct {
	ID            string
	Name          string
	SourceOfTruth integrationdomain.SourceOfTruth
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewOrganization(id, name string, now time.Time) (Organization, error) {
	name = strings.TrimSpace(name)
	if id == "" {
		return Organization{}, fmt.Errorf("organization ID is required")
	}
	if name == "" {
		return Organization{}, fmt.Errorf("organization name is required")
	}
	return Organization{
		ID:            id,
		Name:          name,
		SourceOfTruth: integrationdomain.OwnedBySkawld,
		Version:       1,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}, nil
}
