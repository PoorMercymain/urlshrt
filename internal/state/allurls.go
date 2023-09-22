package state

import (
	"errors"
	"sync"
)

type currentUrls struct {
	Urls *map[string]URLStringJSON
	*sync.Mutex
}

// TODO: change from global
var urls *currentUrls

// InitCurrentURLs is a function to initialize struct with map of current URLs.
func InitCurrentURLs(startURLs *map[string]URLStringJSON) {
	urls = &currentUrls{Urls: startURLs, Mutex: new(sync.Mutex)}
}

// GetCurrentURLsPtr is a function to get pointer to struct with map of current URLs.
func GetCurrentURLsPtr() (*currentUrls, error) {
	if urls != nil {
		return urls, nil
	} else {
		return nil, errors.New("current urls should be initialized")
	}
}
