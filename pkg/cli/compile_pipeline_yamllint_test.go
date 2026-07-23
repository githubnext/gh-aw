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

func TestCompileWorkflows_RunsYamllintForSpecificFiles(t *testing.T) {
	testCompileWorkflowsRunsYamllint(t, []string{"test"}, true)
}

func TestCompileWorkflows_RunsYamllintForDirectoryCompile(t *testing.T) {
	testCompileWorkflowsRunsYamllint(t, nil, false)
}

func testCompileWorkflowsRunsYamllint(t *testing.T, markdownFiles []string, strict bool) {
	t.Helper()

	tmpDir := t.TempDir()
	require.NoError(t, initTestGitRepo(tmpDir))

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))

	workflowContent := `---
name: Test Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
strict: false
---

# Test Workflow

This is a test workflow for yamllint batch execution.
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "test.md"), []byte(workflowContent), 0o644))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	originalRunner := runBatchYamllintOnFiles
	t.Cleanup(func() {
		runBatchYamllintOnFiles = originalRunner
	})

	var gotLockFiles []string
	var gotVerbose bool
	var gotStrict bool
	var calls int
	runBatchYamllintOnFiles = func(lockFiles []string, verbose bool, strictArg bool) error {
		calls++
		gotLockFiles = append([]string(nil), lockFiles...)
		gotVerbose = verbose
		gotStrict = strictArg
		return nil
	}

	config := CompileConfig{
		MarkdownFiles: markdownFiles,
		NoEmit:        false,
		Yamllint:      true,
		Strict:        strict,
	}

	_, err = CompileWorkflows(context.Background(), config)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	assert.Equal(t, []string{filepath.Join(workflowsDir, "test.lock.yml")}, gotLockFiles)
	assert.False(t, gotVerbose)
	assert.Equal(t, strict, gotStrict)
}
