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
	// The parse-guard-vars step is injected to parse variables into JSON arrays at runtime.
	assert.Contains(t, lockFileContent, `id: parse-guard-vars`, "Compiled lock file must include parse-guard-vars step")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.blocked_users`, "Compiled lock file must reference blocked_users step output")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.trusted_users`, "Compiled lock file must reference trusted_users step output")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.approval_labels`, "Compiled lock file must reference approval_labels step output")
	// The step must include the fallback variable env vars.
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_VAR`, "Compiled lock file must pass GH_AW_BLOCKED_USERS_VAR to parse step")
	assert.Contains(t, lockFileContent, `GH_AW_TRUSTED_USERS_VAR`, "Compiled lock file must pass GH_AW_TRUSTED_USERS_VAR to parse step")
	assert.Contains(t, lockFileContent, `GH_AW_APPROVAL_LABELS_VAR`, "Compiled lock file must pass GH_AW_APPROVAL_LABELS_VAR to parse step")
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
	// The parse-guard-vars step receives static values via GH_AW_BLOCKED_USERS_EXTRA and
	// GH_AW_APPROVAL_LABELS_EXTRA at compile time, and parses the GH_AW_GITHUB_* fallback
	// variables at runtime to produce proper JSON arrays.
	assert.Contains(t, lockFileContent, `id: parse-guard-vars`, "Compiled lock file must include parse-guard-vars step")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_EXTRA: spam-bot,compromised-user`, "Compiled lock file must include static blocked-users in step env")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_VAR`, "Compiled lock file must include GH_AW_BLOCKED_USERS_VAR in step env")
	assert.Contains(t, lockFileContent, `GH_AW_APPROVAL_LABELS_EXTRA: human-reviewed,safe-for-agent`, "Compiled lock file must include static approval-labels in step env")
	assert.Contains(t, lockFileContent, `GH_AW_APPROVAL_LABELS_VAR`, "Compiled lock file must include GH_AW_APPROVAL_LABELS_VAR in step env")
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.blocked_users`, "Compiled lock file must reference blocked_users step output")
	assert.Contains(t, lockFileContent, `"approval-labels"`, "Compiled lock file must include approval-labels in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.approval_labels`, "Compiled lock file must reference approval_labels step output")
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
	// The parse-guard-vars step receives user-provided expressions via GH_AW_BLOCKED_USERS_EXTRA
	// and GH_AW_APPROVAL_LABELS_EXTRA; GitHub Actions evaluates the expressions at runtime.
	assert.Contains(t, lockFileContent, `id: parse-guard-vars`, "Compiled lock file must include parse-guard-vars step")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_EXTRA: ${{ vars.BLOCKED_USERS }}`, "Compiled lock file must pass user expression to blocked_users extra")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_VAR`, "Compiled lock file must include GH_AW_BLOCKED_USERS_VAR in step env")
	assert.Contains(t, lockFileContent, `GH_AW_APPROVAL_LABELS_EXTRA: ${{ vars.APPROVAL_LABELS }}`, "Compiled lock file must pass user expression to approval_labels extra")
	assert.Contains(t, lockFileContent, `GH_AW_APPROVAL_LABELS_VAR`, "Compiled lock file must include GH_AW_APPROVAL_LABELS_VAR in step env")
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.blocked_users`, "Compiled lock file must reference blocked_users step output")
	assert.Contains(t, lockFileContent, `"approval-labels"`, "Compiled lock file must include approval-labels")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.approval_labels`, "Compiled lock file must reference approval_labels step output")
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
	// Static comma-separated values are passed to the parse step via GH_AW_BLOCKED_USERS_EXTRA
	// at compile time; the step parses them at runtime into a JSON array.
	assert.Contains(t, lockFileContent, `id: parse-guard-vars`, "Compiled lock file must include parse-guard-vars step")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_EXTRA: spam-bot,compromised-user`, "Compiled lock file must include parsed static items in step env")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_VAR`, "Compiled lock file must include GH_AW_BLOCKED_USERS_VAR in step env")
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.blocked_users`, "Compiled lock file must reference blocked_users step output")
}

// TestGuardPolicyTrustedUsersCompiledOutput verifies that trusted-users is written into the
// compiled guard-policies allow-only block and that the parse-guard-vars step receives the
// static values via GH_AW_TRUSTED_USERS_EXTRA.
func TestGuardPolicyTrustedUsersCompiledOutput(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    min-integrity: approved
    trusted-users:
      - contractor-1
      - partner-dev
    blocked-users:
      - bad-actor
---

# Guard Policy Test

This workflow uses trusted-users alongside blocked-users.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy-trusted.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	lockFilePath := filepath.Join(tmpDir, "test-guard-policy-trusted.lock.yml")
	lockFileBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")

	lockFileContent := string(lockFileBytes)
	// The parse-guard-vars step receives static trusted-users values via GH_AW_TRUSTED_USERS_EXTRA.
	assert.Contains(t, lockFileContent, `id: parse-guard-vars`, "Compiled lock file must include parse-guard-vars step")
	assert.Contains(t, lockFileContent, `GH_AW_TRUSTED_USERS_EXTRA: contractor-1,partner-dev`, "Compiled lock file must include static trusted-users in step env")
	assert.Contains(t, lockFileContent, `GH_AW_TRUSTED_USERS_VAR`, "Compiled lock file must include GH_AW_TRUSTED_USERS_VAR in step env")
	assert.Contains(t, lockFileContent, `GH_AW_BLOCKED_USERS_EXTRA: bad-actor`, "Compiled lock file must include static blocked-users in step env")
	assert.Contains(t, lockFileContent, `"trusted-users"`, "Compiled lock file must include trusted-users in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.trusted_users`, "Compiled lock file must reference trusted_users step output")
	assert.Contains(t, lockFileContent, `"blocked-users"`, "Compiled lock file must include blocked-users in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.blocked_users`, "Compiled lock file must reference blocked_users step output")
}

// TestGuardPolicyTrustedUsersExpressionCompiledOutput verifies that a trusted-users GitHub
// Actions expression is passed through as a string in the compiled guard-policies block.
func TestGuardPolicyTrustedUsersExpressionCompiledOutput(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    min-integrity: approved
    trusted-users: "${{ vars.TRUSTED_USERS }}"
---

# Guard Policy Test

This workflow passes trusted-users as a GitHub Actions expression.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy-trusted-expr.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	lockFilePath := filepath.Join(tmpDir, "test-guard-policy-trusted-expr.lock.yml")
	lockFileBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")

	lockFileContent := string(lockFileBytes)
	assert.Contains(t, lockFileContent, `id: parse-guard-vars`, "Compiled lock file must include parse-guard-vars step")
	assert.Contains(t, lockFileContent, `GH_AW_TRUSTED_USERS_EXTRA: ${{ vars.TRUSTED_USERS }}`, "Compiled lock file must pass user expression to trusted_users extra")
	assert.Contains(t, lockFileContent, `GH_AW_TRUSTED_USERS_VAR`, "Compiled lock file must include GH_AW_TRUSTED_USERS_VAR in step env")
	assert.Contains(t, lockFileContent, `"trusted-users"`, "Compiled lock file must include trusted-users in the guard-policies allow-only block")
	assert.Contains(t, lockFileContent, `steps.parse-guard-vars.outputs.trusted_users`, "Compiled lock file must reference trusted_users step output")
}

// TestGuardPolicyTrustedUsersRequiresMinIntegrity verifies that trusted-users cannot be set
// without a min-integrity guard policy.
func TestGuardPolicyTrustedUsersRequiresMinIntegrity(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  github:
    trusted-users:
      - contractor-1
---

# Guard Policy Test

This workflow sets trusted-users without min-integrity (should fail).
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-guard-policy-trusted-no-integrity.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.Error(t, err, "Expected compilation to fail without min-integrity")
	assert.Contains(t, err.Error(), "min-integrity", "Error should mention min-integrity requirement")
}
