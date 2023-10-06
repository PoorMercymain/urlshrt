package domain

import "sync"

// MutexChanString is a type that represents a channel with mutex.
type MutexChanString struct { // actually, channels are safe for concurrent execution, so the type is cringy, but I'll keep it for now just to make sure that something won't stop working
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
