//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileWorkflows_ShellScriptResourcesIncludeOperationalValueGrader is an
// end-to-end (frontmatter -> compiled WorkflowData) check that a workflow's
// graders.operational-value.run evaluator script, and mcp-scripts.*.run
// scripts, are surfaced through WorkflowData.ShellScriptResources() so the
// compile pipeline's ShellCheck batch actually covers them. See gh-aw#56150.
func TestCompileWorkflows_ShellScriptResourcesIncludeOperationalValueGrader(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, initTestGitRepo(tmpDir))

	gradersDir := filepath.Join(tmpDir, ".github", "workflows", "graders")
	require.NoError(t, os.MkdirAll(gradersDir, 0o755))

	evaluatorScript := "#!/usr/bin/env bash\nset -euo pipefail\necho '{}'\n"
	evaluatorPath := filepath.Join(gradersDir, "example-operational-value.sh")
	require.NoError(t, os.WriteFile(evaluatorPath, []byte(evaluatorScript), 0o755))

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))

	workflowContent := `---
name: Test Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
strict: false
mcp-scripts:
  inspect:
    description: Inspect repository state
    run: |
      git status --short
graders:
  operational-value:
    run: ./graders/example-operational-value.sh
---

# Test Workflow

This is a test workflow exercising frontmatter shell script resources.
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "test.md"), []byte(workflowContent), 0o644))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	config := CompileConfig{
		MarkdownFiles: []string{"test"},
		NoEmit:        true,
	}

	workflowDataList, err := CompileWorkflows(context.Background(), config)
	require.NoError(t, err)
	require.Len(t, workflowDataList, 1)

	resources := workflowDataList[0].ShellScriptResources()

	var gradersFound, mcpScriptsFound bool
	for _, resource := range resources {
		switch resource.Name {
		case "graders.operational-value":
			gradersFound = true
			assert.Equal(t, evaluatorScript, resource.Script)
			assert.Equal(t, "./graders/example-operational-value.sh", resource.Source)
			assert.Equal(t, "bash", resource.Shell)
		case "mcp-scripts.inspect":
			mcpScriptsFound = true
			assert.Equal(t, "git status --short\n", resource.Script)
			assert.Equal(t, "test", resource.Source)
			assert.Equal(t, "bash", resource.Shell)
		}
	}

	assert.True(t, gradersFound, "expected graders.operational-value to be present in ShellScriptResources, got %+v", resources)
	assert.True(t, mcpScriptsFound, "expected mcp-scripts.inspect to be present in ShellScriptResources, got %+v", resources)
}
