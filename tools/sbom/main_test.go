package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileParsersProducePackageURLs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	npmPath := filepath.Join(root, "package-lock.json")
	if err := os.WriteFile(npmPath, []byte(`{
		"packages":{"node_modules/react":{"version":"19.0.0","dev":false}}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	npm, err := npmModules(npmPath)
	if err != nil || len(npm) != 1 || npm[0].PURL != "pkg:npm/react@19.0.0" {
		t.Fatalf("npm = %+v, error = %v", npm, err)
	}
	pubPath := filepath.Join(root, "pubspec.lock")
	if err := os.WriteFile(pubPath, []byte(
		"packages:\n  drift:\n    source: hosted\n    version: \"2.0.0\"\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	pub, err := pubModules(pubPath)
	if err != nil || len(pub) != 1 || pub[0].PURL != "pkg:pub/drift@2.0.0" {
		t.Fatalf("pub = %+v, error = %v", pub, err)
	}
}
