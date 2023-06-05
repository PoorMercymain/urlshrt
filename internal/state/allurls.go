package state

import "sync"

type currentUrls struct {
	Urls *[]URLStringJSON
	*sync.Mutex
}

var (
	urls *currentUrls
)

func InitCurrentURLs(startURLs *[]URLStringJSON) {
	urls = &currentUrls{Urls: startURLs, Mutex: new(sync.Mutex)}
}

func GetCurrentURLsPtr() *currentUrls {
	return urls
}