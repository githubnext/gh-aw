//go:build !integration

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintWorkflowInputKinds(t *testing.T) {
	for _, extension := range []string{".md", ".yml", ".yaml", ".lock.yml"} {
		t.Run(extension, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "workflow"+extension)
			content := "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo '${{ secrets.TOKEN }}'\n"
			if extension == ".md" {
				content = "---\ncheck-for-updates: false\nsteps:\n  - shell: bash\n    run: echo '${{ secrets.TOKEN }}'\n---\n"
			}
			require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
			if extension == ".yml" || extension == ".yaml" {
				require.NoError(t, os.WriteFile(file+".md", []byte("---\nMALFORMED: [PRIVATE\n---\n"), 0o600))
			}
			cmd := NewLintCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"--rules=secret-leak,version-check", "--json", file})
			err := cmd.Execute()
			require.ErrorContains(t, err, "workflow lint found")
			assert.Contains(t, output.String(), `"rule_id":"secret-leak"`)
			if extension == ".md" {
				assert.Contains(t, output.String(), `"rule_id":"version-check"`)
				_, err := os.Stat(filepath.Join(dir, "workflow.lock.yml"))
				assert.ErrorIs(t, err, os.ErrNotExist, "source lint must never compile or write a lock file")
			} else {
				assert.NotContains(t, output.String(), `"rule_id":"version-check"`)
			}
		})
	}
}

func TestLintDirectoryMixedASTs(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"paired.md":       "---\ncheck-for-updates: false\n---\n",
		"paired.lock.yml": "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo public\n",
		"authored.md":     "---\npermissions: read-all\n---\n",
		"README.md":       "Plain documentation without workflow configuration\n",
		"native.yml":      "jobs: {}\n",
		"native.yaml":     "jobs: {}\n",
		"ignore.txt":      "not a workflow",
		"malformed.md":    "---\nkey: [PRIVATE\n---\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	files, err := resolveLockFilesForLint([]string{dir, filepath.Join(dir, "native.yaml")}, dir)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "authored.md"), filepath.Join(dir, "malformed.md"),
		filepath.Join(dir, "native.yaml"), filepath.Join(dir, "native.yml"),
		filepath.Join(dir, "paired.lock.yml"),
	}, files)
	runner, err := workflowlint.Default().Select("version-check")
	require.NoError(t, err)
	diagnostics, err := lintWorkflowFile(context.Background(), filepath.Join(dir, "paired.lock.yml"), runner)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, "version-check", diagnostics[0].RuleID)
}

func TestLintAWNeverRunsActionlint(t *testing.T) {
	t.Setenv("PATH", "")
	dir := t.TempDir()
	file := filepath.Join(dir, "authored.md")
	content := "---\nstrict: false\ncheck-for-updates: false\n---\nPublic prompt\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
	cmd := NewLintCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{file})
	require.NoError(t, cmd.Execute(), "Markdown must not be sent to external actionlint")
	assert.Contains(t, output.String(), "version-check")
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, content, string(after))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "AST adapters must not emit files")
}

func TestLintInputParseFailuresPreserveOtherAST(t *testing.T) {
	for _, badActions := range []bool{true, false} {
		t.Run(map[bool]string{true: "Actions", false: "AW"}[badActions], func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "workflow.lock.yml")
			actions := "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo '${{ secrets.TOKEN }}'\n"
			authored := "---\ncheck-for-updates: false\n---\n"
			wantRule := "secret-leak"
			if badActions {
				actions = "jobs: [VERY_SENSITIVE_SCALAR\n"
				wantRule = "version-check"
			} else {
				authored = "---\nkey: [VERY_SENSITIVE_SCALAR\n---\n"
			}
			require.NoError(t, os.WriteFile(file, []byte(actions), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.md"), []byte(authored), 0o600))
			diagnostics, err := lintWorkflowFile(context.Background(), file, workflowlint.Default())
			if badActions {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "VERY_SENSITIVE_SCALAR")
			} else {
				require.NoError(t, err)
			}
			var ids []string
			for _, diagnostic := range diagnostics {
				ids = append(ids, diagnostic.RuleID)
			}
			assert.Contains(t, ids, wantRule)
		})
	}
}

func TestLintCompanionParsingBestEffort(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source string
	}{
		{name: "plain documentation", source: "Plain documentation without workflow configuration\n"},
		{name: "malformed frontmatter", source: "---\nkey: [VERY_SENSITIVE_SCALAR\n---\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "workflow.lock.yml")
			require.NoError(t, os.WriteFile(file, []byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo public\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.md"), []byte(tt.source), 0o600))
			runner, err := workflowlint.Default().Select("secret-leak")
			require.NoError(t, err)
			var output bytes.Buffer
			require.NoError(t, runWorkflowLint(t.Context(), []string{file}, runner, &output, true))
			assert.JSONEq(t, "[]", output.String())
		})
	}
}

func TestLintMalformedAWInputFailsWithoutSourceValues(t *testing.T) {
	file := filepath.Join(t.TempDir(), "workflow.md")
	require.NoError(t, os.WriteFile(file, []byte("---\nkey: [VERY_SENSITIVE_SCALAR\n---\n"), 0o600))
	var output bytes.Buffer
	err := runWorkflowLint(t.Context(), []string{file}, workflowlint.Default(), &output, true)
	require.ErrorContains(t, err, file+": could not parse workflow AW AST")
	assert.NotContains(t, err.Error(), "VERY_SENSITIVE_SCALAR")
	assert.JSONEq(t, "[]", output.String())
}

func TestLintCompanionReadFailurePreservesCompiledDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "workflow.lock.yml")
	require.NoError(t, os.WriteFile(file, []byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo '${{ secrets.TOKEN }}'\n"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "workflow.md"), 0o700))
	var output bytes.Buffer
	err := runWorkflowLint(t.Context(), []string{file}, workflowlint.Default(), &output, true)
	require.ErrorContains(t, err, file+": reading companion AW document")
	assert.Contains(t, output.String(), `"rule_id":"secret-leak"`)
}

func TestWorkflowLintLoadsInputsPerFile(t *testing.T) {
	for _, tt := range []struct {
		name       string
		javascript *taint.JavaScriptOptions
		ruleID     string
	}{
		{name: "disabled", ruleID: "secret-leak"},
		{name: "deep", javascript: &taint.JavaScriptOptions{Mode: taint.JavaScriptDeep}, ruleID: "secret-leak"},
		{name: "fast unselected", javascript: &taint.JavaScriptOptions{Mode: taint.JavaScriptFast}, ruleID: "inspect-input"},
		{name: "fast selected", javascript: &taint.JavaScriptOptions{Mode: taint.JavaScriptFast, Parser: &lintPrimingParser{}}, ruleID: "secret-leak"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			first := filepath.Join(dir, "first.lock.yml")
			second := filepath.Join(dir, "second.lock.yml")
			actions := []byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo public\n")
			authored := []byte("---\nstrict: false\n---\n")
			require.NoError(t, os.WriteFile(first, actions, 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "first.md"), authored, 0o600))
			require.NoError(t, os.WriteFile(second, []byte("jobs: [PRIVATE\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "second.md"), []byte("---\nkey: [PRIVATE\n---\n"), 0o600))
			checked := 0
			runner, err := workflowlint.New(workflowlint.Rule{
				ID: tt.ruleID, Description: "Inspect inputs as they are loaded", Severity: workflowlint.Warning,
				Check: func(_ context.Context, input workflowlint.Input) ([]workflowlint.Diagnostic, error) {
					checked++
					require.NotNil(t, input.Workflow)
					require.NotNil(t, input.Document)
					if checked == 1 {
						require.NoError(t, os.WriteFile(second, actions, 0o600))
						require.NoError(t, os.WriteFile(filepath.Join(dir, "second.md"), authored, 0o600))
					}
					return nil, nil
				},
			})
			require.NoError(t, err)
			var output bytes.Buffer
			require.NoError(t, runWorkflowLint(t.Context(), []string{first, second}, runner, &output, true, workflowLintOptions{JavaScript: tt.javascript}))
			assert.Equal(t, 2, checked, "load each input only after the preceding file's rules finish")
			assert.JSONEq(t, "[]", output.String())
		})
	}
}
