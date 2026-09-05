//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTopLevelEnvExpressions(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]any
		errorContains []string
	}{
		{
			name: "allows contexts available at workflow scope",
			env: map[string]any{
				"REPOSITORY": "${{ github.repository }}",
				"INPUT":      "${{ inputs.target }}",
				"SETTING":    "${{ vars.SETTING }}",
				"TOKEN":      "${{ secrets.TOKEN }}",
			},
		},
		{
			name: "allows plain text mentioning a job context",
			env: map[string]any{
				"TEXT": "runner.temp is unavailable here",
			},
		},
		{
			name: "ignores job context names in expression string literals",
			env: map[string]any{
				"TEXT": "${{ inputs.value || 'runner.temp' }}",
			},
		},
		{
			name: "allows non-string values",
			env: map[string]any{
				"COUNT":   2,
				"ENABLED": true,
			},
		},
		{
			name: "rejects runner context",
			env: map[string]any{
				"PLAYWRIGHT_BROWSERS_PATH": "${{ runner.temp }}/gh-aw/playwright-browsers",
			},
			errorContains: []string{
				"Validation failed for field 'env.PLAYWRIGHT_BROWSERS_PATH'",
				"unavailable outside jobs: runner",
				"Move this environment variable to a job or step env block",
			},
		},
		{
			name: "rejects bracket notation",
			env: map[string]any{
				"RUNNER_OS": "${{ runner['os'] }}",
			},
			errorContains: []string{"unavailable outside jobs: runner"},
		},
		{
			name: "reports multiple unavailable contexts deterministically",
			env: map[string]any{
				"VALUE": "${{ matrix.os || needs.prepare.outputs.os || runner.os }}",
			},
			errorContains: []string{"unavailable outside jobs: matrix, needs, runner"},
		},
		{
			name: "reports the first invalid environment variable alphabetically",
			env: map[string]any{
				"Z_VALUE": "${{ runner.os }}",
				"A_VALUE": "${{ matrix.os }}",
			},
			errorContains: []string{
				"Validation failed for field 'env.A_VALUE'",
				"unavailable outside jobs: matrix",
			},
		},
		{
			name: "does not confuse a property name with a context",
			env: map[string]any{
				"VALUE": "${{ github.event.runner }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTopLevelEnvExpressions(tt.env)
			if len(tt.errorContains) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, expected := range tt.errorContains {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestParseWorkflowRejectsUnavailableTopLevelEnvContext(t *testing.T) {
	workflow := `---
name: Invalid top-level env
on: workflow_dispatch
engine: copilot
env:
  PLAYWRIGHT_BROWSERS_PATH: ${{ runner.temp }}/gh-aw/playwright-browsers
---

# Test
`

	compiler := NewCompiler(WithNoEmit(true))
	workflowData, err := compiler.ParseWorkflowString(workflow, "test.md")
	require.NoError(t, err)

	_, err = compiler.CompileToYAML(workflowData, "test.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env.PLAYWRIGHT_BROWSERS_PATH")
	assert.Contains(t, err.Error(), "unavailable outside jobs: runner")
}
