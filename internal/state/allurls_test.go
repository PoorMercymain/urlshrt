package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllURLs(t *testing.T) {
	_, err := GetCurrentURLsPtr()
	assert.Error(t, err)
	someMap := make(map[string]URLStringJSON)
	someMap["a"] = URLStringJSON{}
	InitCurrentURLs(&someMap)
	curURLs, err := GetCurrentURLsPtr()
	require.NoError(t, err)
	assert.NotEmpty(t, *curURLs.Urls)
}