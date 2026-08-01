package domain

type SearchQuery struct {
	SiteID         string `json:"site_id"`
	AssetID        string `json:"asset_id,omitempty"`
	Query          string `json:"query"`
	Limit          int    `json:"limit,omitempty"`
	Language       string `json:"language,omitempty"`
	ProcessService string `json:"process_service,omitempty"`
}

type Score struct {
	LexicalRank  int     `json:"lexical_rank,omitempty"`
	VectorRank   int     `json:"vector_rank,omitempty"`
	LexicalScore float64 `json:"lexical_score,omitempty"`
	VectorScore  float64 `json:"vector_score,omitempty"`
	RRFScore     float64 `json:"rrf_score"`
}

type Evidence struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	SourceID    string    `json:"source_id"`
	DocumentID  string    `json:"document_id,omitempty"`
	RevisionID  string    `json:"revision_id,omitempty"`
	Revision    string    `json:"revision,omitempty"`
	Title       string    `json:"title"`
	Locator     string    `json:"locator"`
	Authority   Authority `json:"authority"`
	Content     string    `json:"content"`
	ContentHash string    `json:"content_sha256"`
	Score       Score     `json:"score"`
}

type SearchResult struct {
	RetrievalRunID string     `json:"retrieval_run_id"`
	Items          []Evidence `json:"items"`
}
