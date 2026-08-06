package httpserver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/ndjson"
	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/go-chi/chi/v5"
)

const maxImportBytes = 10 << 20

func mountIntegrationRoutes(router chi.Router, sink integrationapp.ProjectionSink) {
	router.Post("/integrations/imports", runIntegrationImport(sink))
}

func runIntegrationImport(sink integrationapp.ProjectionSink) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
		if err := r.ParseMultipartForm(maxImportBytes); err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid Request",
				"multipart body is required and must be at most 10 MiB")
			return
		}
		snapshot, _, err := r.FormFile("snapshot")
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid Request",
				"a snapshot file part is required")
			return
		}
		defer snapshot.Close()
		tempFile, err := os.CreateTemp("", "import-*.ndjson")
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "Server Error",
				"could not stage import snapshot")
			return
		}
		defer os.Remove(tempFile.Name())
		if _, err := io.Copy(tempFile, io.LimitReader(snapshot, maxImportBytes+1)); err != nil {
			tempFile.Close()
			writeProblem(w, http.StatusBadRequest, "Invalid Request",
				"could not read snapshot upload")
			return
		}
		if err := tempFile.Close(); err != nil {
			writeProblem(w, http.StatusInternalServerError, "Server Error",
				"could not stage import snapshot")
			return
		}
		system := strings.TrimSpace(r.FormValue("system"))
		instance := strings.TrimSpace(r.FormValue("instance"))
		version := strings.TrimSpace(r.FormValue("version"))
		if system == "" || instance == "" || version == "" {
			writeProblem(w, http.StatusBadRequest, "Invalid Request",
				"system, instance, and version form fields are required")
			return
		}
		connector, err := ndjson.New(tempFile.Name(), os.TempDir(), integrationdomain.ConnectorIdentity{
			System: system, Instance: instance, Version: version,
			ReadOnly: true, Capabilities: []integrationdomain.Capability{
				integrationdomain.CapabilityReadAssets,
			},
		})
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid Request", err.Error())
			return
		}
		importer := integrationapp.Importer{Connector: connector, Sink: sink}
		imported := 0
		cursor := ""
		for {
			page, err := importer.Pull(r.Context(), principal, cursor, 100)
			if err != nil {
				writeIntegrationImportError(w, err)
				return
			}
			imported += len(page.Records)
			if page.Complete {
				writeJSON(w, http.StatusOK, map[string]any{
					"imported":    imported,
					"next_cursor": nullableString(page.NextCursor),
					"complete":    true,
				})
				return
			}
			cursor = page.NextCursor
		}
	}
}

func writeIntegrationImportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, integrationapp.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "Forbidden",
			"external import is not permitted for this principal")
	case errors.Is(err, integrationapp.ErrInvalid):
		writeProblem(w, http.StatusBadRequest, "Invalid Request",
			"connector snapshot failed validation")
	default:
		writeProblem(w, http.StatusBadRequest, "Invalid Request",
			fmt.Sprintf("import failed: %v", err))
	}
}
