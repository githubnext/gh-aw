//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── injectAwContextIntoWorkflowCallYAML ───────────────────────────────────

func TestInjectAwContextIntoWorkflowCallYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSubs []string // substrings that must appear in result
		noSubs   []string // substrings that must NOT appear in result
	}{
		{
			name: "no workflow_call trigger – no injection",
			input: `on:
  workflow_dispatch:
    inputs:
      foo:
        type: string`,
			noSubs: []string{"workflow_call"},
		},
		{
			name: "bare workflow_call – adds inputs and aw_context",
			input: `on:
  workflow_call:`,
			wantSubs: []string{
				"workflow_call:",
				"inputs:",
				AwContextInputName + ":",
				"type: string",
				"required: false",
			},
		},
		{
			name: "workflow_call with null value – adds inputs and aw_context",
			input: `on:
  workflow_call: null`,
			wantSubs: []string{
				"inputs:",
				AwContextInputName + ":",
			},
		},
		{
			name: "workflow_call with existing inputs – aw_context added first",
			input: `on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false`,
			wantSubs: []string{
				AwContextInputName + ":",
				"payload:",
				"type: string",
			},
		},
		{
			name: "idempotent – aw_context not duplicated on second call",
			input: `on:
  workflow_call:
    inputs:
      aw_context:
        default: ""
        type: string`,
			// No duplicate should be added
			wantSubs: []string{AwContextInputName + ":"},
		},
		{
			name: "both workflow_dispatch and workflow_call – both injected",
			input: `on:
  workflow_dispatch:
  workflow_call:`,
			wantSubs: []string{
				"workflow_dispatch:",
				"workflow_call:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectAwContextIntoWorkflowCallYAML(tt.input)

			for _, sub := range tt.wantSubs {
				assert.True(t, strings.Contains(result, sub),
					"expected %q to contain %q, got:\n%s", tt.name, sub, result)
			}
			for _, sub := range tt.noSubs {
				assert.False(t, strings.Contains(result, sub),
					"expected %q NOT to contain %q, got:\n%s", tt.name, sub, result)
			}
		})
	}
}

func TestInjectAwContextIntoWorkflowCallYAML_Idempotent(t *testing.T) {
	input := `on:
  workflow_call:`
	first := injectAwContextIntoWorkflowCallYAML(input)
	second := injectAwContextIntoWorkflowCallYAML(first)
	assert.Equal(t, first, second, "calling injection twice should be idempotent")
	// aw_context: should appear exactly once
	count := strings.Count(second, AwContextInputName+":")
	assert.Equal(t, 1, count, "aw_context: should appear exactly once after double injection")
}

func TestInjectAwContextIntoWorkflowCallYAML_PreservesExisting(t *testing.T) {
	input := `on:
  workflow_call:
    inputs:
      payload:
        description: The agent payload
        type: string
        required: false`
	result := injectAwContextIntoWorkflowCallYAML(input)

	// Original inputs preserved
	assert.Contains(t, result, "payload:", "original payload input should be preserved")
	assert.Contains(t, result, "The agent payload", "original description should be preserved")

	// aw_context added
	assert.Contains(t, result, AwContextInputName+":", "aw_context input should be injected")
}
