package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestCompileValidation_NoEmitIgnoresStaleLock(t *testing.T) {
	for _, tt := range []struct {
		name     string
		parsed   bool
		explicit bool
	}{
		{"file", false, true},
		{"parsed", true, true},
		{"file-compiler-mode", false, false},
		{"parsed-compiler-mode", true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.md")
			content := []byte("---\non: workflow_dispatch\npermissions:\n  contents: read\n---\n# Test\nDo nothing.\n")
			require.NoError(t, os.WriteFile(path, content, 0o600))
			lockPath := stringutil.MarkdownToLockFile(path)
			staleLock := []byte("jobs: [malformed YAML\n")
			require.NoError(t, os.WriteFile(lockPath, staleLock, 0o600))

			compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
			opts := CompileValidationOptions{NoEmit: tt.explicit}
			if tt.parsed {
				data, err := compiler.ParseWorkflowFile(path)
				require.NoError(t, err)
				require.NoError(t, CompileWorkflowDataWithValidation(context.Background(), compiler, data, path, opts))
			} else {
				require.NoError(t, CompileWorkflowWithValidation(context.Background(), compiler, path, opts))
			}
			got, err := os.ReadFile(lockPath)
			require.NoError(t, err)
			require.Equal(t, staleLock, got)
			got, err = os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, content, got)
		})
	}
}

func TestCompileSingleFile_NoEmitIgnoresStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.md")
	require.NoError(t, os.WriteFile(path, []byte("---\non: workflow_dispatch\npermissions:\n  contents: read\n---\n# Test\nDo nothing.\n"), 0o600))
	lockPath := stringutil.MarkdownToLockFile(path)
	staleLock := []byte("jobs: [malformed YAML\n")
	require.NoError(t, os.WriteFile(lockPath, staleLock, 0o600))

	compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
	stats := &CompilationStats{}
	require.True(t, compileSingleFile(context.Background(), compiler, path, stats, false, false))
	require.Zero(t, stats.Errors)
	got, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.Equal(t, staleLock, got)
}

func TestCompileWorkflows_ReadOnlyIgnoresStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.md")
	require.NoError(t, os.WriteFile(path, []byte("---\non: workflow_dispatch\npermissions:\n  contents: read\n---\n# Test\nDo nothing.\n"), 0o600))
	lockPath := stringutil.MarkdownToLockFile(path)
	staleLock := []byte("jobs: [malformed YAML\n")
	require.NoError(t, os.WriteFile(lockPath, staleLock, 0o600))

	_, err := CompileWorkflows(context.Background(), CompileConfig{
		MarkdownFiles: []string{path},
		ReadOnly:      true,
	})
	require.NoError(t, err)
	got, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.Equal(t, staleLock, got)
}
