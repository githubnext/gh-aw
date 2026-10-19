//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestGitHubCLIContractCompileWarnings(t *testing.T) {
	for _, test := range []struct {
		name, config, prompt, rule string
	}{
		{"pager syntax", "tools:\n  bash: [gh]\n", "Run `gh pr view 42 --no-pager`.", "github-cli-syntax"},
		{"PR field", "tools:\n  bash: [gh]\n", "Run `gh pr view 42 --json authorAssociation`.", "github-cli-syntax"},
		{"proxy diff", "tools:\n  github:\n    mode: gh-proxy\n", "Run `gh pr diff 42`.", "github-proxy-diff"},
		{"host step", "steps:\n  - env:\n      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n    run: gh pr view 42 --no-pager\n", "Inspect the snapshot.", "github-cli-syntax"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "check.md")
			source := "---\non: workflow_dispatch\nengine: copilot\n" + test.config + "---\n" + test.prompt + "\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			compiler := NewCompiler()
			var compileErr error
			output := testutil.CaptureStderr(t, func() { compileErr = compiler.CompileWorkflow(path) })
			require.NoError(t, compileErr, "best-effort contract linting is advisory")
			require.Contains(t, output, "["+test.rule+"]")
			require.Positive(t, compiler.GetWarningCount())
			_, err := os.Stat(filepath.Join(directory, "check.lock.yml"))
			require.NoError(t, err, "lint warnings must not prevent emission")
		})
	}
}

func TestGitHubCLIContractCompileHandlesExpressions(t *testing.T) {
	for _, test := range []struct {
		name, step string
		wantRule   bool
	}{
		{"expression plus literal pager flag", "gh pr view ${{ github.event.pull_request.number }} --no-pager", true},
		{"non-gh command with expression", "sighthound --sha ${{ github.sha }}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "check.md")
			source := "---\non: workflow_dispatch\nengine: copilot\nstrict: false\nsteps:\n  - run: " + test.step + "\n---\nInspect the snapshot.\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			compiler := NewCompiler()
			var compileErr error
			output := testutil.CaptureStderr(t, func() { compileErr = compiler.CompileWorkflow(path) })
			require.NoError(t, compileErr, "best-effort contract linting is advisory")
			if test.wantRule {
				require.Contains(t, output, "[github-cli-syntax]")
				require.NotContains(t, output, "Incomplete GitHub CLI syntax analysis")
			} else {
				require.NotContains(t, output, "[github-cli-syntax]")
				require.NotContains(t, output, "GitHub CLI contract linting incomplete")
			}
		})
	}
}

func TestGitHubCLIContractDeduplicatesImportedCommands(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Tools:            map[string]any{"bash": []any{"gh"}},
		MarkdownContent:  "Run `gh pr view 42 --no-pager`.",
		ImportedMarkdown: "Run `gh pr view 42 --no-pager`.",
	}
	output := testutil.CaptureStderr(t, func() { compiler.validatePromptTools(data, "check.md") })
	require.Equal(t, 1, compiler.GetWarningCount())
	require.Contains(t, output, "GH_PAGER=cat")
	require.Equal(t, []any{"gh"}, data.Tools["bash"], "linting must not alter grants")
}
