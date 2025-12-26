//go:build integration

package workflow

import (
	"testing"
)

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
// TestSafeOutputsMCPBundlerIntegration tests that the safe-outputs workflow
// correctly includes child_process imports in the generated .cjs files
func TestSafeOutputsMCPBundlerIntegration(t *testing.T) {
	t.Skip("Test skipped - safe-outputs MCP scripts now use require() pattern and are loaded at runtime from external files")
}
