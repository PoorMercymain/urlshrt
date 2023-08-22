package domain

// BatchElement is a type which represent an element of a batch from JSON.
type BatchElement struct {
	ID           string `json:"correlation_id"`
	OriginalURL  string `json:"original_url"`
	ShortenedURL string `json:"short_url"`
}

// BatchElementResult is a type which shall be written to JSON in handler for batch and sent as a response.
type BatchElementResult struct {
	ID           string `json:"correlation_id"`
	ShortenedURL string `json:"short_url"`
}
