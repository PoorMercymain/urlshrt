package util

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintVariable(t *testing.T) {
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	PrintVariable("", "test var")
	PrintVariable("a", "another test var")

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)

	got := buf.String()
	want := "Build test var: N/A\nBuild another test var: a\n"

	require.Equal(t, want, got)
}
