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

// TestCompileWorkflow_InlineSubAgents verifies that inline sub-agent definitions
// are extracted from the markdown body and written as separate files under
// .github/agents/ during compilation.
func TestCompileWorkflow_InlineSubAgents(t *testing.T) {
	// Create a temporary directory structure that mimics a repository layout:
	//   <tmp>/
	//     .github/
	//       workflows/
	//         my-workflow.md
	//       agents/         ← sub-agent files should land here
	tmpDir := testutil.TempDir(t, "inline-sub-agents")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "create workflows dir")

	workflowContent := `---
engine: copilot
on:
  issues:
    types: [opened]
---
# Handle issue

Triage the issue.

## agent: planner
---
engine: copilot
tools:
  github:
    toolsets: [default]
---
You are a planning assistant.

## agent: executor
---
engine: copilot
tools:
  github:
    toolsets: [default]
---
You are an execution specialist.
`

	workflowPath := filepath.Join(workflowsDir, "my-workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "write workflow")

	compiler := NewCompiler()
	// Use gitRoot so resolveAgentsDir does not rely on the ../.. heuristic.
	compiler.gitRoot = tmpDir

	err := compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "compilation should succeed with inline sub-agents")

	agentsDir := filepath.Join(tmpDir, ".github", "agents")

	// Verify planner agent file
	plannerPath := filepath.Join(agentsDir, "planner.md")
	plannerContent, err := os.ReadFile(plannerPath)
	require.NoError(t, err, "planner.md should exist in .github/agents/")
	assert.Contains(t, string(plannerContent), "You are a planning assistant.", "planner.md should contain the agent prompt")
	assert.Contains(t, string(plannerContent), "engine: copilot", "planner.md should contain frontmatter")

	// Verify executor agent file
	executorPath := filepath.Join(agentsDir, "executor.md")
	executorContent, err := os.ReadFile(executorPath)
	require.NoError(t, err, "executor.md should exist in .github/agents/")
	assert.Contains(t, string(executorContent), "You are an execution specialist.", "executor.md should contain the agent prompt")

	// Verify the main workflow prompt does NOT contain the sub-agent sections
	lockPath := workflowPath[:len(workflowPath)-len(".md")] + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "lock file should be generated")
	assert.NotContains(t, string(lockContent), "## agent:", "lock file should not expose separator syntax")
}

// TestCompileWorkflow_InlineSubAgents_NoEmit verifies that sub-agent files are NOT
// written when noEmit is enabled (validation-only mode).
func TestCompileWorkflow_InlineSubAgents_NoEmit(t *testing.T) {
	tmpDir := testutil.TempDir(t, "inline-sub-agents-noemit")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "create workflows dir")

	workflowContent := `---
engine: copilot
on:
  issues:
    types: [opened]
---
# Workflow

Main content.

## agent: helper
---
engine: copilot
---
You are a helper.
`

	workflowPath := filepath.Join(workflowsDir, "noemit-workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "write workflow")

	compiler := NewCompiler(WithNoEmit(true))
	compiler.gitRoot = tmpDir

	err := compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "no-emit compilation should succeed")

	// Neither the lock file nor agent files should exist
	agentsDir := filepath.Join(tmpDir, ".github", "agents")
	_, err = os.Stat(filepath.Join(agentsDir, "helper.md"))
	assert.True(t, os.IsNotExist(err), "agent file should NOT be written in no-emit mode")
}

// TestCompileWorkflow_InlineSubAgents_MainContentPreserved verifies that the main
// workflow content (before the first sub-agent separator) is used for the compiled
// prompt and does not include sub-agent sections.
func TestCompileWorkflow_InlineSubAgents_MainContentPreserved(t *testing.T) {
	tmpDir := testutil.TempDir(t, "inline-sub-agents-content")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "create workflows dir")

	workflowContent := `---
engine: copilot
on:
  issues:
    types: [opened]
---
# My Workflow

Main workflow prompt.

## agent: sidecar
---
engine: copilot
---
Sub-agent prompt content.
`

	workflowPath := filepath.Join(workflowsDir, "content-test.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "write workflow")

	compiler := NewCompiler()
	compiler.gitRoot = tmpDir

	err := compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "compilation should succeed")

	// The lock file uses {{#runtime-import}} for the prompt, so actual content is not
	// inlined at compile time. Instead, verify the separator syntax does NOT appear in
	// the compiled YAML and that the runtime-import macro points to the workflow file.
	lockPath := workflowPath[:len(workflowPath)-len(".md")] + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "lock file should exist")

	lockStr := string(lockContent)
	assert.NotContains(t, lockStr, "## agent:", "separator syntax must not be hardcoded in lock file")

	// The lock file should reference the workflow file via runtime-import
	assert.Contains(t, lockStr, "content-test.md", "lock file should reference the workflow file")

	// The sidecar agent file should contain the sub-agent content
	agentPath := filepath.Join(tmpDir, ".github", "agents", "sidecar.md")
	agentContent, err := os.ReadFile(agentPath)
	require.NoError(t, err, "sidecar.md should be written")
	assert.Contains(t, string(agentContent), "Sub-agent prompt content.", "agent file should contain sub-agent prompt")
}
