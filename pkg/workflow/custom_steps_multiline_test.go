package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomStepsMultilineRunFields tests that custom steps with multiline run fields
// are serialized using YAML's literal block scalar format (|) instead of escaped newlines
func TestCustomStepsMultilineRunFields(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test workflow with multiline run field in custom steps
	workflowContent := `---
engine: copilot
on:
  workflow_dispatch:
permissions:
  contents: read
steps:
  - name: Multi-line shell script
    id: test-script
    run: |
      set -e
      
      # This is a comment
      echo "Starting test"
      mkdir -p /tmp/test-dir
      cd /tmp/test-dir
      
      # Another comment
      echo "Test complete"
  - name: Another step
    run: echo "single line"
---

# Test Workflow

This workflow tests multiline run field serialization.
`

	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to create test workflow file")

	// Compile the workflow
	compiler := NewCompiler(false, "", tempDir)
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file
	lockPath := filepath.Join(tempDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify that the multiline run field uses literal block scalar format
	assert.Contains(t, lockStr, "name: Multi-line shell script", "Lock file should contain the step name")
	assert.Contains(t, lockStr, "run: |", "Lock file should use literal block scalar for multiline run")

	// Verify that the script content is properly formatted with indentation
	assert.Contains(t, lockStr, "  set -e", "Lock file should have properly indented script content")
	assert.Contains(t, lockStr, "  # This is a comment", "Lock file should preserve comments with proper indentation")
	assert.Contains(t, lockStr, "  echo \"Starting test\"", "Lock file should preserve commands with proper indentation")
	assert.Contains(t, lockStr, "  mkdir -p /tmp/test-dir", "Lock file should preserve mkdir command with proper indentation")

	// Verify that newlines are NOT escaped
	assert.NotContains(t, lockStr, "\\n", "Lock file should NOT contain escaped newlines")
	assert.NotContains(t, lockStr, "run: \"set -e\\n", "Lock file should NOT have quoted string with escaped newlines")

	// Verify single-line run command remains simple
	assert.Contains(t, lockStr, "name: Another step", "Lock file should contain second step")
	// Single line commands may or may not use block scalar depending on YAML library behavior
	// Just verify it doesn't have escaped newlines
	singleLineStepStart := strings.Index(lockStr, "name: Another step")
	require.Greater(t, singleLineStepStart, 0, "Should find 'Another step'")
	nextStepOrEnd := len(lockStr)
	if nextIdx := strings.Index(lockStr[singleLineStepStart+100:], "- name:"); nextIdx > 0 {
		nextStepOrEnd = singleLineStepStart + 100 + nextIdx
	}
	singleLineSection := lockStr[singleLineStepStart:nextStepOrEnd]
	assert.NotContains(t, singleLineSection, "\\n", "Single line run should not have escaped newlines")
}

// TestCustomStepsWithImportedStepsMultiline tests that imported steps with multiline
// run fields are also properly serialized
func TestCustomStepsWithImportedStepsMultiline(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a shared workflow with multiline steps
	sharedContent := `---
steps:
  - name: Imported multiline script
    run: |
      echo "This is from an imported step"
      cd /tmp
      ls -la
---
`

	sharedPath := filepath.Join(tempDir, "shared-steps.md")
	err := os.WriteFile(sharedPath, []byte(sharedContent), 0644)
	require.NoError(t, err, "Failed to create shared workflow file")

	// Create a main workflow that imports the shared steps
	workflowContent := `---
engine: copilot
on:
  workflow_dispatch:
permissions:
  contents: read
imports:
  - shared-steps.md
steps:
  - name: Main workflow step
    run: |
      echo "Main workflow"
      pwd
---

# Main Workflow

This workflow imports steps and adds its own.
`

	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to create main workflow file")

	// Compile the workflow
	compiler := NewCompiler(false, "", tempDir)
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file
	lockPath := filepath.Join(tempDir, "main-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify that both imported and main steps use literal block scalar format
	assert.Contains(t, lockStr, "name: Imported multiline script", "Lock file should contain imported step")
	assert.Contains(t, lockStr, "name: Main workflow step", "Lock file should contain main step")

	// Count occurrences of "run: |" to ensure both steps use block scalar
	blockScalarCount := strings.Count(lockStr, "run: |")
	assert.GreaterOrEqual(t, blockScalarCount, 2, "Lock file should have at least 2 multiline run fields with block scalar")

	// Verify no escaped newlines anywhere in the file
	assert.NotContains(t, lockStr, "\\n", "Lock file should NOT contain any escaped newlines")
}

// TestPostStepsMultilineRunFields tests that post-steps with multiline run fields
// are also properly serialized
func TestPostStepsMultilineRunFields(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test workflow with multiline run field in post-steps
	workflowContent := `---
engine: copilot
on:
  workflow_dispatch:
permissions:
  contents: read
post-steps:
  - name: Cleanup script
    run: |
      echo "Cleaning up"
      rm -rf /tmp/test-dir
      echo "Cleanup complete"
---

# Test Workflow

This workflow tests multiline post-steps serialization.
`

	workflowPath := filepath.Join(tempDir, "test-poststeps.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to create test workflow file")

	// Compile the workflow
	compiler := NewCompiler(false, "", tempDir)
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file
	lockPath := filepath.Join(tempDir, "test-poststeps.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify that the post-step multiline run field uses literal block scalar format
	assert.Contains(t, lockStr, "name: Cleanup script", "Lock file should contain the post-step name")
	assert.Contains(t, lockStr, "run: |", "Lock file should use literal block scalar for multiline post-step run")
	assert.Contains(t, lockStr, "  echo \"Cleaning up\"", "Lock file should have properly indented post-step content")
	assert.NotContains(t, lockStr, "\\n", "Lock file should NOT contain escaped newlines in post-steps")
}
