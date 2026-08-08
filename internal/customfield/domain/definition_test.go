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
