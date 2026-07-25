package main

import "testing"

func TestSupportedTargets(t *testing.T) {
	t.Parallel()
	expected := map[string]bool{
		"windows/amd64/zip":   true,
		"windows/arm64/zip":   true,
		"darwin/amd64/tar.gz": true,
		"darwin/arm64/tar.gz": true,
		"linux/amd64/tar.gz":  true,
		"linux/arm64/tar.gz":  true,
	}
	for _, target := range targets {
		key := target.goos + "/" + target.goarch + "/" + target.format
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing package targets: %#v", expected)
	}
}

func TestSafeVersion(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"v0.1.0", "0.1.0-rc.1", "dev"} {
		if !safeVersion.MatchString(version) {
			t.Fatalf("expected version %q to be accepted", version)
		}
	}
	for _, version := range []string{"", "../release", "version with spaces", "v1\nunsafe"} {
		if safeVersion.MatchString(version) {
			t.Fatalf("expected version %q to be rejected", version)
		}
	}
}
