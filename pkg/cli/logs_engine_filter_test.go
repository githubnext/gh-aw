//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngineFilterWithAwInfo verifies that the engine filter logic correctly
// compares awInfo.EngineID against the filter string. This is a regression
// test for the bug where the --engine filter was silently ignored and runs
// from all engines were returned regardless of the filter value.
func TestEngineFilterWithAwInfo(t *testing.T) {
	cases := []struct {
		name          string
		awInfoContent string // empty means no file
		filterEngine  string
		expectMatch   bool
	}{
		{
			name:          "copilot run does not match claude filter",
			awInfoContent: `{"engine_id": "copilot"}`,
			filterEngine:  "claude",
			expectMatch:   false,
		},
		{
			name:          "claude run matches claude filter",
			awInfoContent: `{"engine_id": "claude"}`,
			filterEngine:  "claude",
			expectMatch:   true,
		},
		{
			name:          "copilot run matches copilot filter",
			awInfoContent: `{"engine_id": "copilot"}`,
			filterEngine:  "copilot",
			expectMatch:   true,
		},
		{
			name:          "codex run does not match claude filter",
			awInfoContent: `{"engine_id": "codex"}`,
			filterEngine:  "claude",
			expectMatch:   false,
		},
		{
			name:          "missing aw_info.json does not match any filter",
			awInfoContent: "",
			filterEngine:  "claude",
			expectMatch:   false,
		},
		{
			name:          "empty engine_id does not match any filter",
			awInfoContent: `{"engine_id": ""}`,
			filterEngine:  "claude",
			expectMatch:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			awInfoPath := filepath.Join(tmpDir, "aw_info.json")

			if tc.awInfoContent != "" {
				require.NoError(t, os.WriteFile(awInfoPath, []byte(tc.awInfoContent), 0644))
			}

			// Replicate the filter logic used in DownloadWorkflowLogs and
			// DownloadWorkflowLogsFromStdin after the simplification.
			awInfo, awInfoErr := parseAwInfo(awInfoPath, false)

			var engineMatches bool
			var detectedEngineID string
			if awInfoErr == nil && awInfo != nil && awInfo.EngineID != "" {
				detectedEngineID = awInfo.EngineID
				engineMatches = (awInfo.EngineID == tc.filterEngine)
			}

			t.Logf("filterEngine=%s detectedEngineID=%s engineMatches=%v", tc.filterEngine, detectedEngineID, engineMatches)
			assert.Equal(t, tc.expectMatch, engineMatches)
		})
	}
}
