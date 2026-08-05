package postgres

import "testing"

func TestMimeMatchesNormalizesDetectedTypeParameters(t *testing.T) {
	tests := []struct {
		declared string
		verified string
		want     bool
	}{
		{"text/plain", "text/plain", true},
		{"text/plain", "text/plain; charset=utf-8", true},
		{"text/plain", "TEXT/PLAIN; charset=utf-8", true},
		{"application/pdf", "application/pdf", true},
		{"application/pdf", "application/pdf; charset=binary", true},
		{"image/jpeg", "image/jpeg", true},
		{"image/jpeg", "image/png", false},
		{"text/plain", "application/pdf", false},
		{"audio/m4a", "audio/mp4", true},
		{"audio/mpeg", "audio/mpeg", true},
		{"", "text/plain", false},
	}
	for _, test := range tests {
		if got := mimeMatches(test.declared, test.verified); got != test.want {
			t.Errorf("mimeMatches(%q, %q) = %v, want %v", test.declared, test.verified, got, test.want)
		}
	}
}

func TestNormalizeMIMEReturnsStableMediaType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text/plain; charset=utf-8", "text/plain"},
		{"text/plain", "text/plain"},
		{"TEXT/Plain", "text/plain"},
		{"application/pdf", "application/pdf"},
		{"image/jpeg; name=foo", "image/jpeg"},
		{"", ""},
	}
	for _, test := range tests {
		if got := normalizeMIME(test.input); got != test.want {
			t.Errorf("normalizeMIME(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
