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
	// Check that the guard-policies allow-only block contains the required fields.
	// The MCP Gateway requires repos to be present in the allow-only policy.
	assert.Contains(t, lockFileContent, `"guard-policies"`, "Compiled lock file must include guard-policies block")
	assert.Contains(t, lockFileContent, `"allow-only"`, "Compiled lock file must include allow-only policy")
	assert.Contains(t, lockFileContent, `"min-integrity": "approved"`, "Compiled lock file must include min-integrity=approved")
	assert.Contains(t, lockFileContent, `"repos": "all"`, "Compiled lock file must default repos to 'all'")
	// Fallback expressions for blocked-users and approval-labels should be injected automatically
	// using toJSON() to ensure proper JSON encoding at runtime.
	assert.Contains(t, lockFileContent, `"blocked-users": ${{ toJSON(vars.GH_AW_GITHUB_BLOCKED_USERS || '') }}`,
		"Compiled lock file must inject blocked-users toJSON fallback expression")
	assert.Contains(t, lockFileContent, `"approval-labels": ${{ toJSON(vars.GH_AW_GITHUB_APPROVAL_LABELS || '') }}`,
		"Compiled lock file must inject approval-labels toJSON fallback expression")
}

// TestGuardPolicyBlockedUsersApprovalLabelsCompiledOutput verifies that blocked-users and
// approval-labels are written into the compiled guard-policies allow-only block.
func TestGuardPolicyBlockedUsersApprovalLabelsCompiledOutput(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    allowed-repos:
      - myorg/myrepo
    min-integrity: approved
    blocked-users:
      - spam-bot
      - compromised-user
    approval-labels:
      - human-reviewed
      - safe-for-agent
---

# Guard Policy Test

This workflow uses blocked-users and approval-labels.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy-blocked.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	lockFilePath := filepath.Join(tmpDir, "test-guard-policy-blocked.lock.yml")
	lockFileBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")

	lockFileContent := string(lockFileBytes)
	// With union semantics, blocked-users and approval-labels are rendered as a JSON array
	// containing the explicit values and a toJSON() fallback expression as the last element.
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `spam-bot`, "Compiled lock file must include spam-bot in blocked-users")
	assert.Contains(t, lockFileContent, `compromised-user`, "Compiled lock file must include compromised-user in blocked-users")
	assert.Contains(t, lockFileContent, `toJSON(vars.GH_AW_GITHUB_BLOCKED_USERS`, "Compiled lock file must include blocked-users toJSON fallback")
	assert.Contains(t, lockFileContent, `"approval-labels"`, "Compiled lock file must include approval-labels in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `human-reviewed`, "Compiled lock file must include human-reviewed in approval-labels")
	assert.Contains(t, lockFileContent, `safe-for-agent`, "Compiled lock file must include safe-for-agent in approval-labels")
	assert.Contains(t, lockFileContent, `toJSON(vars.GH_AW_GITHUB_APPROVAL_LABELS`, "Compiled lock file must include approval-labels toJSON fallback")
}

// TestGuardPolicyBlockedUsersExpressionCompiledOutput verifies that blocked-users as a GitHub
// Actions expression is passed through as a string in the compiled guard-policies block.
func TestGuardPolicyBlockedUsersExpressionCompiledOutput(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    allowed-repos: all
    min-integrity: unapproved
    blocked-users: "${{ vars.BLOCKED_USERS }}"
    approval-labels: "${{ vars.APPROVAL_LABELS }}"
---

# Guard Policy Test

This workflow passes blocked-users and approval-labels as expressions.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy-expr.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	lockFilePath := filepath.Join(tmpDir, "test-guard-policy-expr.lock.yml")
	lockFileBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")

	lockFileContent := string(lockFileBytes)
	// Expressions should be unioned with the GH_AW_GITHUB_* fallback in an array.
	// The fallback uses toJSON() for proper JSON encoding.
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users")
	assert.Contains(t, lockFileContent, `vars.BLOCKED_USERS`, "Compiled lock file must preserve the blocked-users expression")
	assert.Contains(t, lockFileContent, `toJSON(vars.GH_AW_GITHUB_BLOCKED_USERS`, "Compiled lock file must union blocked-users with toJSON fallback")
	assert.Contains(t, lockFileContent, `"approval-labels"`, "Compiled lock file must include approval-labels")
	assert.Contains(t, lockFileContent, `vars.APPROVAL_LABELS`, "Compiled lock file must preserve the approval-labels expression")
	assert.Contains(t, lockFileContent, `toJSON(vars.GH_AW_GITHUB_APPROVAL_LABELS`, "Compiled lock file must union approval-labels with toJSON fallback")
}

// TestGuardPolicyBlockedUsersCommaSeparatedCompiledOutput verifies that a static
// comma-separated blocked-users string is split at compile time.
func TestGuardPolicyBlockedUsersCommaSeparatedCompiledOutput(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    allowed-repos: all
    min-integrity: unapproved
    blocked-users: "spam-bot, compromised-user"
---

# Guard Policy Test

This workflow passes blocked-users as a comma-separated string.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy-csv.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	lockFilePath := filepath.Join(tmpDir, "test-guard-policy-csv.lock.yml")
	lockFileBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")

	lockFileContent := string(lockFileBytes)
	// With union semantics, static comma-separated strings are split and then combined with
	// the GH_AW_GITHUB_BLOCKED_USERS toJSON fallback into a JSON array.
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users")
	assert.Contains(t, lockFileContent, `spam-bot`, "Compiled lock file must split spam-bot from comma-separated string")
	assert.Contains(t, lockFileContent, `compromised-user`, "Compiled lock file must split compromised-user from comma-separated string")
	assert.Contains(t, lockFileContent, `toJSON(vars.GH_AW_GITHUB_BLOCKED_USERS`, "Compiled lock file must union with blocked-users toJSON fallback")
}
