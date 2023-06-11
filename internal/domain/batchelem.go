package domain

type BatchElement struct {
	ID string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
	ShortenedURL string `json:"short_url"`
	NeedsWriting bool `json:"-"`
}

type BatchElementResult struct {
	ID string `json:"correlation_id"`
	ShortenedURL string `json:"short_url"`
}