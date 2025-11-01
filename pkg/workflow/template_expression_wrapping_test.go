package workflow

import (
	"testing"
)

func TestWrapExpressionsInTemplateConditionals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple github.event expression",
			input:    "{{#if github.event.issue.number}}content{{/if}}",
			expected: "{{#if ${{ github.event.issue.number }} }}content{{/if}}",
		},
		{
			name:     "github.actor expression",
			input:    "{{#if github.actor}}content{{/if}}",
			expected: "{{#if ${{ github.actor }} }}content{{/if}}",
		},
		{
			name:     "github.repository expression",
			input:    "{{#if github.repository}}content{{/if}}",
			expected: "{{#if ${{ github.repository }} }}content{{/if}}",
		},
		{
			name:     "needs. expression",
			input:    "{{#if needs.activation.outputs.text}}content{{/if}}",
			expected: "{{#if ${{ needs.activation.outputs.text }} }}content{{/if}}",
		},
		{
			name:     "steps. expression",
			input:    "{{#if steps.my-step.outputs.result}}content{{/if}}",
			expected: "{{#if ${{ steps.my-step.outputs.result }} }}content{{/if}}",
		},
		{
			name:     "env. expression",
			input:    "{{#if env.MY_VAR}}content{{/if}}",
			expected: "{{#if ${{ env.MY_VAR }} }}content{{/if}}",
		},
		{
			name:     "already wrapped expression",
			input:    "{{#if ${{ github.event.issue.number }} }}content{{/if}}",
			expected: "{{#if ${{ github.event.issue.number }} }}content{{/if}}",
		},
		{
			name:     "literal true value (wrapped)",
			input:    "{{#if true}}content{{/if}}",
			expected: "{{#if ${{ true }} }}content{{/if}}",
		},
		{
			name:     "literal false value (wrapped)",
			input:    "{{#if false}}content{{/if}}",
			expected: "{{#if ${{ false }} }}content{{/if}}",
		},
		{
			name:     "literal string value (wrapped)",
			input:    "{{#if some_literal}}content{{/if}}",
			expected: "{{#if ${{ some_literal }} }}content{{/if}}",
		},
		{
			name:     "multiple conditionals",
			input:    "{{#if github.actor}}first{{/if}}\n{{#if github.repository}}second{{/if}}",
			expected: "{{#if ${{ github.actor }} }}first{{/if}}\n{{#if ${{ github.repository }} }}second{{/if}}",
		},
		{
			name:     "mixed wrapped and unwrapped",
			input:    "{{#if github.actor}}first{{/if}}\n{{#if ${{ github.repository }} }}second{{/if}}",
			expected: "{{#if ${{ github.actor }} }}first{{/if}}\n{{#if ${{ github.repository }} }}second{{/if}}",
		},
		{
			name:     "expression with extra whitespace",
			input:    "{{#if   github.event.issue.number  }}content{{/if}}",
			expected: "{{#if ${{ github.event.issue.number }} }}content{{/if}}",
		},
		{
			name: "multiline content with multiple conditionals",
			input: `# Test Template

{{#if github.event.issue.number}}
This should be shown if there's an issue number.
{{/if}}

{{#if github.actor}}
This should be shown if there's an actor.
{{/if}}

Normal content here.`,
			expected: `# Test Template

{{#if ${{ github.event.issue.number }} }}
This should be shown if there's an issue number.
{{/if}}

{{#if ${{ github.actor }} }}
This should be shown if there's an actor.
{{/if}}

Normal content here.`,
		},
		{
			name:     "complex github.event path",
			input:    "{{#if github.event.pull_request.number}}content{{/if}}",
			expected: "{{#if ${{ github.event.pull_request.number }} }}content{{/if}}",
		},
		{
			name:     "github.run_id expression",
			input:    "{{#if github.run_id}}content{{/if}}",
			expected: "{{#if ${{ github.run_id }} }}content{{/if}}",
		},
		{
			name:     "environment variable reference (should not be wrapped)",
			input:    "{{#if ${GH_AW_EXPR_D892F163}}}content{{/if}}",
			expected: "{{#if ${GH_AW_EXPR_D892F163}}}content{{/if}}",
		},
		{
			name:     "multiple environment variable references",
			input:    "{{#if ${GH_AW_EXPR_ABC123}}}first{{/if}}\n{{#if ${GH_AW_EXPR_DEF456}}}second{{/if}}",
			expected: "{{#if ${GH_AW_EXPR_ABC123}}}first{{/if}}\n{{#if ${GH_AW_EXPR_DEF456}}}second{{/if}}",
		},
		{
			name:     "mixed github expression and env var reference",
			input:    "{{#if github.actor}}first{{/if}}\n{{#if ${GH_AW_EXPR_ABC123}}}second{{/if}}",
			expected: "{{#if ${{ github.actor }} }}first{{/if}}\n{{#if ${GH_AW_EXPR_ABC123}}}second{{/if}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapExpressionsInTemplateConditionals(tt.input)
			if result != tt.expected {
				t.Errorf("wrapExpressionsInTemplateConditionals() = %q, want %q", result, tt.expected)
			}
		})
	}
}
