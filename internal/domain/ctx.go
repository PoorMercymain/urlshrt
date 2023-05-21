package domain

type ctx struct {
	urls       *[]URL
	address    string
	randomSeed int64
	db         *Database
	json       OriginalURL
}

func NewContext(urls *[]URL, address string, randomSeed int64, db *Database, origURL string, isOrigURLSet bool) *ctx {
	return &ctx{urls, address, randomSeed, db, OriginalURL{URL: origURL, IsSet: isOrigURLSet}}
}