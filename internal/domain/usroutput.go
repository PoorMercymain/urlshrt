package domain

type UserOutput struct {
	ShortURL string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}