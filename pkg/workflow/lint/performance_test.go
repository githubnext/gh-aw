package lint

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintPerformanceMarkers(t *testing.T) {
	if os.Getenv("GH_AW_LINT_PERF_HELPER") == "1" {
		runner, err := New(
			Rule{ID: "public-rule", Description: "public", Severity: Error,
				Check: func(context.Context, Input) ([]Diagnostic, error) {
					return []Diagnostic{
						{Path: "name", Message: "private diagnostic"},
						{Path: "name", Message: "private diagnostic"},
					}, errors.New("private failure")
				}},
			Rule{ID: "aw-only", Description: "aw", Severity: Error, Targets: AWAST,
				Check: func(context.Context, Input) ([]Diagnostic, error) {
					t.Fatal("an unsupported target must not be checked or timed")
					return nil, nil
				}},
		)
		require.NoError(t, err)
		input := Input{Workflow: &actions.Workflow{Name: "private workflow"}}
		diagnostics, err := runner.Run(context.Background(), input)
		require.ErrorContains(t, err, "private failure")
		require.Len(t, diagnostics, 1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = runner.Run(ctx, input)
		require.ErrorIs(t, err, context.Canceled)
		return
	}
	for _, debug := range []string{"workflow:lint:perf", "-workflow:lint:perf"} {
		t.Run(debug, func(t *testing.T) {
			// Logger configuration is fixed at process initialization.
			cmd := exec.Command(os.Args[0], "-test.run=^TestLintPerformanceMarkers$")
			cmd.Env = append(os.Environ(), "GH_AW_LINT_PERF_HELPER=1", "DEBUG="+debug, "DEBUG_COLORS=0")
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			output, err := cmd.Output()
			require.NoError(t, err, "%s\n%s", output, stderr.String())
			if debug[0] == '-' {
				assert.Empty(t, stderr.String())
				return
			}
			text := stderr.String()
			assert.Contains(t, text, "rule_start rule_id=public-rule actions=true aw=false")
			assert.Regexp(t, `rule_complete rule_id=public-rule duration_ns=\d+ diagnostics=2 failed=true`, text)
			assert.Regexp(t, `diagnostics_complete duration_ns=\d+ before=2 after=1`, text)
			assert.Regexp(t, `run_complete duration_ns=\d+ rules=1 diagnostics=1 failed=true`, text)
			assert.Regexp(t, `run_complete duration_ns=\d+ rules=0 diagnostics=0 failed=true`, text)
			assert.NotContains(t, text, "aw-only")
			assert.NotContains(t, text, "private")
		})
	}
}
