//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptLintConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name  string
		flags []string
		error string
	}{
		{"disabled", nil, ""},
		{"explicit module", []string{"--javascript-module", t.TempDir()}, ""},
		{"explicit deep", []string{"--javascript-module", t.TempDir(), "--javascript-mode", "deep"}, ""},
		{"invalid mode", []string{"--javascript-module", t.TempDir(), "--javascript-mode", "unknown"}, "must be fast or deep"},
		{"mode requires module", []string{"--javascript-mode", "deep"}, "requires --javascript-module"},
		{"root requires module", []string{"--javascript-root", t.TempDir()}, "require --javascript-module"},
		{"mapping requires root", []string{"--javascript-module", t.TempDir(), "--javascript-runtime-path", "/runner=helpers"}, "requires --javascript-root"},
		{"invalid mapping", []string{"--javascript-module", t.TempDir(), "--javascript-root", t.TempDir(), "--javascript-runtime-path", "/runner=../outside"}, "repository-relative-directory"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewLintCommand()
			require.NoError(t, cmd.ParseFlags(tt.flags))
			options, closeFiles, err := javascriptLintOptions(cmd)
			if tt.error != "" {
				require.ErrorContains(t, err, tt.error)
			} else {
				require.NoError(t, err)
				if len(tt.flags) == 0 {
					assert.Nil(t, options)
				} else {
					require.NotNil(t, options)
					assert.NotNil(t, options.Parser)
					mode, err := cmd.Flags().GetString("javascript-mode")
					require.NoError(t, err)
					assert.Equal(t, mode, string(options.Mode))
				}
			}
			require.NoError(t, closeFiles())
		})
	}
}

func TestJavaScriptLintCLI(t *testing.T) {
	module, err := filepath.Abs("../workflow/taint/javascript")
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(module, "node_modules", "@typescript", "native-preview", "package.json")); err != nil {
		t.Skip("requires npm ci --prefix pkg/workflow/taint/javascript")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "javascript.lock.yml")
	for _, tt := range []struct {
		name    string
		source  string
		module  string
		want    workflowlint.Severity
		message string
		debug   bool
	}{
		{"safe authentication", `fetch("https://example.test", {headers:{Authorization:process.env.TOKEN}})`, module, "", "", false},
		{"leak", `console.log(process.env.TOKEN)`, module, workflowlint.Error, "reaches log", false},
		{"unknown visible", `unknown(process.env.TOKEN)`, module, workflowlint.Warning, "not modeled", false},
		{"unknown debug", `unknown(process.env.TOKEN)`, module, workflowlint.Warning, "not modeled", true},
		{"unavailable module visible", `console.log(process.env.TOKEN)`, t.TempDir(), workflowlint.Warning, "AST unavailable", false},
		{"unavailable module debug", `console.log(process.env.TOKEN)`, t.TempDir(), workflowlint.Warning, "AST unavailable", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			content := "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - env:\n          TOKEN: ${{ secrets.TOKEN }}\n        shell: node {0}\n        run: |\n          " + tt.source + "\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
			cmd := NewLintCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			args := []string{"--rules", "secret-leak", "--json", "--javascript-module", tt.module, file}
			if tt.debug {
				args = append(args, "--debug")
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tt.want == workflowlint.Error {
				require.ErrorContains(t, err, "workflow lint found")
			} else {
				require.NoError(t, err)
			}
			var diagnostics []workflowLintDiagnostic
			require.NoError(t, json.NewDecoder(&output).Decode(&diagnostics), strings.TrimSpace(output.String()))
			if tt.want == "" {
				assert.Empty(t, diagnostics)
			} else {
				require.NotEmpty(t, diagnostics)
				assert.Equal(t, tt.want, diagnostics[0].Severity)
				assert.Contains(t, diagnostics[0].Message, tt.message)
				assert.NotContains(t, diagnostics[0].Message, tt.source)
			}
		})
	}
}
