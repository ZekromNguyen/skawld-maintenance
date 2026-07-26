package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const buildInfoPackage = "github.com/ZekromNguyen/skawld-maintenance/internal/platform/buildinfo"

type target struct {
	goos   string
	goarch string
	format string
}

var targets = []target{
	{goos: "windows", goarch: "amd64", format: "zip"},
	{goos: "windows", goarch: "arm64", format: "zip"},
	{goos: "darwin", goarch: "amd64", format: "tar.gz"},
	{goos: "darwin", goarch: "arm64", format: "tar.gz"},
	{goos: "linux", goarch: "amd64", format: "tar.gz"},
	{goos: "linux", goarch: "arm64", format: "tar.gz"},
}

var safeVersion = regexp.MustCompile(`\A[A-Za-z0-9][A-Za-z0-9._-]*\z`)

func main() {
	var (
		version   = flag.String("version", "dev", "artifact version")
		commit    = flag.String("commit", "unknown", "source commit")
		builtAt   = flag.String("built-at", "", "RFC3339 build time")
		output    = flag.String("output", "dist/packages", "artifact output root")
		clean     = flag.Bool("clean", false, "replace an existing version directory")
		platforms = flag.String(
			"platforms",
			"all",
			"comma-separated target operating systems: windows,darwin,linux or all",
		)
	)
	flag.Parse()

	selectedTargets, err := selectTargets(*platforms)
	if err != nil {
		fmt.Fprintln(os.Stderr, "package:", err)
		os.Exit(1)
	}
	if err := run(*version, *commit, *builtAt, *output, *clean, selectedTargets); err != nil {
		fmt.Fprintln(os.Stderr, "package:", err)
		os.Exit(1)
	}
}

func run(
	version, commit, builtAt, output string,
	clean bool,
	buildTargets []target,
) error {
	if !safeVersion.MatchString(version) {
		return fmt.Errorf("version %q contains unsupported characters", version)
	}
	if strings.TrimSpace(commit) == "" || strings.ContainsAny(commit, "\r\n") {
		return errors.New("commit must be a non-empty single-line value")
	}
	buildTime := time.Now().UTC().Truncate(time.Second)
	if builtAt != "" {
		parsed, err := time.Parse(time.RFC3339, builtAt)
		if err != nil {
			return fmt.Errorf("parse built-at: %w", err)
		}
		buildTime = parsed.UTC()
	}

	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	outputRoot := output
	if !filepath.IsAbs(outputRoot) {
		outputRoot = filepath.Join(root, outputRoot)
	}
	releaseDir := filepath.Join(outputRoot, version)
	if _, err := os.Stat(releaseDir); err == nil {
		if !clean {
			return fmt.Errorf("%s already exists; pass -clean to replace it", releaseDir)
		}
		if err := os.RemoveAll(releaseDir); err != nil {
			return fmt.Errorf("clean release directory: %w", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect release directory: %w", err)
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return fmt.Errorf("create release directory: %w", err)
	}

	temporaryRoot, err := os.MkdirTemp("", "skawld-maintenance-package-*")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	var archives []string
	for _, buildTarget := range buildTargets {
		archivePath, err := buildTargetArchive(
			root,
			temporaryRoot,
			releaseDir,
			version,
			commit,
			buildTime,
			buildTarget,
		)
		if err != nil {
			return err
		}
		archives = append(archives, archivePath)
		fmt.Printf("created %s\n", archivePath)
	}
	if err := writeChecksums(releaseDir, archives); err != nil {
		return err
	}
	fmt.Printf("checksums %s\n", filepath.Join(releaseDir, "SHA256SUMS"))
	return nil
}

func selectTargets(platforms string) ([]target, error) {
	requested := strings.Split(strings.ToLower(strings.TrimSpace(platforms)), ",")
	if len(requested) == 1 && requested[0] == "all" {
		return append([]target(nil), targets...), nil
	}

	allowed := map[string]bool{
		"windows": true,
		"darwin":  true,
		"linux":   true,
	}
	selectedPlatforms := make(map[string]bool, len(requested))
	for _, platform := range requested {
		platform = strings.TrimSpace(platform)
		if platform == "" || !allowed[platform] {
			return nil, fmt.Errorf(
				"unsupported platform %q; use windows,darwin,linux or all",
				platform,
			)
		}
		selectedPlatforms[platform] = true
	}

	selected := make([]target, 0, len(targets))
	for _, buildTarget := range targets {
		if selectedPlatforms[buildTarget.goos] {
			selected = append(selected, buildTarget)
		}
	}
	return selected, nil
}

func repositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("go.mod not found in current directory or parents")
		}
		current = parent
	}
}

func buildTargetArchive(
	root, temporaryRoot, releaseDir, version, commit string,
	builtAt time.Time,
	buildTarget target,
) (string, error) {
	packageName := fmt.Sprintf(
		"skawld-maintenance_%s_%s_%s",
		version,
		buildTarget.goos,
		buildTarget.goarch,
	)
	staging := filepath.Join(temporaryRoot, packageName)
	if err := os.MkdirAll(filepath.Join(staging, "contracts"), 0o755); err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(staging, "evaldata"), 0o755); err != nil {
		return "", fmt.Errorf("create evaluation fixture directory: %w", err)
	}

	extension := ""
	if buildTarget.goos == "windows" {
		extension = ".exe"
	}
	ldflags := strings.Join([]string{
		"-s",
		"-w",
		"-X", buildInfoPackage + ".version=" + version,
		"-X", buildInfoPackage + ".commit=" + commit,
		"-X", buildInfoPackage + ".builtAt=" + builtAt.Format(time.RFC3339),
	}, " ")
	for _, binary := range []string{"api", "worker", "migrate", "eval"} {
		outputPath := filepath.Join(staging, binary+extension)
		command := exec.Command(
			"go", "build",
			"-trimpath",
			"-buildvcs=false",
			"-ldflags="+ldflags,
			"-o", outputPath,
			"./cmd/"+binary,
		)
		command.Dir = root
		command.Env = targetEnvironment(buildTarget)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return "", fmt.Errorf(
				"build %s for %s/%s: %w",
				binary,
				buildTarget.goos,
				buildTarget.goarch,
				err,
			)
		}
		info, err := os.Stat(outputPath)
		if err != nil {
			return "", fmt.Errorf("inspect packaged binary %s: %w", binary, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("packaged binary %s is not a regular file", outputPath)
		}
	}
	if err := copyFile(
		filepath.Join(root, "deployments", "packages", "README.txt"),
		filepath.Join(staging, "README.txt"),
	); err != nil {
		return "", err
	}
	if err := copyFile(
		filepath.Join(root, ".env.example"),
		filepath.Join(staging, ".env.example"),
	); err != nil {
		return "", err
	}
	if err := copyFile(
		filepath.Join(root, "api", "openapi.yaml"),
		filepath.Join(staging, "contracts", "openapi.yaml"),
	); err != nil {
		return "", err
	}
	if err := copyFile(
		filepath.Join(root, "test", "evaldata", "pilot-v1.json"),
		filepath.Join(staging, "evaldata", "pilot-v1.json"),
	); err != nil {
		return "", err
	}

	archivePath := filepath.Join(releaseDir, packageName+"."+buildTarget.format)
	switch buildTarget.format {
	case "zip":
		if err := writeZip(archivePath, staging, builtAt); err != nil {
			return "", err
		}
	case "tar.gz":
		if err := writeTarGzip(archivePath, staging, builtAt); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported archive format %q", buildTarget.format)
	}
	return archivePath, nil
}

func targetEnvironment(buildTarget target) []string {
	environment := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOOS=") ||
			strings.HasPrefix(entry, "GOARCH=") ||
			strings.HasPrefix(entry, "CGO_ENABLED=") {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment,
		"GOOS="+buildTarget.goos,
		"GOARCH="+buildTarget.goarch,
		"CGO_ENABLED=0",
	)
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open package file %s: %w", source, err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create package file %s: %w", destination, err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return fmt.Errorf("copy package file %s: %w", source, err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close package file %s: %w", destination, err)
	}
	return nil
}

func writeZip(destination, source string, modified time.Time) error {
	output, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	writer := zip.NewWriter(output)
	walkErr := walkPackage(source, func(path, archiveName string, info fs.FileInfo) error {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = archiveName
		header.Modified = modified
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		entry, err := writer.CreateHeader(header)
		if err != nil || info.IsDir() {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		_, err = io.Copy(entry, input)
		return err
	})
	closeErr := writer.Close()
	fileCloseErr := output.Close()
	return errors.Join(walkErr, closeErr, fileCloseErr)
}

func writeTarGzip(destination, source string, modified time.Time) error {
	output, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create tar archive: %w", err)
	}
	gzipWriter := gzip.NewWriter(output)
	gzipWriter.Header.ModTime = modified
	tarWriter := tar.NewWriter(gzipWriter)
	walkErr := walkPackage(source, func(path, archiveName string, info fs.FileInfo) error {
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = archiveName
		header.ModTime = modified
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}
		if err := tarWriter.WriteHeader(header); err != nil || info.IsDir() {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		_, err = io.Copy(tarWriter, input)
		return err
	})
	tarCloseErr := tarWriter.Close()
	gzipCloseErr := gzipWriter.Close()
	fileCloseErr := output.Close()
	return errors.Join(walkErr, tarCloseErr, gzipCloseErr, fileCloseErr)
}

func walkPackage(
	source string,
	visit func(path, archiveName string, info fs.FileInfo) error,
) error {
	parent := filepath.Dir(source)
	return filepath.Walk(source, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}
		return visit(path, filepath.ToSlash(relative), info)
	})
}

func writeChecksums(releaseDir string, archives []string) error {
	sort.Strings(archives)
	output, err := os.Create(filepath.Join(releaseDir, "SHA256SUMS"))
	if err != nil {
		return fmt.Errorf("create checksum manifest: %w", err)
	}
	defer output.Close()
	for _, archivePath := range archives {
		input, err := os.Open(archivePath)
		if err != nil {
			return fmt.Errorf("open archive for checksum: %w", err)
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, input)
		closeErr := input.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return fmt.Errorf("hash archive: %w", err)
		}
		if _, err := fmt.Fprintf(
			output,
			"%s  %s\n",
			hex.EncodeToString(hash.Sum(nil)),
			filepath.Base(archivePath),
		); err != nil {
			return fmt.Errorf("write checksum manifest: %w", err)
		}
	}
	return nil
}
