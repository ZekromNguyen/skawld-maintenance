package auth

import "testing"

func TestSafeReturnTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{input: "/incidents?state=open", want: "/incidents?state=open"},
		{input: "https://evil.example", want: "/"},
		{input: "//evil.example/path", want: "/"},
		{input: "incidents", want: "/"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			if got := safeReturnTo(test.input); got != test.want {
				t.Fatalf("safeReturnTo(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestRandomTokenIsNonEmptyAndUnique(t *testing.T) {
	t.Parallel()
	first, err := randomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first == second {
		t.Fatal("tokens must be non-empty and unique")
	}
}
