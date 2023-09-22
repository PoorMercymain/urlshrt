package util

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	rand := rand.New(rand.NewSource(0))
	str := GenerateRandomString(0, rand)
	require.Empty(t, str)

	str = GenerateRandomString(15, rand)
	require.Len(t, str, 15)
}
