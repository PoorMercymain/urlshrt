package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortAddress(t *testing.T) {
	InitShortAddress("ab")
	assert.Equal(t, "ab", GetBaseShortAddress())
}
