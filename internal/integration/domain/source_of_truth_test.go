package domain

import "testing"

func TestSourceOfTruthValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		source      SourceOfTruth
		hasExternal bool
		wantError   bool
	}{
		{name: "native", source: OwnedBySkawld},
		{name: "projection", source: ExternalReference, hasExternal: true},
		{name: "projection missing reference", source: ExternalReference, wantError: true},
		{name: "native claiming external authority", source: OwnedBySkawld, hasExternal: true, wantError: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.source.Validate(test.hasExternal)
			if (err != nil) != test.wantError {
				t.Fatalf("Validate() error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}
