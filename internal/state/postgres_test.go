package state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	pg := &Postgres{}

	require.Empty(t, pg)

	ptr, err := pg.GetPgPtr()
	require.Error(t, err)
	require.Nil(t, ptr)

	dsn := pg.GetDSN()
	require.Empty(t, dsn)
}
