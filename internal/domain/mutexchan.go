package domain

import "sync"

type MutexChanString struct {
	Channel chan URLWithID
	*sync.Mutex
}

type URLWithID struct {
	URL string
	ID  int64
}

func NewMutexChanString(channel chan URLWithID) *MutexChanString {
	return &MutexChanString{Channel: channel, Mutex: &sync.Mutex{}}
}
