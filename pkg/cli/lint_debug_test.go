//go:build !integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowLintCoverageDebug(t *testing.T) {
	file := filepath.Join(t.TempDir(), "test.lock.yml")
	const source = "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: |\n          unknown '${{ secrets.TOKEN }}'\n          echo '${{ secrets.TOKEN }}'\n"
	require.NoError(t, os.WriteFile(file, []byte(source), 0o600))
	runner, err := workflowlint.Default().Select("secret-leak")
	require.NoError(t, err)
	internal, err := lintWorkflowFile(context.Background(), file, runner)
	require.NoError(t, err)
	require.Len(t, internal, 2, "coverage diagnostics must remain available to library callers")
	for _, debug := range []bool{false, true} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run("debug="+strconv.FormatBool(debug)+"/json="+strconv.FormatBool(jsonOutput), func(t *testing.T) {
				cmd := NewLintCommand()
				var output bytes.Buffer
				cmd.SetOut(&output)
				cmd.SetErr(&output)
				cmd.SilenceErrors, cmd.SilenceUsage = true, true
				args := []string{"--rules=secret-leak", "--debug=" + strconv.FormatBool(debug), file}
				if jsonOutput {
					args = append(args, "--json")
				}
				cmd.SetArgs(args)
				require.ErrorContains(t, cmd.Execute(), "workflow lint found 1 error")
				if jsonOutput {
					var diagnostics []workflowLintDiagnostic
					require.NoError(t, json.Unmarshal(output.Bytes(), &diagnostics))
					expected := 1
					if debug {
						expected = 2
					}
					require.Len(t, diagnostics, expected)
					assert.Contains(t, output.String(), `"severity":"error"`)
					assert.Equal(t, debug, bytes.Contains(output.Bytes(), []byte(`"severity":"warning"`)))
				} else {
					assert.Contains(t, output.String(), "error [secret-leak]")
					assert.Equal(t, debug, bytes.Contains(output.Bytes(), []byte("warning [secret-leak]")))
				}
			})
		}
	}
}
