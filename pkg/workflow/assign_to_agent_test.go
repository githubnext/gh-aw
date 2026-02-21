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
