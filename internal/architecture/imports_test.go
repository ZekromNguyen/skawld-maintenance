package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestDependencyBoundaries(t *testing.T) {
	t.Parallel()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test path")
	}
	internalRoot := filepath.Clean(filepath.Join(filepath.Dir(current), ".."))

	err := filepath.WalkDir(internalRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		relative, err := filepath.Rel(internalRoot, path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if strings.Contains(importPath, "skawld-sdk-go") &&
				!strings.HasPrefix(filepath.ToSlash(relative), "skawld/") {
				t.Errorf("%s imports SDK package %q outside internal/skawld", relative, importPath)
			}
			if isDomainPath(relative) && forbiddenDomainImport(importPath) {
				t.Errorf("%s domain package imports infrastructure %q", relative, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSDKImportsAreConfinedToIntegrationAndContractTests(t *testing.T) {
	t.Parallel()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test path")
	}
	repositoryRoot := filepath.Clean(
		filepath.Join(filepath.Dir(current), "..", ".."),
	)
	err := filepath.WalkDir(
		repositoryRoot,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				switch entry.Name() {
				case ".git", "node_modules", ".dart_tool", "build", "dist":
					return filepath.SkipDir
				default:
					return nil
				}
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			relative, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return err
			}
			file, err := parser.ParseFile(
				token.NewFileSet(),
				path,
				nil,
				parser.ImportsOnly,
			)
			if err != nil {
				return err
			}
			for _, imported := range file.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				if !strings.Contains(importPath, "skawld-sdk-go") {
					continue
				}
				slash := filepath.ToSlash(relative)
				if strings.HasPrefix(slash, "internal/skawld/") ||
					strings.HasPrefix(slash, "test/contract/sdk/") {
					continue
				}
				t.Errorf(
					"%s imports SDK package %q outside the integration boundary",
					relative,
					importPath,
				)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func isDomainPath(path string) bool {
	slash := "/" + filepath.ToSlash(path)
	return strings.Contains(slash, "/domain/")
}

func forbiddenDomainImport(path string) bool {
	for _, prefix := range []string{
		"net/http",
		"github.com/jackc/pgx",
		"github.com/riverqueue/river",
		"github.com/coreos/go-oidc",
		"golang.org/x/oauth2",
		"github.com/ZekromNguyen/skawld-maintenance/internal/platform",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
