//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGuardPolicyMinIntegrityOnly verifies that specifying only min-integrity
// under tools.github compiles successfully without requiring an explicit repos field.
// When repos is omitted, it should default to "all" (regression test for the fix).
func TestGuardPolicyMinIntegrityOnly(t *testing.T) {
	tests := []struct {
		name            string
		workflowContent string
		expectError     bool
		errorContains   string
	}{
		{
			name: "min-integrity only defaults repos to all",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    min-integrity: none
---

# Guard Policy Test

This workflow uses min-integrity without specifying repos.
`,
			expectError: false,
		},
		{
			name: "min-integrity with explicit repos=all compiles",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    repos: all
    min-integrity: unapproved
---

# Guard Policy Test

This workflow uses both repos and min-integrity.
`,
			expectError: false,
		},
		{
			name: "min-integrity with repos=public compiles",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    repos: public
    min-integrity: approved
---

# Guard Policy Test

This workflow restricts to public repos.
`,
			expectError: false,
		},
		{
			name: "min-integrity with repos array compiles",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    repos:
      - owner/repo
    min-integrity: merged
---

# Guard Policy Test

This workflow specifies a repos array.
`,
			expectError: false,
		},
		{
			name: "repos only without min-integrity fails validation",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    repos: all
---

# Guard Policy Test

This workflow specifies repos without min-integrity.
`,
			expectError:   true,
			errorContains: "min-integrity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			workflowPath := filepath.Join(tmpDir, "test-guard-policy.md")
			err := os.WriteFile(workflowPath, []byte(tt.workflowContent), 0644)
			require.NoError(t, err, "Failed to write workflow file")

			compiler := workflow.NewCompiler()
			err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)

			if tt.expectError {
				require.Error(t, err, "Expected compilation to fail")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should mention %q", tt.errorContains)
				}
			} else {
				assert.NoError(t, err, "Expected compilation to succeed")
			}
		})
	}
}

// TestGuardPolicyMinIntegrityOnlyCompiledOutput verifies that when only min-integrity is
// specified (without repos), the compiled lock file includes repos="all" in the guard policy.
// This is a regression test for the MCP Gateway requirement that allow-only must include repos.
func TestGuardPolicyMinIntegrityOnlyCompiledOutput(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    min-integrity: approved
---

# Guard Policy Test

This workflow uses min-integrity without specifying repos.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	// Read the compiled lock file and verify it contains the correct guard-policies JSON block.
	// The MCP Gateway requires repos to be present in the allow-only policy.
	lockFilePath := filepath.Join(tmpDir, "test-guard-policy.lock.yml")
	lockFileBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")

	lockFileContent := string(lockFileBytes)
	// Check that the guard-policies allow-only block contains both repos=all and min-integrity=approved
	// in the correct JSON structure expected by the MCP Gateway.
	assert.Contains(t, lockFileContent, `"guard-policies": {`+"\n"+`                  "allow-only": {`+"\n"+`                    "min-integrity": "approved",`+"\n"+`                    "repos": "all"`,
		"Compiled lock file must include repos=all and min-integrity=approved in the guard-policies allow-only block")
}
