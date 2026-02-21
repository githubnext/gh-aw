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

// TestAssignToAgentDeprecatedDefaultAgentKey tests that 'default-agent' is accepted as a deprecated alias
func TestAssignToAgentDeprecatedDefaultAgentKey(t *testing.T) {
	tmpDir := testutil.TempDir(t, "assign-to-agent-deprecated-test")

	workflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  assign-to-agent:
    default-agent: copilot
---

# Test Workflow

This workflow tests deprecated 'default-agent' key.
`
	testFile := filepath.Join(tmpDir, "test-assign-to-agent.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test workflow")

	compiler := NewCompilerWithVersion("1.0.0")
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse workflow")

	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.AssignToAgent, "AssignToAgent should not be nil")
	assert.Equal(t, "copilot", workflowData.SafeOutputs.AssignToAgent.DefaultAgent, "Deprecated 'default-agent' key should be accepted as alias for 'name'")
}

// TestAssignToAgentNameTakesPrecedenceOverDefaultAgent tests that 'name' takes precedence over 'default-agent'
func TestAssignToAgentNameTakesPrecedenceOverDefaultAgent(t *testing.T) {
	tmpDir := testutil.TempDir(t, "assign-to-agent-precedence-test")

	workflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  assign-to-agent:
    name: copilot
    default-agent: other-agent
---

# Test Workflow

This workflow tests that 'name' takes precedence over 'default-agent'.
`
	testFile := filepath.Join(tmpDir, "test-assign-to-agent.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test workflow")

	compiler := NewCompilerWithVersion("1.0.0")
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse workflow")

	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.AssignToAgent, "AssignToAgent should not be nil")
	assert.Equal(t, "copilot", workflowData.SafeOutputs.AssignToAgent.DefaultAgent, "Canonical 'name' key should take precedence over deprecated 'default-agent'")
}
