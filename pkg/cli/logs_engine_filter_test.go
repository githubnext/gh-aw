//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMatchEngineFilter verifies that matchEngineFilter correctly compares
// awInfo.EngineID against the filter string. This is a regression test for the
// bug where the --engine filter was silently ignored and runs from all engines
// were returned regardless of the filter value.
func TestMatchEngineFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		awInfoContent    string // empty means no file
		filterEngine     string
		expectMatch      bool
		expectDetectedID string
	}{
		{
			name:             "copilot run does not match claude filter",
			awInfoContent:    `{"engine_id": "copilot"}`,
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "copilot",
		},
		{
			name:             "claude run matches claude filter",
			awInfoContent:    `{"engine_id": "claude"}`,
			filterEngine:     "claude",
			expectMatch:      true,
			expectDetectedID: "claude",
		},
		{
			name:             "copilot run matches copilot filter",
			awInfoContent:    `{"engine_id": "copilot"}`,
			filterEngine:     "copilot",
			expectMatch:      true,
			expectDetectedID: "copilot",
		},
		{
			name:             "codex run does not match claude filter",
			awInfoContent:    `{"engine_id": "codex"}`,
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "codex",
		},
		{
			name:             "missing aw_info.json does not match any filter",
			awInfoContent:    "",
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "",
		},
		{
			name:             "empty engine_id does not match any filter",
			awInfoContent:    `{"engine_id": ""}`,
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			awInfoPath := filepath.Join(tmpDir, "aw_info.json")

			if tc.awInfoContent != "" {
				require.NoError(t, os.WriteFile(awInfoPath, []byte(tc.awInfoContent), 0644))
			}

			awInfo, awInfoErr := parseAwInfo(awInfoPath, false)
			gotMatch, gotDetectedID := matchEngineFilter(awInfo, awInfoErr, tc.filterEngine)

			assert.Equal(t, tc.expectMatch, gotMatch, "match")
			assert.Equal(t, tc.expectDetectedID, gotDetectedID, "detectedEngineID")
		})
	}
}

// TestMatchRuntimeFilter verifies that matchRuntimeFilter correctly compares
// awInfo.AgentRuntime against the filter string.
func TestMatchRuntimeFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                  string
		awInfoContent         string // empty means no file
		filterRuntime         string
		expectMatch           bool
		expectDetectedRuntime string
	}{
		{
			name:                  "gvisor run does not match docker-sbx filter",
			awInfoContent:         `{"agent_runtime": "gvisor"}`,
			filterRuntime:         "docker-sbx",
			expectMatch:           false,
			expectDetectedRuntime: "gvisor",
		},
		{
			name:                  "gvisor run matches gvisor filter",
			awInfoContent:         `{"agent_runtime": "gvisor"}`,
			filterRuntime:         "gvisor",
			expectMatch:           true,
			expectDetectedRuntime: "gvisor",
		},
		{
			name:                  "docker-sbx run matches docker-sbx filter",
			awInfoContent:         `{"agent_runtime": "docker-sbx"}`,
			filterRuntime:         "docker-sbx",
			expectMatch:           true,
			expectDetectedRuntime: "docker-sbx",
		},
		{
			name:                  "missing aw_info.json does not match any filter",
			awInfoContent:         "",
			filterRuntime:         "gvisor",
			expectMatch:           false,
			expectDetectedRuntime: "",
		},
		{
			name:                  "empty agent_runtime does not match any filter",
			awInfoContent:         `{"agent_runtime": ""}`,
			filterRuntime:         "gvisor",
			expectMatch:           false,
			expectDetectedRuntime: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			awInfoPath := filepath.Join(tmpDir, "aw_info.json")

			if tc.awInfoContent != "" {
				require.NoError(t, os.WriteFile(awInfoPath, []byte(tc.awInfoContent), 0644))
			}

			awInfo, awInfoErr := parseAwInfo(awInfoPath, false)
			gotMatch, gotDetectedRuntime := matchRuntimeFilter(awInfo, awInfoErr, tc.filterRuntime)

			assert.Equal(t, tc.expectMatch, gotMatch, "match")
			assert.Equal(t, tc.expectDetectedRuntime, gotDetectedRuntime, "detectedRuntime")
		})
	}
}
