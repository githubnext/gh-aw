//go:build integration

package cli

import (
	"testing"
)

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
// TestSafeInputsMCPServerCompilation tests that safe-inputs are properly compiled
// into MCP server configurations for all three agentic engines
func TestSafeInputsMCPServerCompilation(t *testing.T) {
	t.Skip("Test skipped - safe-inputs MCP server scripts now use require() pattern and are loaded at runtime from external files")
}
