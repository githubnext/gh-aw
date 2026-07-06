//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMatchEngineFilter verifies that matchEngineFilter correctly compares
// awInfo.EngineID against the filter string, and falls back to lockFileEngineID
// when aw_info.json is absent or has no engine_id.
func TestMatchEngineFilter(t *testing.T) {
	cases := []struct {
		name             string
		awInfoContent    string // empty means no file
		lockFileEngineID string
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
			name:             "missing aw_info.json does not match any filter without lock file fallback",
			awInfoContent:    "",
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "",
		},
		{
			name:             "empty engine_id does not match any filter without lock file fallback",
			awInfoContent:    `{"engine_id": ""}`,
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "",
		},
		// Lock file fallback cases
		{
			name:             "missing aw_info.json matches filter via lock file fallback",
			awInfoContent:    "",
			lockFileEngineID: "copilot",
			filterEngine:     "copilot",
			expectMatch:      true,
			expectDetectedID: "copilot",
		},
		{
			name:             "missing aw_info.json does not match different filter via lock file fallback",
			awInfoContent:    "",
			lockFileEngineID: "copilot",
			filterEngine:     "claude",
			expectMatch:      false,
			expectDetectedID: "copilot",
		},
		{
			name:             "empty engine_id in aw_info.json falls back to lock file",
			awInfoContent:    `{"engine_id": ""}`,
			lockFileEngineID: "copilot",
			filterEngine:     "copilot",
			expectMatch:      true,
			expectDetectedID: "copilot",
		},
		{
			name:             "aw_info.json engine_id takes precedence over lock file fallback",
			awInfoContent:    `{"engine_id": "claude"}`,
			lockFileEngineID: "copilot",
			filterEngine:     "claude",
			expectMatch:      true,
			expectDetectedID: "claude",
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
			gotMatch, gotDetectedID := matchEngineFilter(awInfo, awInfoErr, tc.filterEngine, tc.lockFileEngineID)

			assert.Equal(t, tc.expectMatch, gotMatch, "match")
			assert.Equal(t, tc.expectDetectedID, gotDetectedID, "detectedEngineID")
		})
	}
}

// TestExtractEngineIDFromLockFile verifies that extractEngineIDFromLockFile
// correctly reads the agent_id from the gh-aw-metadata comment in a lock file.
func TestExtractEngineIDFromLockFile(t *testing.T) {
	cases := []struct {
		name            string
		lockFileContent string // empty means no file
		workflowPath    string // relative path used in test
		expectEngineID  string
	}{
		{
			name: "copilot agent_id from lock file",
			lockFileContent: `# gh-aw-metadata: {"schema_version":"v3","frontmatter_hash":"abc123","agent_id":"copilot"}
name: daily-cli-tools-tester
`,
			expectEngineID: "copilot",
		},
		{
			name: "claude agent_id from lock file",
			lockFileContent: `# gh-aw-metadata: {"schema_version":"v3","frontmatter_hash":"def456","agent_id":"claude"}
name: smoke-claude
`,
			expectEngineID: "claude",
		},
		{
			name:            "missing lock file returns empty",
			lockFileContent: "",
			expectEngineID:  "",
		},
		{
			name: "lock file without agent_id returns empty",
			lockFileContent: `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"xyz789"}
name: no-agent-id-workflow
`,
			expectEngineID: "",
		},
		{
			name:           "empty workflow path returns empty",
			workflowPath:   "",
			expectEngineID: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Use .lock.yml extension by default
			workflowPath := tc.workflowPath
			if workflowPath == "" && tc.name != "empty workflow path returns empty" {
				// Create a lock file and use its path
				lockFilePath := filepath.Join(tmpDir, "test-workflow.lock.yml")
				if tc.lockFileContent != "" {
					require.NoError(t, os.WriteFile(lockFilePath, []byte(tc.lockFileContent), 0644))
				}
				workflowPath = lockFilePath
			}

			gotEngineID := extractEngineIDFromLockFile(workflowPath, false)
			assert.Equal(t, tc.expectEngineID, gotEngineID)
		})
	}
}

// TestResolveToLockFilePath verifies the path resolution logic.
func TestResolveToLockFilePath(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{".github/workflows/foo.lock.yml", ".github/workflows/foo.lock.yml"},
		{".github/workflows/foo.yml", ".github/workflows/foo.lock.yml"},
		{".github/workflows/foo.md", ".github/workflows/foo.lock.yml"},
		{"foo.lock.yml", "foo.lock.yml"},
		{"foo.yml", "foo.lock.yml"},
		{"foo.md", "foo.lock.yml"},
		{"", ""},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := resolveToLockFilePath(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestExtractEngineIDFromLockFilePlainYml verifies that a plain .yml path is
// resolved to the corresponding .lock.yml file.
func TestExtractEngineIDFromLockFilePlainYml(t *testing.T) {
	tmpDir := t.TempDir()

	lockContent := `# gh-aw-metadata: {"schema_version":"v3","frontmatter_hash":"abc","agent_id":"copilot"}
name: test
`
	// Write the lock file with .lock.yml extension
	lockFilePath := filepath.Join(tmpDir, "my-workflow.lock.yml")
	require.NoError(t, os.WriteFile(lockFilePath, []byte(lockContent), 0644))

	// Pass the .yml path (without .lock) — should resolve to the lock file
	ymlPath := strings.TrimSuffix(lockFilePath, ".lock.yml") + ".yml"
	got := extractEngineIDFromLockFile(ymlPath, false)
	assert.Equal(t, "copilot", got)
}
