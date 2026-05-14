//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCTR015AllowedLabelsGlobScope tests that the compiler warns (CTR-015) when
// a bare "*" wildcard appears in any safe-outputs allowed-labels field.
func TestCTR015AllowedLabelsGlobScope(t *testing.T) {
	basePermissions := `
permissions:
  contents: read
  issues: read

on:
  issues:
    types: [opened]

engine: claude
strict: false
`

	tests := []struct {
		name        string
		safeOutputs string
		expectWarn  bool
	}{
		{
			name: "create-issue with bare * in allowed-labels triggers warning",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["*"]
`,
			expectWarn: true,
		},
		{
			name: "create-discussion with bare * triggers warning",
			safeOutputs: `safe-outputs:
  create-discussion:
    allowed-labels: ["*"]
`,
			expectWarn: true,
		},
		{
			name: "create-pull-request with bare * triggers warning",
			safeOutputs: `safe-outputs:
  create-pull-request:
    allowed-labels: ["*"]
`,
			expectWarn: true,
		},
		{
			name: "merge-pull-request with bare * triggers warning",
			safeOutputs: `safe-outputs:
  merge-pull-request:
    allowed-labels: ["*"]
`,
			expectWarn: true,
		},
		{
			name: "update-discussion with bare * triggers warning",
			safeOutputs: `safe-outputs:
  update-discussion:
    allowed-labels: ["*"]
`,
			expectWarn: true,
		},
		{
			name: "specific label names do not trigger warning",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["bug", "enhancement"]
`,
			expectWarn: false,
		},
		{
			name: "prefix glob pattern does not trigger warning",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["team-*", "priority-*"]
`,
			expectWarn: false,
		},
		{
			name: "mixed specific and bare * triggers warning",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["bug", "*", "enhancement"]
`,
			expectWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "ctr015-test")

			content := "---\n" + basePermissions + tt.safeOutputs + "---\n\n# Test Workflow\n\nTest body.\n"
			wfPath := filepath.Join(tmpDir, "test.md")
			err := os.WriteFile(wfPath, []byte(content), 0o600)
			require.NoError(t, err, "Should write test workflow file")

			compiler := NewCompiler()
			_, _ = compiler.Compile(wfPath)

			if tt.expectWarn {
				assert.Greater(t, compiler.GetWarningCount(), 0,
					"CTR-015: expected warning for bare \"*\" in allowed-labels")
			}
		})
	}
}

