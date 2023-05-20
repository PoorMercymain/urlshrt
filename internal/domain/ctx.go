package domain

type ctx struct {
	urls *[]URL
	address string
	randomSeed int64
	db *Database
}

func NewContext(urls *[]URL, address string, randomSeed int64, db *Database) *ctx {
	return &ctx{ urls, address, randomSeed, db }
}