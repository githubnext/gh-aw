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

// TestAssignToAgentCanonicalNameKey tests that 'name' is the canonical key for assigning an agent
func TestAssignToAgentCanonicalNameKey(t *testing.T) {
	tmpDir := testutil.TempDir(t, "assign-to-agent-name-test")

	workflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  assign-to-agent:
    name: copilot
---

# Test Workflow

This workflow tests canonical 'name' key.
`
	testFile := filepath.Join(tmpDir, "test-assign-to-agent.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test workflow")

	compiler := NewCompilerWithVersion("1.0.0")
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse workflow")

	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.AssignToAgent, "AssignToAgent should not be nil")
	assert.Equal(t, "copilot", workflowData.SafeOutputs.AssignToAgent.DefaultAgent, "Should parse 'name' key as DefaultAgent")
}

// TestAssignToAgentStepHasContinueOnError verifies that the assign_to_agent step has
// continue-on-error: true so that assignment failures propagate as outputs to the
// conclusion job without blocking other safe outputs from being processed.
func TestAssignToAgentStepHasContinueOnError(t *testing.T) {
	tmpDir := testutil.TempDir(t, "assign-to-agent-continue-on-error")

	workflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  assign-to-agent:
    name: copilot
---

# Test Workflow

This workflow tests that the assign_to_agent step has continue-on-error: true.
`
	testFile := filepath.Join(tmpDir, "test-assign-to-agent-coe.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := filepath.Join(tmpDir, "test-assign-to-agent-coe.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "id: assign_to_agent",
		"Expected assign_to_agent step in generated workflow")
	assert.Contains(t, lockContent, "continue-on-error: true",
		"Expected assign_to_agent step to have continue-on-error: true so failures propagate as outputs to the conclusion job")
	assert.Contains(t, lockContent, "assign_to_agent_assignment_error_count",
		"Expected safe_outputs job to export assign_to_agent_assignment_error_count output for failure propagation")
}
