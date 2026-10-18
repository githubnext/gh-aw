//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
