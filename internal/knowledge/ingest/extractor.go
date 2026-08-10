package ingest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	maxDocumentBytes  = int64(50 << 20)
	maxExtractedBytes = int64(10 << 20)
	maxPages          = 500
)

var (
	ErrUnsupportedDocument = errors.New("unsupported document format")
	ErrDocumentTooLarge    = errors.New("document exceeds extraction limits")
	ErrNoExtractableText   = errors.New("document contains no extractable text; OCR is not enabled")
)

type Page struct {
	Number int
	Text   string
}

type Extractor interface {
	Extract(context.Context, io.Reader, string) ([]Page, error)
}

type BoundedExtractor struct {
	PDFToTextBinary string
}

func (e BoundedExtractor) Extract(
	ctx context.Context,
	source io.Reader,
	mimeType string,
) ([]Page, error) {
	switch mimeType {
	case "text/plain":
		content, err := readBounded(source, maxExtractedBytes)
		if err != nil {
			return nil, err
		}
		text := normalizeText(string(content))
		if text == "" {
			return nil, ErrNoExtractableText
		}
		return []Page{{Number: 1, Text: text}}, nil
	case "application/pdf":
		return e.extractPDF(ctx, source)
	default:
		return nil, ErrUnsupportedDocument
	}
}

func (e BoundedExtractor) extractPDF(
	ctx context.Context,
	source io.Reader,
) ([]Page, error) {
	binary := e.PDFToTextBinary
	if binary == "" {
		binary = "pdftotext"
	}
	input, err := os.CreateTemp("", "skawld-document-*.pdf")
	if err != nil {
		return nil, err
	}
	inputPath := input.Name()
	outputPath := inputPath + ".txt"
	defer func() {
		_ = os.Remove(inputPath)
		_ = os.Remove(outputPath)
	}()
	written, err := io.Copy(input, io.LimitReader(source, maxDocumentBytes+1))
	closeErr := input.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if written > maxDocumentBytes {
		return nil, ErrDocumentTooLarge
	}
	command := exec.CommandContext(
		ctx, binary, "-q", "-enc", "UTF-8", "-layout", inputPath, outputPath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("PDF text extraction failed: %w: %s", err, boundedError(output))
	}
	file, err := os.Open(outputPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := readBounded(file, maxExtractedBytes)
	if err != nil {
		return nil, err
	}
	rawPages := strings.Split(string(content), "\f")
	if len(rawPages) > maxPages {
		return nil, ErrDocumentTooLarge
	}
	pages := make([]Page, 0, len(rawPages))
	for index, raw := range rawPages {
		text := normalizeText(raw)
		if text != "" {
			pages = append(pages, Page{Number: index + 1, Text: text})
		}
	}
	if len(pages) == 0 {
		return nil, ErrNoExtractableText
	}
	return pages, nil
}

func readBounded(source io.Reader, limit int64) ([]byte, error) {
	var buffer bytes.Buffer
	written, err := io.Copy(&buffer, io.LimitReader(source, limit+1))
	if err != nil {
		return nil, err
	}
	if written > limit {
		return nil, ErrDocumentTooLarge
	}
	return buffer.Bytes(), nil
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = strings.TrimSpace(lines[index])
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func boundedError(value []byte) string {
	if len(value) > 512 {
		value = value[:512]
	}
	return strings.TrimSpace(string(value))
}
