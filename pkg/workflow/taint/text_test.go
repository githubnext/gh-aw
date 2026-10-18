package taint

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeTextMatchesExistingInterpolation(t *testing.T) {
	rule := SecretLeakRule()
	bindings := map[string][]Source{"env.bound": {{Reference: "secrets.BOUND", Path: "env.BOUND"}}}
	bound := scope{"env.bound": bindings["env.bound"]}
	for _, text := range []string{
		"plain $BOUND",
		"${{ secrets.TOKEN }}",
		"first\n${{ secrets.TOKEN }} and ${{ env.bound }}\n",
		"${{ format('it''s }} {0}', secrets.TOKEN) }} tail",
		"${{ format('${{literal}} {0}', secrets.TOKEN) }} ${{ secrets.OTHER }}",
		"${{\n secrets.TOKEN\n }}\n${{ env.bound }}",
		"${{ secrets['TOKEN'] }}",
		"${{ contains(secrets.TOKEN, 'x') && 'yes' || 'no' }}",
		"${{ toJSON(secrets) }}",
	} {
		t.Run(text, func(t *testing.T) {
			a := &analyzer{rule: rule}
			rendered, oldSources, err := a.interpolate(text, "run", bound, nil)
			require.NoError(t, err)
			assert.Equal(t, text, rendered)
			expressions, err := AnalyzeText(context.Background(), text, "run", rule, bindings)
			require.NoError(t, err)
			var sources value
			for _, expression := range expressions {
				sources = union(sources, expression.Sources)
			}
			assert.Equal(t, oldSources, sources)
		})
	}
}

func TestAnalyzeTextLinesAndReplacement(t *testing.T) {
	text := "prefix\n${{\n secrets.TOKEN\n }} and ${{ secrets.SECOND }}\n${{ secrets.THIRD }}"
	expressions, err := AnalyzeText(context.Background(), text, "run", SecretLeakRule(), nil)
	require.NoError(t, err)
	require.Len(t, expressions, 3)
	assert.Equal(t, []int{2, 4, 5}, []int{expressions[0].Line, expressions[1].Line, expressions[2].Line})
	a := &analyzer{rule: SecretLeakRule()}
	rendered, sources, err := a.interpolate(text, "run", nil, func(i int, _ value) string {
		return strings.Repeat("x", i+1)
	})
	require.NoError(t, err)
	assert.Equal(t, "prefix\nx and xx\nxxx", rendered)
	assert.Len(t, sources, 3)
}

func TestAnalyzeTextMalformedExpressionsAreValueFree(t *testing.T) {
	for _, broken := range []string{
		"${{ secrets.TOKEN",
		"${{ format('PRIVATE_VALUE', secrets.TOKEN) + }}",
		"${{ 'PRIVATE_VALUE }}",
		"${{ }}",
	} {
		text := "${{ secrets.FIRST }}\n\n" + broken
		expressions, err := AnalyzeText(context.Background(), text, "jobs.test.steps[0].run", SecretLeakRule(), nil)
		require.Error(t, err)
		var parseError *ExpressionError
		require.ErrorAs(t, err, &parseError)
		assert.Equal(t, 3, parseError.Line)
		assert.Equal(t, "jobs.test.steps[0].run", parseError.Path)
		assert.NotContains(t, err.Error(), "PRIVATE_VALUE")
		assert.NotContains(t, err.Error(), "TOKEN")
		require.Len(t, expressions, 1)
		assert.Equal(t, "secrets.first", expressions[0].Sources[0].Reference)
		a := &analyzer{rule: SecretLeakRule()}
		rendered, sources, oldErr := a.interpolate(text, parseError.Path, nil, nil)
		require.Error(t, oldErr)
		assert.Equal(t, err.Error(), oldErr.Error())
		assert.Empty(t, rendered)
		assert.Empty(t, sources)
	}
}

func TestAnalyzeTextCancellationAndBindings(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := AnalyzeText(ctx, "plain", "run", SecretLeakRule(), nil)
	require.ErrorIs(t, err, context.Canceled)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	rule := SecretLeakRule()
	rule.IsSource = func(string) bool {
		cancel()
		return true
	}
	_, err = AnalyzeText(ctx, "${{ secrets.TOKEN }}", "run", rule, nil)
	require.ErrorIs(t, err, context.Canceled)

	_, err = AnalyzeText(context.Background(), "", "run", Rule{}, nil)
	require.ErrorContains(t, err, "source predicate")
	bindings := map[string][]Source{"ENV.TITLE": {{Reference: "inputs.title", Path: "env.TITLE"}}}
	expressions, err := AnalyzeText(context.Background(), "${{ env['title'] }} $TITLE", "run", SecretLeakRule(), bindings)
	require.NoError(t, err)
	require.Len(t, expressions, 1)
	assert.Equal(t, bindings["ENV.TITLE"], expressions[0].Sources)
}
