package domain

import "fmt"

type UniqueError struct {
	Err error
}

func (ue *UniqueError) Error() string {
	return fmt.Sprintf("%v", ue.Err)
}

func NewUniqueError(err error) error {
	return &UniqueError{
		Err:  err,
	}
}

type ErrorDeleted struct {
	Err error
}

func (ed *ErrorDeleted) Error() string {
	return fmt.Sprintf("%v", ed.Err)
}

func NewErrorDeleted(err error) error {
	return &ErrorDeleted{
		Err:  err,
	}
}