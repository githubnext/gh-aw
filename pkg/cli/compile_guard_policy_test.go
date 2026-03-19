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
