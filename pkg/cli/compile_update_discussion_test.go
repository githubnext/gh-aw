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

// TestCompileUpdateDiscussionFieldEnforcement verifies that field-level filtering
// (filterToolSchemaFields) is correctly reflected in the compiled lock file when
// update-discussion is configured with title and labels but not body.
//
// Expected handler config after compilation:
//   - allow_title: true   (title: is configured)
//   - allow_labels: true  (labels: is configured)
//   - allow_body absent   (body: is NOT configured — blocked by filterToolSchemaFields and runtime)
//   - allowed_labels: ["smoke-test","general"]
func TestCompileUpdateDiscussionFieldEnforcement(t *testing.T) {
	const workflowContent = `---
on:
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  pull-requests: read
engine: copilot
safe-outputs:
  update-discussion:
    max: 4
    target: "*"
    title:
    labels:
    allowed-labels: ["smoke-test", "general"]
timeout-minutes: 10
---

# Test: update-discussion field-level enforcement

Verifies that filterToolSchemaFields correctly restricts which fields
the agent can modify when using update-discussion.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-update-discussion-field-enforcement.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write workflow file")

	compiler := workflow.NewCompiler()
	err = CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false)
	require.NoError(t, err, "Expected compilation to succeed")

	lockFilePath := filepath.Join(tmpDir, "test-update-discussion-field-enforcement.lock.yml")
	lockBytes, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read compiled lock file")
	lockContent := string(lockBytes)

	// allow_title must be present (title: is configured in the workflow)
	assert.Contains(t, lockContent, `"allow_title":true`,
		"Lock file should contain allow_title:true in handler config")

	// allow_labels must be present (labels: is configured in the workflow)
	assert.Contains(t, lockContent, `"allow_labels":true`,
		"Lock file should contain allow_labels:true in handler config")

	// allow_body must be absent (body: is NOT configured — filtered by filterToolSchemaFields)
	assert.NotContains(t, lockContent, `"allow_body"`,
		"Lock file must NOT contain allow_body since body updates are not configured")

	// allowed_labels must list the configured allowed labels
	assert.Contains(t, lockContent, `"allowed_labels":["smoke-test","general"]`,
		`Lock file should contain allowed_labels:["smoke-test","general"] in handler config`)

	// handler config must be embedded in the lock file
	assert.Contains(t, lockContent, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
		"Lock file should contain GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG")
}
