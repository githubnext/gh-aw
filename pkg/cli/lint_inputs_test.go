//go:build !integration

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
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
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "VERY_SENSITIVE_SCALAR")
			var ids []string
			for _, diagnostic := range diagnostics {
				ids = append(ids, diagnostic.RuleID)
			}
			assert.Contains(t, ids, wantRule)
		})
	}
}
