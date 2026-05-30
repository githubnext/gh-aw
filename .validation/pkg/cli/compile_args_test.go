//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandCompileArg_RegularFile(t *testing.T) {
	// A plain workflow name should be returned as-is
	result, err := expandCompileArg("my-workflow", false)
	require.NoError(t, err, "plain workflow name should not error")
	assert.Equal(t, []string{"my-workflow"}, result, "plain workflow name should be returned unchanged")
}

func TestExpandCompileArg_FilePath(t *testing.T) {
	// A non-existent file path should be returned as-is
	result, err := expandCompileArg(".github/workflows/my-workflow.md", false)
	require.NoError(t, err, "non-existent file path should not error")
	assert.Equal(t, []string{".github/workflows/my-workflow.md"}, result, "file path should be returned unchanged")
}

func TestExpandCompileArg_LocalDirectory(t *testing.T) {
	// Create a temp directory with workflow files
	tmpDir := t.TempDir()
	writeWorkflowFile(t, tmpDir, "workflow-a.md")
	writeWorkflowFile(t, tmpDir, "workflow-b.md")

	result, err := expandCompileArg(tmpDir, false)
	require.NoError(t, err, "local directory should expand without error")
	assert.Len(t, result, 2, "should return all .md files in the directory")
}

func TestExpandCompileArg_LocalDirectory_Empty(t *testing.T) {
	// Directory with no .md files should error
	tmpDir := t.TempDir()
	_, err := expandCompileArg(tmpDir, false)
	require.Error(t, err, "empty directory should return an error")
	assert.Contains(t, err.Error(), "no workflow markdown files found", "error should mention no workflow files")
}

func TestExpandCompileArg_URLPassthrough(t *testing.T) {
	// URLs should be returned as-is (not processed)
	url := "https://github.com/org/repo/tree/main/.github/workflows"
	result, err := expandCompileArg(url, false)
	require.NoError(t, err, "URL should not error")
	assert.Equal(t, []string{url}, result, "URL should be returned unchanged")
}

func TestResolveCompileArgs_Empty(t *testing.T) {
	result, err := resolveCompileArgs(nil, false)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestResolveCompileArgs_Mixed(t *testing.T) {
	// Create a temp directory with a workflow file
	tmpDir := t.TempDir()
	writeWorkflowFile(t, tmpDir, "workflow-a.md")

	// Mix: a plain workflow name + a directory
	result, err := resolveCompileArgs([]string{"plain-workflow", tmpDir}, false)
	require.NoError(t, err, "mixed args should expand without error")
	require.Len(t, result, 2, "should have one plain name + one expanded file")

	// The plain workflow name should be unchanged
	assert.Equal(t, "plain-workflow", result[0], "first arg should be unchanged")
	// The directory arg should be the single .md file in tmpDir
	assert.True(t, strings.HasSuffix(result[1], "workflow-a.md"), "second arg should be the expanded .md file")
}

// writeWorkflowFile creates a minimal workflow .md file in dir with the given name.
func writeWorkflowFile(t *testing.T, dir, name string) {
	t.Helper()
	content := "---\non: push\n---\n\n# Test Workflow\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644),
		"write workflow file %s", name)
}
