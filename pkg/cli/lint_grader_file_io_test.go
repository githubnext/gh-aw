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

func TestGraderFileIOCLI(t *testing.T) {
	module, err := filepath.Abs("../workflow/taint/javascript")
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(module, "node_modules", "@typescript", "native-preview", "package.json")); err != nil {
		t.Skip("requires npm ci --prefix pkg/workflow/taint/javascript")
	}
	for _, tt := range []struct {
		name, script, module, extension string
		level                           workflowlint.Severity
		parseFailure                    bool
	}{
		{"pure", "return session.events.length;", module, ".md", "", false},
		{"file read", "return require('node:fs').readFileSync('PRIVATE_PATH');", module, ".md", workflowlint.Error, false},
		{"paired lock", "return require('node:fs').readFileSync('PRIVATE_PATH');", module, ".lock.yml", workflowlint.Error, false},
		{"native actions excluded", "return require('node:fs');", module, ".yaml", "", false},
		{"parser disabled warning", "return require('node:fs');", "", ".md", workflowlint.Warning, false},
		{"parser unavailable fails", "return 1;", t.TempDir(), ".md", "", true},
		{"invalid syntax fails", "return (PRIVATE_SOURCE;", module, ".md", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "test.md"), []byte("---\ngraders:\n  custom:\n    script: "+tt.script+"\n---\nPrompt\n"), 0o600))
			file := filepath.Join(dir, "test"+tt.extension)
			if tt.extension != ".md" {
				require.NoError(t, os.WriteFile(file, []byte("jobs:\n  check:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo public\n"), 0o600))
			}
			cmd := NewLintCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			args := []string{"--rules=grader-file-io", "--json", file}
			if tt.module != "" {
				args = append(args, "--javascript-module", tt.module)
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tt.parseFailure {
				require.ErrorContains(t, err, "JavaScript AST unavailable")
			} else if tt.level == workflowlint.Error {
				require.ErrorContains(t, err, "workflow lint found")
			} else {
				require.NoError(t, err)
			}
			assert.NotContains(t, output.String(), "PRIVATE_PATH")
			assert.NotContains(t, output.String(), "PRIVATE_SOURCE")
			var diagnostics []workflowLintDiagnostic
			require.NoError(t, json.NewDecoder(&output).Decode(&diagnostics))
			if tt.level == "" {
				assert.Empty(t, diagnostics)
			} else {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, "grader-file-io", diagnostics[0].RuleID)
				assert.Equal(t, tt.level, diagnostics[0].Severity)
				assert.Equal(t, "graders.custom.script", diagnostics[0].Path)
			}
		})
	}
}
