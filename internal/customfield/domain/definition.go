package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type FieldType string

const (
	FieldTypeText        FieldType = "TEXT"
	FieldTypeNumber      FieldType = "NUMBER"
	FieldTypeDate        FieldType = "DATE"
	FieldTypeSelect      FieldType = "SELECT"
	FieldTypeMultiSelect FieldType = "MULTI_SELECT"
)

type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusRetired Status = "RETIRED"
)

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Config struct {
	Required  bool     `json:"required,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
	Regex     string   `json:"regex,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Options   []Option `json:"options,omitempty"`
}

type Definition struct {
	ID             string
	OrganizationID string
	EntityType     string
	Key            string
	Label          string
	Description    string
	FieldType      FieldType
	Config         Config
	Status         Status
	SortOrder      int
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RetiredAt      *time.Time
}

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func New(value Definition) (Definition, error) {
	value.Key = strings.TrimSpace(value.Key)
	value.Label = strings.TrimSpace(value.Label)
	value.EntityType = strings.TrimSpace(value.EntityType)
	if value.OrganizationID == "" || value.EntityType == "" {
		return Definition{}, errors.New("field definition scope is required")
	}
	if !keyPattern.MatchString(value.Key) {
		return Definition{}, errors.New("field key must match ^[a-z][a-z0-9_]{1,63}$")
	}
	if value.Label == "" {
		return Definition{}, errors.New("field label is required")
	}
	switch value.FieldType {
	case FieldTypeText, FieldTypeNumber, FieldTypeDate, FieldTypeSelect, FieldTypeMultiSelect:
	default:
		return Definition{}, fmt.Errorf("unsupported field type %q", value.FieldType)
	}
	if (value.FieldType == FieldTypeSelect || value.FieldType == FieldTypeMultiSelect) && len(value.Config.Options) == 0 {
		return Definition{}, errors.New("select fields require at least one option")
	}
	if value.FieldType == FieldTypeSelect || value.FieldType == FieldTypeMultiSelect {
		seen := make(map[string]struct{}, len(value.Config.Options))
		for _, option := range value.Config.Options {
			option.Value = strings.TrimSpace(option.Value)
			if option.Value == "" {
				return Definition{}, errors.New("select option values are required")
			}
			if _, dup := seen[option.Value]; dup {
				return Definition{}, fmt.Errorf("duplicate select option value %q", option.Value)
			}
			seen[option.Value] = struct{}{}
		}
	}
	if value.Status == "" {
		value.Status = StatusActive
	}
	if value.Status != StatusActive && value.Status != StatusRetired {
		return Definition{}, errors.New("unsupported field status")
	}
	value.Version = 1
	return value, nil
}

func (d Definition) Validate() error {
	_, err := New(d)
	return err
}

func ValidateValue(d Definition, value any) error {
	if value == nil {
		if d.Config.Required {
			return fmt.Errorf("%s is required", d.Label)
		}
		return nil
	}
	switch d.FieldType {
	case FieldTypeText:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be text", d.Label)
		}
		text = strings.TrimSpace(text)
		if d.Config.Required && text == "" {
			return fmt.Errorf("%s is required", d.Label)
		}
		if d.Config.MaxLength > 0 && len(text) > d.Config.MaxLength {
			return fmt.Errorf("%s must be at most %d characters", d.Label, d.Config.MaxLength)
		}
		if d.Config.Regex != "" {
			matched, err := regexp.MatchString(d.Config.Regex, text)
			if err != nil {
				return fmt.Errorf("%s has an invalid pattern", d.Label)
			}
			if !matched {
				return fmt.Errorf("%s does not match the required pattern", d.Label)
			}
		}
	case FieldTypeNumber:
		number, ok := toFloat(value)
		if !ok {
			return fmt.Errorf("%s must be a number", d.Label)
		}
		if d.Config.Min != nil && number < *d.Config.Min {
			return fmt.Errorf("%s must be at least %v", d.Label, *d.Config.Min)
		}
		if d.Config.Max != nil && number > *d.Config.Max {
			return fmt.Errorf("%s must be at most %v", d.Label, *d.Config.Max)
		}
	case FieldTypeDate:
		date, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a date", d.Label)
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return fmt.Errorf("%s must be a date in YYYY-MM-DD format", d.Label)
		}
	case FieldTypeSelect:
		selected, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a single option", d.Label)
		}
		if !hasOption(d, selected) {
			return fmt.Errorf("%s has an unknown option", d.Label)
		}
	case FieldTypeMultiSelect:
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be a list of options", d.Label)
		}
		if len(items) == 0 {
			return fmt.Errorf("%s must select at least one option", d.Label)
		}
		for _, item := range items {
			text, ok := item.(string)
			if !ok || !hasOption(d, text) {
				return fmt.Errorf("%s has an unknown option", d.Label)
			}
		}
	default:
		return fmt.Errorf("unsupported field type %q", d.FieldType)
	}
	return nil
}

func hasOption(d Definition, value string) bool {
	for _, option := range d.Config.Options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func toFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
