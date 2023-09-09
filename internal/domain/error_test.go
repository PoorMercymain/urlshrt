package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueError(t *testing.T) {
	ue := NewUniqueError(errors.New("test"))
	require.Error(t, ue)

	str := ue.Error()
	require.NotEmpty(t, str)
}