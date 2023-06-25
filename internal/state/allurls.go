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

func InitCurrentURLs(startURLs *map[string]URLStringJSON) {
	urls = &currentUrls{Urls: startURLs, Mutex: new(sync.Mutex)}
}

func GetCurrentURLsPtr() (*currentUrls, error) {
	if urls != nil {
		return urls, nil
	} else {
		return nil, errors.New("current urls should be initialized")
	}
}
