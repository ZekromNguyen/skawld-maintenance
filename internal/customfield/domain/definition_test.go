package domain

import (
	"strings"
	"testing"
)

func TestNewValidatesKey(t *testing.T) {
	for _, key := range []string{"po_number", "a", "shutdown_time"} {
		if _, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: key, Label: "L", FieldType: FieldTypeText}); err != nil {
			t.Errorf("key %q should be valid: %v", key, err)
		}
	}
	for _, key := range []string{"", "PO_Number", "1abc", "with space", strings.Repeat("x", 65), "has-dash"} {
		if _, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: key, Label: "L", FieldType: FieldTypeText}); err == nil {
			t.Errorf("key %q should be invalid", key)
		}
	}
}

func TestNewRequiresSelectOptions(t *testing.T) {
	_, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: "zone", Label: "Zone", FieldType: FieldTypeSelect, Config: Config{Options: nil}})
	if err == nil {
		t.Fatal("SELECT without options must fail")
	}
	if _, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: "zone", Label: "Zone", FieldType: FieldTypeSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}}}}); err != nil {
		t.Fatalf("SELECT with options must pass: %v", err)
	}
}

func TestValidateValueText(t *testing.T) {
	d := Definition{FieldType: FieldTypeText, Config: Config{Required: true, MaxLength: 10}}
	if err := ValidateValue(d, nil); err == nil {
		t.Error("required nil must fail")
	}
	if err := ValidateValue(d, "toolongvalue"); err == nil {
		t.Error("over max length must fail")
	}
	if err := ValidateValue(d, "ok"); err != nil {
		t.Errorf("valid text must pass: %v", err)
	}
	if err := ValidateValue(d, 42); err == nil {
		t.Error("non-string must fail")
	}
}

func TestValidateValueNumber(t *testing.T) {
	min, max := 0.0, 100.0
	d := Definition{FieldType: FieldTypeNumber, Config: Config{Min: &min, Max: &max}}
	if err := ValidateValue(d, 50.0); err != nil {
		t.Errorf("in-range must pass: %v", err)
	}
	if err := ValidateValue(d, 101.0); err == nil {
		t.Error("above max must fail")
	}
	if err := ValidateValue(d, "fifty"); err == nil {
		t.Error("non-number must fail")
	}
}

func TestValidateValueDate(t *testing.T) {
	d := Definition{FieldType: FieldTypeDate}
	if err := ValidateValue(d, "2026-08-09"); err != nil {
		t.Errorf("valid date must pass: %v", err)
	}
	if err := ValidateValue(d, "not-a-date"); err == nil {
		t.Error("invalid date must fail")
	}
}

func TestValidateValueSelect(t *testing.T) {
	d := Definition{FieldType: FieldTypeSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}}}
	if err := ValidateValue(d, "a"); err != nil {
		t.Errorf("known option must pass: %v", err)
	}
	if err := ValidateValue(d, "zzz"); err == nil {
		t.Error("unknown option must fail")
	}
}

func TestValidateValueMultiSelect(t *testing.T) {
	d := Definition{FieldType: FieldTypeMultiSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}}}
	if err := ValidateValue(d, []any{"a", "b"}); err != nil {
		t.Errorf("known options must pass: %v", err)
	}
	if err := ValidateValue(d, []any{"a", "zzz"}); err == nil {
		t.Error("unknown option must fail")
	}
	if err := ValidateValue(d, []any{}); err == nil {
		t.Error("empty array must fail")
	}
}

func TestValidateValueRegex(t *testing.T) {
	d := Definition{FieldType: FieldTypeText, Config: Config{Regex: `^PO-\d+$`}}
	if err := ValidateValue(d, "PO-123"); err != nil {
		t.Errorf("matching value must pass: %v", err)
	}
	if err := ValidateValue(d, "nope"); err == nil {
		t.Error("non-matching value must fail")
	}
}

func TestNewRequiresScopeAndLabel(t *testing.T) {
	base := Definition{OrganizationID: "o1", EntityType: "incident", Key: "po", Label: "PO", FieldType: FieldTypeText}
	if _, err := New(base); err != nil {
		t.Fatalf("valid definition must pass: %v", err)
	}
	noOrg := base
	noOrg.OrganizationID = ""
	if _, err := New(noOrg); err == nil {
		t.Error("missing organization must fail")
	}
	noEntity := base
	noEntity.EntityType = ""
	if _, err := New(noEntity); err == nil {
		t.Error("missing entity type must fail")
	}
	noLabel := base
	noLabel.Label = ""
	if _, err := New(noLabel); err == nil {
		t.Error("missing label must fail")
	}
	badType := base
	badType.FieldType = "URL"
	if _, err := New(badType); err == nil {
		t.Error("unsupported field type must fail")
	}
}

func TestNewStatusDefaultsAndValidates(t *testing.T) {
	base := Definition{OrganizationID: "o1", EntityType: "incident", Key: "po", Label: "PO", FieldType: FieldTypeText}
	value, err := New(base)
	if err != nil {
		t.Fatalf("valid definition must pass: %v", err)
	}
	if value.Status != StatusActive {
		t.Fatalf("status must default to ACTIVE, got %q", value.Status)
	}
	badStatus := base
	badStatus.Status = Status("DRAFT")
	if _, err := New(badStatus); err == nil {
		t.Error("unsupported status must fail")
	}
}

func TestNewTrimsAndDeduplicatesOptionValues(t *testing.T) {
	base := Definition{
		OrganizationID: "o1", EntityType: "incident", Key: "zone", Label: "Zone",
		FieldType: FieldTypeSelect,
		Config:    Config{Options: []Option{{Label: "A", Value: "  a  "}, {Label: "B", Value: "b"}}},
	}
	value, err := New(base)
	if err != nil {
		t.Fatalf("options with whitespace must be accepted: %v", err)
	}
	if value.Config.Options[0].Value != "a" {
		t.Fatalf("option value must be trimmed, got %q", value.Config.Options[0].Value)
	}
	if err := ValidateValue(value, "a"); err != nil {
		t.Fatalf("trimmed option must be selectable: %v", err)
	}
	dup := base
	dup.Config.Options = []Option{{Label: "A", Value: "a"}, {Label: "B", Value: " a "}}
	if _, err := New(dup); err == nil {
		t.Error("duplicate option values after trim must fail")
	}
	empty := base
	empty.Config.Options = []Option{{Label: "A", Value: "  "}}
	if _, err := New(empty); err == nil {
		t.Error("empty option value after trim must fail")
	}
}

func TestValidateValueNilNotRequired(t *testing.T) {
	d := Definition{FieldType: FieldTypeText}
	if err := ValidateValue(d, nil); err != nil {
		t.Errorf("nil with Required=false must pass: %v", err)
	}
}

func TestValidateValueNumberBelowMin(t *testing.T) {
	min := 10.0
	d := Definition{FieldType: FieldTypeNumber, Config: Config{Min: &min}}
	if err := ValidateValue(d, 5.0); err == nil {
		t.Error("below min must fail")
	}
}

func TestValidateValueSelectNonString(t *testing.T) {
	d := Definition{FieldType: FieldTypeSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}}}}
	if err := ValidateValue(d, 42); err == nil {
		t.Error("non-string select value must fail")
	}
}

func TestValidateValueDateStrictFormat(t *testing.T) {
	d := Definition{FieldType: FieldTypeDate}
	if err := ValidateValue(d, "2026-8-9"); err == nil {
		t.Error("non-padded date must fail strict YYYY-MM-DD")
	}
	if err := ValidateValue(d, "2026-08-09"); err != nil {
		t.Errorf("padded date must pass: %v", err)
	}
}

func TestValidateValueTextMaxLengthRunes(t *testing.T) {
	d := Definition{FieldType: FieldTypeText, Config: Config{MaxLength: 3}}
	if err := ValidateValue(d, "e\u0301"); err != nil {
		t.Errorf("3 combining runes within max 3 must pass: %v", err)
	}
	if err := ValidateValue(d, "éééé"); err == nil {
		t.Error("4 characters must fail max length 3")
	}
}
