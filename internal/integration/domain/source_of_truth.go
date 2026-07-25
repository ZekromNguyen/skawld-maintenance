package domain

import "fmt"

type SourceOfTruth string

const (
	OwnedBySkawld     SourceOfTruth = "OWNED_BY_SKAWLD"
	ExternalReference SourceOfTruth = "EXTERNAL_REFERENCE"
)

func (s SourceOfTruth) Validate(hasExternalReference bool) error {
	switch s {
	case OwnedBySkawld:
		if hasExternalReference {
			return fmt.Errorf("Skawld-owned record cannot declare an authoritative external reference")
		}
	case ExternalReference:
		if !hasExternalReference {
			return fmt.Errorf("external projection requires an external reference")
		}
	default:
		return fmt.Errorf("unsupported source of truth %q", s)
	}
	return nil
}
