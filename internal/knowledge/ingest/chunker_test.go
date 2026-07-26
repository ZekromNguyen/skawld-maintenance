package ingest

import (
	"strings"
	"testing"
)

func TestChunkPagesProducesStableBoundedEvidence(t *testing.T) {
	t.Parallel()
	pages := []Page{
		{Number: 7, Text: "Inspect lubrication condition.\n\n" + strings.Repeat("x", 1600)},
	}
	first := ChunkPages(pages)
	second := ChunkPages(pages)
	if len(first) < 2 {
		t.Fatalf("chunks = %d, want at least 2", len(first))
	}
	if first[0].Locator != "page:7/chunk:1" {
		t.Fatalf("locator = %q", first[0].Locator)
	}
	for index := range first {
		if len(first[index].Content) > maxChunkCharacters {
			t.Errorf("chunk %d exceeds bound", index)
		}
		if first[index].ContentSHA256 != second[index].ContentSHA256 {
			t.Errorf("chunk %d hash is unstable", index)
		}
		if first[index].TokenEstimate <= 0 {
			t.Errorf("chunk %d has invalid token estimate", index)
		}
	}
}
