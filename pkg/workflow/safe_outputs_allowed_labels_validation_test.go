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

// TestCTR015AllowedLabelsGlobScope tests that the compiler rejects (CTR-015) when
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
		expectError bool
	}{
		{
			name: "create-issue with bare * in allowed-labels triggers error",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["*"]
`,
			expectError: true,
		},
		{
			name: "create-discussion with bare * triggers error",
			safeOutputs: `safe-outputs:
  create-discussion:
    allowed-labels: ["*"]
`,
			expectError: true,
		},
		{
			name: "create-pull-request with bare * triggers error",
			safeOutputs: `safe-outputs:
  create-pull-request:
    allowed-labels: ["*"]
`,
			expectError: true,
		},
		{
			name: "merge-pull-request with bare * triggers error",
			safeOutputs: `safe-outputs:
  merge-pull-request:
    allowed-labels: ["*"]
`,
			expectError: true,
		},
		{
			name: "update-discussion with bare * triggers error",
			safeOutputs: `safe-outputs:
  update-discussion:
    allowed-labels: ["*"]
`,
			expectError: true,
		},
		{
			name: "specific label names do not trigger error",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["bug", "enhancement"]
`,
			expectError: false,
		},
		{
			name: "prefix glob pattern does not trigger error",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["team-*", "priority-*"]
`,
			expectError: false,
		},
		{
			name: "mixed specific and bare * triggers error",
			safeOutputs: `safe-outputs:
  create-issue:
    allowed-labels: ["bug", "*", "enhancement"]
`,
			expectError: true,
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
			compileErr := compiler.CompileWorkflow(wfPath)

			if tt.expectError {
				assert.Error(t, compileErr,
					"CTR-015: expected error for bare \"*\" in allowed-labels")
				assert.Contains(t, compileErr.Error(), "CTR-015",
					"CTR-015: error message should reference the rule ID")
			} else {
				assert.NoError(t, compileErr,
					"CTR-015: did not expect error for valid allowed-labels")
			}
		})
	}
}

