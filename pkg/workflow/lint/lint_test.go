package lint

import (
	"context"
	"errors"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleRegistrationAndSelection(t *testing.T) {
	runner := Default()
	_, err := New(runner.Rules()...)
	require.NoError(t, err, "built-in rule registrations must satisfy the same contract as extensions")
	assert.Len(t, runner.Rules(), 1)
	selected, err := runner.Select("secret-leak", "secret-leak")
	require.NoError(t, err)
	assert.Len(t, selected.Rules(), 1)
	_, err = runner.Select("missing")
	require.ErrorContains(t, err, "unknown workflow lint rule")
	_, err = New(SecretLeakRule(), SecretLeakRule())
	require.ErrorContains(t, err, "duplicate")
	for _, rule := range []Rule{{}, {ID: "Bad ID"}, {ID: "valid", Description: "test", Check: func(context.Context, Input) ([]Diagnostic, error) { return nil, nil }}} {
		_, err := New(rule)
		require.Error(t, err)
	}
	rules := runner.Rules()
	rules[0].ID = "changed"
	assert.Equal(t, "secret-leak", runner.Rules()[0].ID)
}

func TestIndependentRulesAndFailures(t *testing.T) {
	first := Rule{
		ID: "structural", Description: "An independent non-taint rule", Severity: Warning,
		Check: func(_ context.Context, input Input) ([]Diagnostic, error) {
			return []Diagnostic{{Path: "name", Message: input.Workflow.Name}}, errors.New("check incomplete")
		},
	}
	second := Rule{
		ID: "another", Description: "Continue after an earlier rule failure", Severity: Error,
		Check: func(context.Context, Input) ([]Diagnostic, error) {
			return []Diagnostic{{Path: "env", Message: "found"}}, nil
		},
	}
	runner, err := New(first, second)
	require.NoError(t, err)
	diagnostics, err := runner.Run(context.Background(), Input{Workflow: &actions.Workflow{Name: "example"}})
	require.ErrorContains(t, err, "lint rule structural: check incomplete")
	require.Len(t, diagnostics, 2)
	assert.Equal(t, "another", diagnostics[0].RuleID)
	assert.Equal(t, Error, diagnostics[0].Severity)
	assert.Equal(t, Warning, diagnostics[1].Severity)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = runner.Run(ctx, Input{Workflow: &actions.Workflow{}})
	require.ErrorIs(t, err, context.Canceled)
	_, err = runner.Run(context.Background(), Input{})
	require.Error(t, err)
}

func TestSecretRuleDiagnostics(t *testing.T) {
	workflow, err := actions.Parse([]byte(`
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo '${{ secrets.TOKEN }}'
      - shell: pwsh
        run: Write-Output test
`))
	require.NoError(t, err)
	diagnostics, err := Default().Run(context.Background(), Input{Workflow: workflow})
	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, Error, diagnostics[0].Severity)
	assert.Equal(t, Warning, diagnostics[1].Severity)
	assert.NotContains(t, diagnostics[0].Message, "TOKEN")
	assert.Contains(t, diagnostics[1].Message, "Incomplete")
}
