package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	logger := GetLogger()
	require.Empty(t, logger)

	err := InitLogger()
	require.NoError(t, err)

	logger = GetLogger()
	require.NotEmpty(t, logger)
}
