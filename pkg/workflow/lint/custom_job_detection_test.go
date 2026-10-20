package lint

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomJobDetectionConditions(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   any
		safe    bool
		invalid bool
	}{
		{"implicit success", nil, true, false},
		{"boolean true", true, true, false},
		{"boolean false", false, true, false},
		{"ordinary predicate", "needs.agent.result == 'success'", true, false},
		{"string is not status call", "contains('always()', 'always')", true, false},
		{"always bypass", "always()", false, false},
		{"failure bypass", "failure()", false, false},
		{"cancelled bypass", "cancelled()", false, false},
		{"not cancelled bypass", "!cancelled()", false, false},
		{"not skipped is insufficient", "always() && needs.detection.result != 'skipped'", false, false},
		{"not failure is insufficient", "always() && needs.detection.result != 'failure'", false, false},
		{"guarded always", "always() && needs.detection.result == 'success'", true, false},
		{"wrapped guard", "${{ always() && needs.detection.result == 'success' }}", true, false},
		{"reversed guard", "'success' == needs.detection.result && always()", true, false},
		{"case and brackets", "ALWAYS() && NEEDS['Detection']['Result'] == 'SUCCESS'", true, false},
		{"success function", "success()", true, false},
		{"success function guard", "always() && success()", true, false},
		{"or bypass", "always() || needs.detection.result == 'success'", false, false},
		{"nested or bypass", "always() && (needs.detection.result == 'success' || inputs.override)", false, false},
		{"both branches guarded", "(always() && needs.detection.result == 'success') || success()", true, false},
		{"outer guard", "(always() || inputs.override) && needs.detection.result == 'success'", true, false},
		{"negated success bypass", "always() && !(needs.detection.result == 'success')", false, false},
		{"other job", "always() && needs.scan.result == 'success'", false, false},
		{"other field", "always() && needs.detection.outputs.result == 'success'", false, false},
		{"dynamic index", "always() && needs[inputs.job].result == 'success'", false, false},
		{"quoted guard", "always() && 'needs.detection.result == ''success'''", false, false},
		{"commented guard", "always() # needs.detection.result == 'success'", false, true},
		{"malformed", "always() &&", false, true},
		{"empty", "", false, true},
		{"incomplete wrapper", "${{ always()", false, true},
		{"trailing expression", "${{ always() }} || true", false, true},
		{"early closing delimiter", "always() }} && needs.detection.result == 'success'", false, true},
		{"non condition", []any{"success()"}, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			safe, err := customJobDetectionCondition(tt.value)
			if tt.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.safe, safe)
		})
	}
}

func TestCustomJobDetectionAuthoredJobs(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source string
		path   string
	}{
		{"missing detection", "jobs:\n  report:\n    needs: agent\n", "jobs.report.needs"},
		{"missing detection with explicit guard", "jobs:\n  report:\n    needs: agent\n    if: always() && needs.detection.result == 'success'\n", "jobs.report.needs"},
		{"always override", "jobs:\n  report:\n    needs: [agent, detection]\n    if: always()\n", "jobs.report.if"},
		{"implicit success", "jobs:\n  report:\n    needs: [agent, detection]\n", ""},
		{"explicit success", "jobs:\n  report:\n    needs: [agent, detection]\n    if: always() && needs.detection.result == 'success'\n", ""},
		{"unrelated job", "jobs:\n  prepare:\n    needs: activation\n", ""},
		{"no dependencies", "jobs:\n  prepare:\n    runs-on: ubuntu-latest\n", ""},
		{"compiler owned conclusion", "jobs:\n  conclusion:\n    needs: agent\n    if: always()\n", ""},
		{"disabled detection", "safe-outputs:\n  threat-detection: false\njobs:\n  report:\n    needs: agent\n", ""},
		{"disabled detection engine", "safe-outputs:\n  threat-detection:\n    engine: false\n    steps: []\njobs:\n  report:\n    needs: agent\n", ""},
		{"custom detection steps still run", "safe-outputs:\n  threat-detection:\n    engine: false\n    steps:\n      - run: echo scan\njobs:\n  report:\n    needs: agent\n", "jobs.report.needs"},
		{"conditional detection", "safe-outputs:\n  threat-detection: '${{ inputs.detect }}'\njobs:\n  report:\n    needs: agent\n", "jobs.report.needs"},
		{"custom jobs not inferred from safe output names", "safe-outputs:\n  jobs:\n    report:\n      needs: agent\n", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := aw.Parse("---\n" + tt.source + "---\nPrompt\n")
			require.NoError(t, err)
			runner, err := Default().Select("custom-job-detection")
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), Input{Document: doc})
			require.NoError(t, err)
			if tt.path == "" {
				assert.Empty(t, diagnostics)
			} else {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, tt.path, diagnostics[0].Path)
				assert.Equal(t, Error, diagnostics[0].Severity)
			}
		})
	}
}

func TestCustomJobDetectionInvalidInput(t *testing.T) {
	rule := CustomJobDetectionRule()
	for _, source := range []string{
		"jobs: invalid\n",
		"jobs:\n  report: invalid\n",
		"jobs:\n  report:\n    needs: [agent, 123]\n",
		"jobs:\n  report:\n    needs: [agent, detection]\n    if: always() &&\n",
		"safe-outputs: invalid\n",
		"safe-outputs:\n  threat-detection:\n    engine: false\n    steps: invalid\n",
	} {
		doc, err := aw.Parse("---\n" + source + "---\n")
		require.NoError(t, err)
		_, err = rule.Check(context.Background(), Input{Document: doc})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "123")
	}
	_, err := rule.Check(context.Background(), Input{})
	require.Error(t, err)
	doc, err := aw.Parse("---\njobs:\n  report:\n    needs: agent\n---\n")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = rule.Check(ctx, Input{Document: doc})
	require.ErrorIs(t, err, context.Canceled)
	diagnostics, err := rule.Check(context.Background(), Input{Document: doc, Workflow: &actions.Workflow{}})
	require.NoError(t, err)
	assert.Empty(t, diagnostics, "the compiled graph confirms detection is absent")
}

func TestCustomJobDetectionSharedFragments(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", ".github", "workflows", "shared", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	runner, err := Default().Select("custom-job-detection")
	require.NoError(t, err)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			source, err := os.ReadFile(file)
			require.NoError(t, err)
			doc, err := aw.Parse(string(source))
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), Input{Document: doc})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	}
}
