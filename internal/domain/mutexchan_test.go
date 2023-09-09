package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMutexChan(t *testing.T) {
	ch := make(chan URLWithID)
	mc := NewMutexChanString(ch)
	require.NotEmpty(t, mc)
}