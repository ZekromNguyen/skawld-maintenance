package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type component struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
	Scope   string `json:"scope,omitempty"`
}

type bom struct {
	Format     string      `json:"bomFormat"`
	Spec       string      `json:"specVersion"`
	Version    int         `json:"version"`
	Metadata   metadata    `json:"metadata"`
	Components []component `json:"components"`
}

type metadata struct {
	Component component `json:"component"`
}

func main() {
	output := flag.String(
		"output", "dist/sbom/skawld-maintenance.cdx.json",
		"CycloneDX JSON output path",
	)
	version := flag.String("version", "dev", "product version")
	flag.Parse()
	components, err := collect(".")
	if err != nil {
		fatal(err)
	}
	value := bom{
		Format: "CycloneDX", Spec: "1.6", Version: 1,
		Metadata: metadata{Component: component{
			Type: "application", Name: "skawld-maintenance",
			Version: *version,
			PURL:    "pkg:generic/skawld-maintenance@" + *version,
		}},
		Components: components,
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fatal(err)
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, content, 0o644); err != nil {
		fatal(err)
	}
	fmt.Println(*output)
}

func collect(root string) ([]component, error) {
	goComponents, err := goModules(root)
	if err != nil {
		return nil, err
	}
	npmComponents, err := npmModules(
		filepath.Join(root, "web", "package-lock.json"),
	)
	if err != nil {
		return nil, err
	}
	pubComponents, err := pubModules(
		filepath.Join(root, "mobile", "pubspec.lock"),
	)
	if err != nil {
		return nil, err
	}
	result := append(goComponents, npmComponents...)
	result = append(result, pubComponents...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].PURL < result[j].PURL
	})
	return result, nil
}

func goModules(root string) ([]component, error) {
	command := exec.Command("go", "list", "-m", "-json", "all")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list Go modules: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	var result []component
	for {
		var module struct {
			Path    string
			Version string
			Main    bool
			Replace *struct {
				Path    string
				Version string
			}
		}
		if err := decoder.Decode(&module); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		if module.Main {
			continue
		}
		if module.Replace != nil {
			module.Path = module.Replace.Path
			module.Version = module.Replace.Version
		}
		if module.Path == "" || module.Version == "" {
			continue
		}
		result = append(result, component{
			Type: "library", Name: module.Path, Version: module.Version,
			PURL:  "pkg:golang/" + module.Path + "@" + module.Version,
			Scope: "required",
		})
	}
	return result, nil
}

func npmModules(path string) ([]component, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lock struct {
		Packages map[string]struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Dev     bool   `json:"dev"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, err
	}
	var result []component
	for path, pkg := range lock.Packages {
		if path == "" || pkg.Version == "" {
			continue
		}
		name := pkg.Name
		if name == "" {
			index := strings.LastIndex(path, "node_modules/")
			name = path[index+len("node_modules/"):]
		}
		scope := "required"
		if pkg.Dev {
			scope = "optional"
		}
		result = append(result, component{
			Type: "library", Name: name, Version: pkg.Version,
			PURL: "pkg:npm/" + name + "@" + pkg.Version, Scope: scope,
		})
	}
	return result, nil
}

func pubModules(path string) ([]component, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var result []component
	var name string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "  ") &&
			!strings.HasPrefix(line, "    ") &&
			strings.HasSuffix(line, ":") {
			name = strings.TrimSuffix(strings.TrimSpace(line), ":")
			continue
		}
		if name != "" && strings.HasPrefix(line, "    version:") {
			version := strings.Trim(
				strings.TrimSpace(strings.TrimPrefix(
					strings.TrimSpace(line), "version:",
				)), `"`,
			)
			result = append(result, component{
				Type: "library", Name: name, Version: version,
				PURL: "pkg:pub/" + name + "@" + version, Scope: "required",
			})
			name = ""
		}
	}
	return result, scanner.Err()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
