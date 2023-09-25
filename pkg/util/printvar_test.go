package util

import "testing"

func TestPrintVariable(t *testing.T) {
	PrintVariable("", "test var")
	PrintVariable("a", "another test var")
	// Build test var: N/A
	// Build another test var: a
}
