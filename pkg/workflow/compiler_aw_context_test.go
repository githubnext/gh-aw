//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectAwContextInput(t *testing.T) {
	tests := []struct {
		name            string
		onSection       string
		expectContains  []string
		expectAbsent    []string
		expectUnchanged bool
	}{
		{
			name: "no workflow_dispatch - no injection",
			onSection: `on:
  push:
  pull_request:`,
			expectUnchanged: true,
		},
		{
			name: "workflow_dispatch without inputs - injects aw_context",
			onSection: `on:
  workflow_dispatch:`,
			expectContains: []string{
				"aw_context",
				awContextInputDescription,
				"required: false",
			},
		},
		{
			name: "workflow_dispatch with existing inputs - appends aw_context",
			onSection: `on:
  workflow_dispatch:
    inputs:
      my_param:
        description: User parameter
        type: string`,
			expectContains: []string{
				"my_param",
				"aw_context",
				awContextInputDescription,
			},
		},
		{
			name: "aw_context already present - no duplicate injection",
			onSection: `on:
  workflow_dispatch:
    inputs:
      aw_context:
        description: already here
        type: string`,
			expectContains: []string{"aw_context", "already here"},
			expectAbsent:   []string{awContextInputDescription},
		},
		{
			name: "workflow_call and workflow_dispatch combined - only dispatch gets aw_context",
			onSection: `on:
  workflow_call:
  workflow_dispatch:`,
			expectContains: []string{
				"aw_context",
				"workflow_call",
			},
		},
		{
			name: "push and workflow_dispatch - injects aw_context",
			onSection: `on:
  push:
  workflow_dispatch:`,
			expectContains: []string{"aw_context"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectAwContextInput(tt.onSection)

			if tt.expectUnchanged {
				assert.Equal(t, tt.onSection, result, "expected on section to be unchanged")
				return
			}

			for _, want := range tt.expectContains {
				assert.Contains(t, result, want, "expected result to contain %q", want)
			}
			for _, absent := range tt.expectAbsent {
				assert.NotContains(t, result, absent, "expected result NOT to contain %q", absent)
			}
		})
	}
}

func TestInjectAwContextInputIdempotent(t *testing.T) {
	onSection := `on:
  workflow_dispatch:
    inputs:
      my_param:
        description: User parameter
        type: string`

	first := injectAwContextInput(onSection)
	second := injectAwContextInput(first)

	require.Equal(t, first, second, "repeated injection should be idempotent")
}

func TestInjectAwContextInputInputType(t *testing.T) {
	onSection := `on:
  workflow_dispatch:`

	result := injectAwContextInput(onSection)

	assert.Contains(t, result, "aw_context", "should contain aw_context")
	assert.Contains(t, result, "type: string", "aw_context should be of type string")
	assert.Contains(t, result, "required: false", "aw_context should be optional")
}

func TestInjectAwContextInputPreservesCronExpressions(t *testing.T) {
	onSection := `"on":
  issues:
    types:
    - opened
    - closed
  schedule:
  - cron: "0 8 * * *"
  workflow_dispatch:`

	result := injectAwContextInput(onSection)

	// The cron expression must be preserved with quotes after re-marshaling
	assert.Contains(t, result, `cron: "0 8 * * *"`, "cron expression should be preserved with quotes")
	// aw_context should be injected
	assert.Contains(t, result, "aw_context", "aw_context should be injected")
}
