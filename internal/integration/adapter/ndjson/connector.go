package ndjson

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

const maximumRecordBytes = 1024 * 1024

// Connector imports an immutable newline-delimited JSON export produced by an
// EAM/CMMS. It is deliberately read-only and does not make the external export
// authoritative inside Skawld.
type Connector struct {
	identity integrationdomain.ConnectorIdentity
	path     string
}

func New(
	path string,
	allowedRoot string,
	identity integrationdomain.ConnectorIdentity,
) (Connector, error) {
	if err := identity.Validate(); err != nil {
		return Connector{}, err
	}
	resolvedRoot, err := filepath.EvalSymlinks(allowedRoot)
	if err != nil {
		return Connector{}, fmt.Errorf("resolve connector root: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Connector{}, fmt.Errorf("resolve connector snapshot: %w", err)
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return Connector{}, errors.New("connector snapshot is outside allowed root")
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return Connector{}, fmt.Errorf("inspect connector snapshot: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Connector{}, errors.New("connector snapshot must be a regular file")
	}
	return Connector{identity: identity, path: resolvedPath}, nil
}

func (c Connector) Identity() integrationdomain.ConnectorIdentity {
	return c.identity
}

func (c Connector) Pull(
	ctx context.Context,
	request integrationdomain.PullRequest,
) (integrationdomain.Page, error) {
	if request.Limit < 1 || request.Limit > 500 {
		return integrationdomain.Page{}, errors.New("limit must be between 1 and 500")
	}
	file, err := os.Open(c.path)
	if err != nil {
		return integrationdomain.Page{}, fmt.Errorf("open connector snapshot: %w", err)
	}
	defer file.Close()
	contentDigest, err := digest(file)
	if err != nil {
		return integrationdomain.Page{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return integrationdomain.Page{}, fmt.Errorf("rewind connector snapshot: %w", err)
	}
	offset, err := parseCursor(request.Cursor, contentDigest)
	if err != nil {
		return integrationdomain.Page{}, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maximumRecordBytes)
	result := make([]integrationdomain.ExternalRecord, 0, request.Limit)
	line := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return integrationdomain.Page{}, err
		}
		if line < offset {
			line++
			continue
		}
		if len(result) == request.Limit {
			return integrationdomain.Page{
				Records: result,
				NextCursor: fmt.Sprintf(
					"%s:%d", contentDigest, line,
				),
				Complete: false,
			}, nil
		}
		record, err := decodeRecord(scanner.Bytes(), line+1)
		if err != nil {
			return integrationdomain.Page{}, errors.Join(integrationapp.ErrInvalid, err)
		}
		result = append(result, record)
		line++
	}
	if err := scanner.Err(); err != nil {
		return integrationdomain.Page{}, fmt.Errorf("scan connector snapshot: %w", err)
	}
	return integrationdomain.Page{Records: result, Complete: true}, nil
}

func digest(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("hash connector snapshot: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func parseCursor(cursor, expectedDigest string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	parts := strings.Split(cursor, ":")
	if len(parts) != 2 || parts[0] != expectedDigest {
		return 0, errors.New("connector cursor does not match current snapshot")
	}
	offset, err := strconv.Atoi(parts[1])
	if err != nil || offset < 0 {
		return 0, errors.New("connector cursor offset is invalid")
	}
	return offset, nil
}

func decodeRecord(
	content []byte,
	line int,
) (integrationdomain.ExternalRecord, error) {
	var record integrationdomain.ExternalRecord
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return integrationdomain.ExternalRecord{}, fmt.Errorf(
			"decode connector snapshot line %d: %w", line, err,
		)
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return integrationdomain.ExternalRecord{}, fmt.Errorf(
			"connector snapshot line %d contains trailing JSON", line,
		)
	}
	return record, nil
}
