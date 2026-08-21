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
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    string
	}{
		{
			name:        "valid redirect string",
			frontmatter: map[string]any{"redirect": "owner/repo/workflows/new.md@main"},
			expected:    "owner/repo/workflows/new.md@main",
		},
		{
			name:        "missing redirect field",
			frontmatter: map[string]any{},
			expected:    "",
		},
		{
			name:        "empty redirect string",
			frontmatter: map[string]any{"redirect": ""},
			expected:    "",
		},
		{
			name:        "whitespace padded redirect string",
			frontmatter: map[string]any{"redirect": "  owner/repo/workflows/new.md@main  "},
			expected:    "owner/repo/workflows/new.md@main",
		},
		{
			name:        "non-string redirect integer",
			frontmatter: map[string]any{"redirect": 123},
			expected:    "",
		},
		{
			name:        "non-string redirect boolean",
			frontmatter: map[string]any{"redirect": true},
			expected:    "",
		},
		{
			name:        "non-string redirect object",
			frontmatter: map[string]any{"redirect": map[string]any{"target": "owner/repo/workflows/new.md@main"}},
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redirect := compiler.extractRedirect(tt.frontmatter)
			assert.Equal(t, tt.expected, redirect, "redirect extraction should match expected value")
		})
	}
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

	assert.Contains(t, output, "workflow redirect configured: updates move to owner/repo/workflows/new-location.md@main", "compile output should include the redirect info message format")
	assert.Contains(t, output, "owner/repo/workflows/new-location.md@main", "compile output should include redirect target")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	require.NoError(t, os.Remove(lockFile), "lock file should be cleaned up")
}
