package domain

// UserOutput is a type of element used to show user's URLs in specific handler.
type UserOutput struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
