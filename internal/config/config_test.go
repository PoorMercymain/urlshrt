package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddrWithCheck(t *testing.T) {
	a := AddrWithCheck{}
	require.Empty(t, a)
	require.False(t, a.WasSet)

	require.NoError(t, a.Set("ab"))
	require.NotEmpty(t, a)
	require.Len(t, a.Addr, 2)
	require.True(t, a.WasSet)

	c := Config{HTTPAddr: a, ShortAddr: a, JSONFile: "a", DSN: "a", HTTPSEnabled: "y"}
	require.NotEmpty(t, c)

	str := a.String()
	require.NotEmpty(t, str)
}
