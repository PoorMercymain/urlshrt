package domain

import "sync"

// Data which needs to be available on different levels of the app
type Data struct {
	urls       *[]URLStringJSON
	address    string
	randomSeed int64
	db         *Database
	json       OriginalURL
	*sync.Mutex
}

func NewData(urls *[]URLStringJSON, address string, randomSeed int64, db *Database, origURL string, isOrigURLSet bool, mutex *sync.Mutex) *Data {
	return &Data{urls: urls, address: address, randomSeed: randomSeed, db: db, json: OriginalURL{URL: origURL, IsSet: isOrigURLSet}, Mutex: mutex}
}
