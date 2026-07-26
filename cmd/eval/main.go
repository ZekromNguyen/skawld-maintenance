package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/offline"
)

func main() {
	datasetPath := flag.String(
		"dataset", "test/evaldata/pilot-v1.json",
		"frozen evaluation dataset",
	)
	outputPath := flag.String(
		"output", "", "optional JSON report path; stdout when empty",
	)
	flag.Parse()

	dataset, err := offline.Load(*datasetPath)
	if err != nil {
		fatal(err)
	}
	report, evaluationErr := offline.Run(dataset, time.Now().UTC())
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(err)
	}
	content = append(content, '\n')
	if *outputPath == "" {
		if _, err := os.Stdout.Write(content); err != nil {
			fatal(err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
			fatal(err)
		}
		if err := os.WriteFile(*outputPath, content, 0o644); err != nil {
			fatal(err)
		}
	}
	if evaluationErr != nil {
		fatal(evaluationErr)
	}
}

func fatal(err error) {
	if errors.Is(err, offline.ErrGateFailed) {
		fmt.Fprintln(os.Stderr, "pilot evaluation gates failed")
		os.Exit(3)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
