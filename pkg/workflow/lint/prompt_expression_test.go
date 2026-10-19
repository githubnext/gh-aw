package lint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptExpressionPolicy(t *testing.T) {
	for _, tt := range []struct {
		name, prompt string
		lines        []int
	}{
		{"ordinary text", "A prompt without expressions.", nil},
		{"single line", "${{ github.repository }}", nil},
		{"multiline", "Heading\n${{\n github.repository\n}}", []int{2}},
		{"CRLF", "Heading\r\n${{\r\n github.repository\r\n}}", []int{2}},
		{"carriage return", "${{\r github.repository }}", []int{1}},
		{"newline inside string", "${{ format('first\nsecond') }}", []int{1}},
		{"quoted delimiter", "${{ format('}}{0}', github.repository) }}", nil},
		{"doubled quote", "${{ format('it''s }} {0}', github.repository) }}", nil},
		{"dot property", "Heading\n${{ inputs.value.constructor }}", []int{2}},
		{"bracket property", "${{ inputs['__proto__'] }}", []int{1}},
		{"nested property", "${{ format('{0}', inputs.value.prototype) }}", []int{1}},
		{"computed property inspected", "${{ inputs[inputs.value.constructor] }}", []int{1}},
		{"literal is not traversal", "${{ format('constructor __proto__') }}", nil},
		{"dynamic property not inferred", "${{ inputs[inputs.key] }}", nil},
		{"near miss", "${{ inputs.constructor_version }}", nil},
		{"frontmatter not prompt policy", "A safe prompt.", nil},
		{"code fences are still interpolated", "```\n${{ inputs.value.constructor }}\n```", []int{2}},
		{"all expressions", "${{ inputs.value.constructor }}\n${{\n github.repository\n}}", []int{1, 2}},
		{"two violations same expression", "${{\n inputs.value.constructor }}", []int{1, 1}},
		{"invalid expression", "Heading\n${{ PRIVATE_VALUE( }}", []int{2}},
		{"unterminated expression", "${{ PRIVATE_VALUE", []int{1}},
		{"earlier findings retained", "${{ inputs.constructor }}\n${{ PRIVATE_VALUE", []int{1, 2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, strict := range []string{"", "strict: false\n"} {
				input := policyTestInput(t, strict+"env: {VALUE: '${{ inputs.constructor }}'}")
				input.Document.Markdown = tt.prompt
				before, err := json.Marshal(input)
				require.NoError(t, err)
				runner, err := Default().Select("prompt-expression")
				require.NoError(t, err)
				diagnostics, err := runner.Run(context.Background(), input)
				require.NoError(t, err)
				var lines []int
				for _, diagnostic := range diagnostics {
					lines = append(lines, diagnostic.Line)
					assert.Equal(t, "prompt", diagnostic.Path)
					assert.Equal(t, Error, diagnostic.Severity)
					assert.Equal(t, "CTR-010", diagnostic.ThreatRuleID)
					assert.NotContains(t, diagnostic.Message, "PRIVATE_VALUE")
				}
				assert.Equal(t, tt.lines, lines)
				after, err := json.Marshal(input)
				require.NoError(t, err)
				assert.Equal(t, before, after)
			}
		})
	}
}

func TestPromptExpressionBlockedProperties(t *testing.T) {
	for _, property := range constants.DangerousPropertyNames {
		for _, expression := range []string{"inputs.value." + property, "inputs.value['" + property + "']"} {
			input := policyTestInput(t, "")
			input.Document.Markdown = "${{ " + expression + " }}"
			diagnostics, err := PromptExpressionRule().Check(context.Background(), input)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1, expression)
		}
	}
}

func TestPromptExpressionCancellation(t *testing.T) {
	input := policyTestInput(t, "")
	input.Document.Markdown = "${{ format('{0}', inputs.value) }}\n${{ inputs.constructor }}"
	ctx := &integrityCancelContext{Context: context.Background(), remaining: 5}
	_, err := PromptExpressionRule().Check(ctx, input)
	assert.ErrorIs(t, err, context.Canceled)
}
