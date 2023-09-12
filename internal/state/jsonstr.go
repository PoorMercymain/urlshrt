package state

// URLStringJSON is a type which contains data which is needed for saving URLs in a database.
type URLStringJSON struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UUID        int    `json:"uuid"`
}
