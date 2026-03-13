//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractLastConsoleMessage verifies that extractLastConsoleMessage correctly
// filters debug log lines and returns only user-facing console messages.
func TestExtractLastConsoleMessage(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		expected string
	}{
		{
			name: "filters debug logs and returns error message",
			stderr: `workflow:script_registry Creating new script registry +151ns
workflow:domains Loading ecosystem domains from embedded JSON +760µs
workflow:domains Loaded 31 ecosystem categories +161µs
cli:audit Starting audit for workflow run: runID=99999999999 +916µs
cli:audit Using output directory: /tmp/gh-aw/aw-mcp/logs/run-99999999999 +14µs
✗ failed to fetch run metadata: workflow run 99999999999 not found. Please verify the run ID is correct`,
			expected: "✗ failed to fetch run metadata: workflow run 99999999999 not found. Please verify the run ID is correct",
		},
		{
			name:     "empty stderr returns empty string",
			stderr:   "",
			expected: "",
		},
		{
			name:     "only whitespace returns empty string",
			stderr:   "   \n\n  ",
			expected: "",
		},
		{
			name:     "only debug logs falls back to last non-empty line",
			stderr:   "workflow:foo Starting +100ns\ncli:bar Processing +200µs",
			expected: "cli:bar Processing +200µs",
		},
		{
			name:     "console error message with no debug logs",
			stderr:   "✗ some error occurred",
			expected: "✗ some error occurred",
		},
		{
			name:     "console success message",
			stderr:   "✓ operation completed",
			expected: "✓ operation completed",
		},
		{
			name:     "console info message",
			stderr:   "ℹ loading configuration",
			expected: "ℹ loading configuration",
		},
		{
			name:     "console warning message",
			stderr:   "⚠ deprecated option",
			expected: "⚠ deprecated option",
		},
		{
			name: "multiple console messages returns last one",
			stderr: `ℹ starting up
✗ first error
✗ second error`,
			expected: "✗ second error",
		},
		{
			name: "debug logs after console message are skipped (last console returned)",
			stderr: `✗ some error
workflow:foo Cleanup +50ms`,
			expected: "✗ some error",
		},
		{
			name: "authentication error from logs command (GitHub Actions context)",
			stderr: `cli:logs_orchestrator Starting workflow log download: workflow=, count=100
ℹ Fetching workflow runs from GitHub Actions...
✗ GitHub CLI authentication required. Run 'gh auth login' first`,
			expected: "✗ GitHub CLI authentication required. Run 'gh auth login' first",
		},
		{
			name:     "cobra error format without console symbols",
			stderr:   "Error: GitHub CLI authentication required. Run 'gh auth login' first",
			expected: "Error: GitHub CLI authentication required. Run 'gh auth login' first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLastConsoleMessage(tt.stderr)
			assert.Equal(t, tt.expected, result, "should extract correct message from stderr")
		})
	}
}
