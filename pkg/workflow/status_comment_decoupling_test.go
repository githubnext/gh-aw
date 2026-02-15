//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCommentDecouplingCompilation(t *testing.T) {
	// Create a temporary directory for test outputs
	tmpDir, err := os.MkdirTemp("", "status-comment-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	// Copy test workflow to temp dir
	testWorkflowPath := filepath.Join(tmpDir, "test-status-comment-decoupling.md")
	testWorkflowContent := `---
name: test-status-comment-decoupling
on:
  reaction: eyes
  status-comment: false
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  create-issue:
    max: 1
---

# Test Status Comment Decoupling

Test workflow with ai-reaction but no status comments.
`
	err = os.WriteFile(testWorkflowPath, []byte(testWorkflowContent), 0644)
	require.NoError(t, err, "Failed to write test workflow")

	// Compile the workflow
	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testWorkflowPath)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file
	lockFilePath := filepath.Join(tmpDir, "test-status-comment-decoupling.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read lock file")
	lockContentStr := string(lockContent)

	// Verify the workflow has reactions
	assert.Contains(t, lockContentStr, "Add eyes reaction for immediate feedback",
		"Should have reaction step in pre-activation job")
	assert.Contains(t, lockContentStr, "GH_AW_REACTION",
		"Should have reaction environment variable")

	// Verify the workflow does NOT have status comment steps
	assert.NotContains(t, lockContentStr, "Add comment with workflow run link",
		"Should NOT have activation comment step")
	assert.NotContains(t, lockContentStr, "Update reaction comment with completion status",
		"Should NOT have conclusion update step")

	// Test reaction without status-comment (breaking change - no automatic bundling)
	testWorkflowPath2 := filepath.Join(tmpDir, "test-reaction-only.md")
	testWorkflowContent2 := `---
name: test-reaction-only
on:
  reaction: eyes
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  create-issue:
    max: 1
---

# Test Reaction Only

Test workflow with ai-reaction but no status-comment field (should NOT create status comments).
`
	err = os.WriteFile(testWorkflowPath2, []byte(testWorkflowContent2), 0644)
	require.NoError(t, err, "Failed to write test workflow 2")

	// Compile the workflow
	err = compiler.CompileWorkflow(testWorkflowPath2)
	require.NoError(t, err, "Failed to compile workflow 2")

	// Read the generated lock file
	lockFilePath2 := filepath.Join(tmpDir, "test-reaction-only.lock.yml")
	lockContent2, err := os.ReadFile(lockFilePath2)
	require.NoError(t, err, "Failed to read lock file 2")
	lockContentStr2 := string(lockContent2)

	// Verify the workflow has reactions but NO status comments (breaking change)
	assert.Contains(t, lockContentStr2, "Add eyes reaction for immediate feedback",
		"Should have reaction step in pre-activation job")
	assert.NotContains(t, lockContentStr2, "Add comment with workflow run link",
		"Should NOT have activation comment step (no automatic bundling)")
	assert.NotContains(t, lockContentStr2, "Update reaction comment with completion status",
		"Should NOT have conclusion update step (no automatic bundling)")

	// Test explicit status-comment: true
	testWorkflowPath3 := filepath.Join(tmpDir, "test-explicit-true.md")
	testWorkflowContent3 := `---
name: test-explicit-true
on:
  reaction: eyes
  status-comment: true
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  create-issue:
    max: 1
---

# Test Explicit True

Test workflow with explicit status-comment: true.
`
	err = os.WriteFile(testWorkflowPath3, []byte(testWorkflowContent3), 0644)
	require.NoError(t, err, "Failed to write test workflow 3")

	// Compile the workflow
	err = compiler.CompileWorkflow(testWorkflowPath3)
	require.NoError(t, err, "Failed to compile workflow 3")

	// Read the generated lock file
	lockFilePath3 := filepath.Join(tmpDir, "test-explicit-true.lock.yml")
	lockContent3, err := os.ReadFile(lockFilePath3)
	require.NoError(t, err, "Failed to read lock file 3")
	lockContentStr3 := string(lockContent3)

	// Verify the workflow has reactions AND status comments
	assert.Contains(t, lockContentStr3, "Add eyes reaction for immediate feedback",
		"Should have reaction step in pre-activation job")
	assert.Contains(t, lockContentStr3, "Add comment with workflow run link",
		"Should have activation comment step when explicit true")
	assert.Contains(t, lockContentStr3, "Update reaction comment with completion status",
		"Should have conclusion update step when explicit true")
}
