package domain

import "fmt"

// UniqueError is a type to check error of unique violation from database.
type UniqueError struct {
	Err error
}

func (ue *UniqueError) Error() string {
	return fmt.Sprintf("%v", ue.Err)
}

func NewUniqueError(err error) error {
	return &UniqueError{
		Err: err,
	}
}
