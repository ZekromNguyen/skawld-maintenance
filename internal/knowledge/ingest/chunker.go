package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const maxChunkCharacters = 1400

type Chunk struct {
	Ordinal       int
	Locator       string
	Content       string
	ContentSHA256 string
	TokenEstimate int
}

func ChunkPages(pages []Page) []Chunk {
	var result []Chunk
	ordinal := 1
	for _, page := range pages {
		paragraphs := strings.Split(page.Text, "\n\n")
		var current strings.Builder
		flush := func() {
			content := strings.TrimSpace(current.String())
			current.Reset()
			if content == "" {
				return
			}
			sum := sha256.Sum256([]byte(content))
			result = append(result, Chunk{
				Ordinal: ordinal,
				Locator: fmt.Sprintf("page:%d/chunk:%d", page.Number, ordinal),
				Content: content, ContentSHA256: hex.EncodeToString(sum[:]),
				TokenEstimate: max(1, (len(strings.Fields(content))*4+2)/3),
			})
			ordinal++
		}
		for _, paragraph := range paragraphs {
			paragraph = strings.TrimSpace(paragraph)
			if paragraph == "" {
				continue
			}
			if current.Len() > 0 && current.Len()+len(paragraph)+2 > maxChunkCharacters {
				flush()
			}
			if len(paragraph) > maxChunkCharacters {
				for len(paragraph) > maxChunkCharacters {
					current.WriteString(paragraph[:maxChunkCharacters])
					flush()
					paragraph = strings.TrimSpace(paragraph[maxChunkCharacters:])
				}
			}
			if paragraph == "" {
				continue
			}
			if current.Len() > 0 {
				current.WriteString("\n\n")
			}
			current.WriteString(paragraph)
		}
		flush()
	}
	return result
}
