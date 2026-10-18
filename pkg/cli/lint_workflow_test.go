//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWorkflowLintCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test binaries use a POSIX shell")
	}
	dir := actionlintTestDir(t)
	script := `#!/bin/sh
if [ "$#" -eq 1 ] && [ "$1" = "--version" ]; then
  printf '1.7.12\n'
  exit 0
fi
printf '%s\n' "$@" > "$ACTIONLINT_TEST_ARGS"
printf '[]\n'
exit "$ACTIONLINT_TEST_EXIT"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "actionlint"), []byte(script), 0o755))
	t.Setenv("PATH", dir)
	originalVersion, originalStats := actionlintVersion, actionlintStats
	actionlintVersion, actionlintStats = "", nil
	t.Cleanup(func() { actionlintVersion, actionlintStats = originalVersion, originalStats })
	for _, tt := range []struct {
		name           string
		run            string
		exit           string
		flags          []string
		wantASTError   bool
		wantASTWarning bool
		wantToolError  bool
	}{
		{name: "safe", run: "echo safe", exit: "0"},
		{name: "secret disclosure", run: "echo '${{ secrets.TOKEN }}'", exit: "0", wantASTError: true},
		{name: "actionlint failure", run: "echo safe", exit: "2", wantToolError: true},
		{name: "both fail", run: "echo '${{ secrets.TOKEN }}'", exit: "2", wantASTError: true, wantToolError: true},
		{name: "coverage warning", run: "unknown '${{ secrets.TOKEN }}'", exit: "0", wantASTWarning: true},
		{name: "script integrations", run: "echo '${{ secrets.TOKEN }}'", exit: "0", flags: []string{"--shellcheck", "--pyflakes"}, wantASTError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(dir, tt.name+".lock.yml")
			require.NoError(t, os.WriteFile(file, []byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: "+tt.run+"\n"), 0o600))
			argsPath := file + ".args"
			t.Setenv("ACTIONLINT_TEST_ARGS", argsPath)
			t.Setenv("ACTIONLINT_TEST_EXIT", tt.exit)
			cmd := NewLintCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(append(tt.flags, file))
			var runErr error
			testutil.CaptureStderr(t, func() { runErr = cmd.Execute() })
			if tt.wantASTError {
				require.ErrorContains(t, runErr, "workflow lint found 1 error")
				assert.Contains(t, output.String(), "error [secret-leak]")
			}
			if tt.wantASTWarning {
				assert.Contains(t, output.String(), "warning [secret-leak]")
			}
			if tt.wantToolError {
				require.ErrorContains(t, runErr, "actionlint failed with exit code 2")
			}
			if !tt.wantASTError && !tt.wantToolError {
				require.NoError(t, runErr)
			}
			args, err := os.ReadFile(argsPath)
			require.NoError(t, err, "actionlint must run even when AST lint fails")
			if len(tt.flags) == 0 {
				assert.Contains(t, string(args), "-shellcheck=\n")
				assert.Contains(t, string(args), "-pyflakes=\n")
			} else {
				assert.NotContains(t, string(args), "-shellcheck=")
				assert.NotContains(t, string(args), "-pyflakes=")
			}
		})
	}
}

func TestBuiltInWorkflowLintCommand(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.lock.yml")
	write := func(content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
	}
	execute := func(args ...string) (string, error) {
		t.Helper()
		cmd := NewLintCommand()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return output.String(), err
	}
	write("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo '${{ secrets.TOKEN }}'\n")
	output, err := execute("--rules", "secret-leak", "--json", file)
	require.ErrorContains(t, err, "found 1 error")
	// Cobra appends usage/errors in a standalone command. The first JSON value
	// must remain independently decodable, including when findings fail the gate.
	var diagnostics []workflowLintDiagnostic
	require.NoError(t, json.NewDecoder(bytes.NewBufferString(output)).Decode(&diagnostics))
	require.Len(t, diagnostics, 1)
	assert.Equal(t, "secret-leak", diagnostics[0].RuleID)
	assert.Equal(t, file, diagnostics[0].File)
	require.Len(t, diagnostics[0].Related, 1)
	assert.NotContains(t, diagnostics[0].Message, "TOKEN")

	write("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo safe\n")
	output, err = execute("--rules=secret-leak", "--json", file)
	require.NoError(t, err)
	assert.JSONEq(t, "[]", output)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.md"), []byte("---\non: workflow_dispatch\n---\nPrint ${{ secrets.TOKEN }}.\n"), 0o600))
	output, err = execute("--rules=secret-leak", "--json", file)
	require.Error(t, err)
	require.NoError(t, json.NewDecoder(bytes.NewBufferString(output)).Decode(&diagnostics))
	require.Len(t, diagnostics, 1)
	assert.Equal(t, "markdown", diagnostics[0].Path)

	output, err = execute("--list-rules")
	require.NoError(t, err)
	assert.Contains(t, output, "secret-leak")
	_, err = execute("--rules=missing", file)
	require.ErrorContains(t, err, "unknown workflow lint rule")
	_, err = execute("--json", file)
	require.ErrorContains(t, err, "--json requires --rules")
	_, err = execute("--rules=secret-leak", "--shellcheck", file)
	require.ErrorContains(t, err, "require actionlint")
}

func TestBuiltInWorkflowLintReadAndParseFailures(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad.lock.yml")
	require.NoError(t, os.WriteFile(file, []byte("jobs: [private-literal"), 0o600))
	cmd := NewLintCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--rules=secret-leak", "--json", file})
	err := cmd.Execute()
	require.ErrorContains(t, err, "could not parse workflow Actions AST")
	assert.NotContains(t, output.String(), "private-literal")
}
