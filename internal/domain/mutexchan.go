package domain

import "sync"

type MutexChanString struct {
	Channel chan string
	*sync.Mutex
}

func NewMutexChanString(channel chan string) *MutexChanString {
	return &MutexChanString{Channel: channel, Mutex: &sync.Mutex{}}
}
