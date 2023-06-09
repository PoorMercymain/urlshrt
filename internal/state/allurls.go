package state

import (
	"errors"
	"sync"
)

type currentUrls struct {
	Urls *[]URLStringJSON
	*sync.Mutex
}

var urls *currentUrls

func InitCurrentURLs(startURLs *[]URLStringJSON) {
	urls = &currentUrls{Urls: startURLs, Mutex: new(sync.Mutex)}
}

func GetCurrentURLsPtr() (*currentUrls, error) {
	if urls != nil {
		return urls, nil
	} else {
		return nil, errors.New("curtrent urls should be initialized")
	}
}
