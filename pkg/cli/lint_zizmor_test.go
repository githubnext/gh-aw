//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoricalZizmorRulesCLI(t *testing.T) {
	const job = "jobs:\n  check:\n    runs-on: ubuntu-latest\n"
	for _, test := range []struct {
		id, source, path string
		severity         workflowlint.Severity
	}{
		{"artipacked", job + "    steps:\n      - uses: actions/checkout@v6\n", "jobs.check.steps[0].uses", workflowlint.Warning},
		{"cache-poisoning", "on: release\n" + job + "    steps:\n      - uses: actions/setup-go@v6\n", "jobs.check.steps[0].uses", workflowlint.Warning},
		{"excessive-permissions", "permissions: {issues: write}\n" + job + "    steps:\n      - run: echo public\n", "permissions.issues", workflowlint.Warning},
		{"obfuscation", job + "    outputs: {comment_id: \"${{ '' }}\"}\n    steps:\n      - run: echo public\n", "jobs.check.outputs.comment_id", workflowlint.Warning},
		{"template-injection", job + "    steps:\n      - uses: actions/github-script@v8\n        with:\n          script: console.log('${{ inputs.title }}')\n", "jobs.check.steps[0].with.script", workflowlint.Error},
	} {
		for _, extension := range []string{".md", ".lock.yml", ".yaml"} {
			t.Run(test.id+extension, func(t *testing.T) {
				dir := t.TempDir()
				file := filepath.Join(dir, "check"+extension)
				source := test.source
				if extension == ".md" {
					source = "---\n" + source + "---\nPrompt.\n"
				}
				require.NoError(t, os.WriteFile(file, []byte(source), 0o600))
				cmd := NewLintCommand()
				var output bytes.Buffer
				cmd.SetOut(&output)
				cmd.SetErr(&output)
				cmd.SetArgs([]string{"--rules=" + test.id, "--json", file})
				err := cmd.Execute()
				if test.severity == workflowlint.Error {
					require.ErrorContains(t, err, "workflow lint found 1 error")
				} else {
					require.NoError(t, err)
				}
				var diagnostics []workflowLintDiagnostic
				require.NoError(t, json.NewDecoder(&output).Decode(&diagnostics))
				if extension == ".md" && test.id == "excessive-permissions" {
					assert.Empty(t, diagnostics, "AW root grants scope the agent, not the workflow")
					return
				}
				require.Len(t, diagnostics, 1)
				assert.Equal(t, test.id, diagnostics[0].RuleID)
				assert.Equal(t, test.severity, diagnostics[0].Severity)
				assert.Equal(t, test.path, diagnostics[0].Path)
			})
		}
	}
}
