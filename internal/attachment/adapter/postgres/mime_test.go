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
		// Issue #8 regression: declared m4a with parameters still aliases mp4.
		{"audio/m4a; charset=binary", "audio/mp4", true},
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

// TestDetectMIMERecognizesRIFFWAVAndFtypMP4Prefixes pins the magic-number
// branches of detectMIME that http.DetectContentType alone misclassifies:
// a WAV file's RIFF/WAVE header and an MP4 container's ftyp box.
func TestDetectMIMERecognizesRIFFWAVAndFtypMP4Prefixes(t *testing.T) {
	tests := []struct {
		name   string
		prefix []byte
		want   string
	}{
		{
			name:   "WAV RIFF header",
			prefix: []byte("RIFF\x00\x00\x00\x00WAVEfmt "),
			want:   "audio/wav",
		},
		{
			name:   "MP4 ftyp box",
			prefix: []byte("\x00\x00\x00\x18ftypM4A \x00\x00\x00\x00"),
			want:   "audio/mp4",
		},
		{
			name:   "MP4 ftyp box with isom brand",
			prefix: []byte("\x00\x00\x00\x20ftypisom\x00\x00\x00\x00"),
			want:   "audio/mp4",
		},
		{
			name:   "plain text yields text/plain",
			prefix: []byte("hello world, just ASCII bytes here"),
			want:   "text/plain; charset=utf-8",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if got := detectMIME(test.prefix); got != test.want {
				t.Fatalf("detectMIME(%q) = %q, want %q", test.prefix, got, test.want)
			}
		})
	}
}
