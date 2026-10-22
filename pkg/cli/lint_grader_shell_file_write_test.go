//go:build !integration

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func graderShellCLIFixture(t *testing.T) (root, workflow string) {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".grader-shell-cli-test-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	root, err = filepath.Abs(dir)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".github", "workflows", "shared"), 0o700))
	workflow = filepath.Join(root, ".github", "workflows", "shared", "grader.md")
	require.NoError(t, os.WriteFile(workflow, []byte("---\ngraders:\n  operational-value:\n    enabled: false\n    run: ./check.sh\n---\nPrompt\n"), 0o600))
	return root, workflow
}

func TestLintGraderShellFileWriteCLI(t *testing.T) {
	root, workflow := graderShellCLIFixture(t)
	script := filepath.Join(filepath.Dir(workflow), "check.sh")
	marker := filepath.Join(root, "must-not-exist")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/bash\ntouch '"+marker+"'\n"), 0o600))
	t.Setenv("PATH", "")
	cmd := NewLintCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--rules=grader-shell-file-write", "--json", workflow})
	require.ErrorContains(t, cmd.Execute(), "workflow lint found 1 error")
	assert.Contains(t, output.String(), `"rule_id":"grader-shell-file-write"`)
	assert.Contains(t, output.String(), `"path":"graders.operational-value.run"`)
	assert.Contains(t, output.String(), `"script_line":2`)
	assert.NotContains(t, output.String(), marker)
	_, err := os.Stat(marker)
	require.ErrorIs(t, err, os.ErrNotExist, "lint must never execute the grader")
	_, err = os.Stat(filepath.Join(filepath.Dir(workflow), "grader.lock.yml"))
	require.ErrorIs(t, err, os.ErrNotExist, "lint must never compile")
}

func TestLintGraderShellSourcePath(t *testing.T) {
	for _, companion := range []bool{false, true} {
		t.Run(map[bool]string{false: "imported fragment", true: "compiled companion"}[companion], func(t *testing.T) {
			_, workflow := graderShellCLIFixture(t)
			require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(workflow), "check.sh"), []byte("#!/bin/bash\njq '.events | length' \"$GH_AW_SESSION\"\n"), 0o600))
			file := workflow
			if companion {
				file = filepath.Join(filepath.Dir(workflow), "grader.lock.yml")
				require.NoError(t, os.WriteFile(file, []byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo public\n"), 0o600))
			}
			input, err := loadWorkflowLintInput(file)
			require.NoError(t, err)
			assert.Equal(t, workflow, input.SourcePath)
			runner, err := workflowlint.Default().Select("grader-shell-file-write")
			require.NoError(t, err)
			diagnostics, err := lintWorkflowFile(t.Context(), file, runner)
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	}
}

func TestLintGraderShellMissingAndInvalidSources(t *testing.T) {
	_, workflow := graderShellCLIFixture(t)
	runner, err := workflowlint.Default().Select("grader-shell-file-write")
	require.NoError(t, err)
	var output bytes.Buffer
	err = runWorkflowLint(t.Context(), []string{workflow}, runner, &output, true)
	require.ErrorContains(t, err, "existing regular Bash file")
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(workflow), "check.sh"), []byte("#!/bin/bash\nif PRIVATE\n"), 0o600))
	output.Reset()
	err = runWorkflowLint(t.Context(), []string{workflow}, runner, &output, true)
	require.ErrorContains(t, err, "invalid Bash syntax")
	assert.NotContains(t, err.Error(), "PRIVATE")
	assert.JSONEq(t, "[]", output.String())
}
