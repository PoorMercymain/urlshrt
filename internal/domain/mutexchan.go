package domain

import "sync"

// MutexChanString is a type that represents a channel with mutex.
type MutexChanString struct {
	Channel chan URLWithID
	*sync.Mutex
}

// URLWithID is a type which represents an URL with id of it's user.
type URLWithID struct {
	URL string
	ID  int64
}

func NewMutexChanString(channel chan URLWithID) *MutexChanString {
	return &MutexChanString{Channel: channel, Mutex: &sync.Mutex{}}
}
