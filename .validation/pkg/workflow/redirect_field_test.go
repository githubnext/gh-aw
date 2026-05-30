//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectFieldExtraction(t *testing.T) {
	compiler := NewCompiler()
	redirect := compiler.extractRedirect(map[string]any{"redirect": "owner/repo/workflows/new.md@main"})
	assert.Equal(t, "owner/repo/workflows/new.md@main", redirect, "redirect should be extracted from frontmatter")

	redirect = compiler.extractRedirect(map[string]any{})
	assert.Empty(t, redirect, "missing redirect should return empty string")
}

func TestCompileWorkflow_PrintsRedirectInfoMessage(t *testing.T) {
	tmpDir := testutil.TempDir(t, "redirect-compile-test")
	workflowFile := filepath.Join(tmpDir, "redirected.md")
	workflowContent := `---
redirect: owner/repo/workflows/new-location.md@main
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
---

# Redirected Workflow`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644), "workflow fixture should be written")

	compiler := NewCompiler()
	output := testutil.CaptureStderr(t, func() {
		err := compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "workflow should compile when redirect is configured")
	})

	assert.Contains(t, output, "workflow redirect configured", "compile output should describe redirect usage")
	assert.Contains(t, output, "owner/repo/workflows/new-location.md@main", "compile output should include redirect target")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	require.NoError(t, os.Remove(lockFile), "lock file should be cleaned up")
}
