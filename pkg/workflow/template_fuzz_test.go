package workflow

import (
	"strings"
	"testing"
)

// FuzzWrapExpressionsInTemplateConditionals performs fuzz testing on the template
// conditional expression wrapper to ensure it handles all inputs without panicking
// and correctly wraps/preserves expressions.
//
// The fuzzer validates that:
// 1. The function never panics on any input
// 2. GitHub expressions are properly wrapped in ${{ }}
// 3. Already-wrapped expressions are preserved
// 4. Environment variables (${...}) are not wrapped
// 5. Placeholder references (__...) are not wrapped
// 6. Empty expressions are wrapped as ${{ false }}
// 7. Malformed input is handled gracefully
func FuzzWrapExpressionsInTemplateConditionals(f *testing.F) {
	// Seed corpus with typical GitHub expressions
	f.Add("{{#if github.event.issue.number}}content{{/if}}")
	f.Add("{{#if github.actor}}content{{/if}}")
	f.Add("{{#if github.repository}}content{{/if}}")
	f.Add("{{#if needs.activation.outputs.text}}content{{/if}}")
	f.Add("{{#if steps.my-step.outputs.result}}content{{/if}}")
	f.Add("{{#if env.MY_VAR}}content{{/if}}")

	// Already wrapped expressions (should be preserved)
	f.Add("{{#if ${{ github.event.issue.number }} }}content{{/if}}")
	f.Add("{{#if ${{ github.actor }}}}content{{/if}}")

	// Environment variables (should not be wrapped)
	f.Add("{{#if ${GH_AW_EXPR_D892F163}}}content{{/if}}")
	f.Add("{{#if ${GH_AW_EXPR_ABC123}}}content{{/if}}")

	// Placeholder references (should not be wrapped)
	f.Add("{{#if __PLACEHOLDER__}}content{{/if}}")
	f.Add("{{#if __VAR_123__}}content{{/if}}")

	// Empty expressions
	f.Add("{{#if }}content{{/if}}")
	f.Add("{{#if   }}content{{/if}}")
	f.Add("{{#if\t}}content{{/if}}")

	// Literal values
	f.Add("{{#if true}}content{{/if}}")
	f.Add("{{#if false}}content{{/if}}")
	f.Add("{{#if 0}}content{{/if}}")
	f.Add("{{#if 1}}content{{/if}}")

	// Multiple conditionals
	f.Add("{{#if github.actor}}first{{/if}}\n{{#if github.repository}}second{{/if}}")
	f.Add("{{#if github.actor}}A{{/if}} {{#if github.repository }}B{{/if}} {{#if ${{ github.ref }} }}C{{/if}}")

	// Edge cases with whitespace
	f.Add("{{#if github.actor }}content{{/if}}")
	f.Add("{{#if  github.actor  }}content{{/if}}")
	f.Add("  {{#if github.actor}}content{{/if}}")
	f.Add("{{#if\tgithub.actor}}content{{/if}}")

	// Malformed inputs
	f.Add("{{#if github.actor}}")
	f.Add("{{/if}}")
	f.Add("{{#if")
	f.Add("}}")
	f.Add("{{#if }}{{#if }}")

	// Nested braces
	f.Add("{{#if ${{ ${{ github.actor }} }} }}content{{/if}}")
	f.Add("{{#if {github.actor}}}content{{/if}}")

	// Special characters
	f.Add("{{#if github.actor!}}content{{/if}}")
	f.Add("{{#if github-actor}}content{{/if}}")
	f.Add("{{#if github.actor.value}}content{{/if}}")

	// Unicode and control characters
	f.Add("{{#if github.actor™}}content{{/if}}")
	f.Add("{{#if github.actor\n}}content{{/if}}")
	f.Add("{{#if github.actor\x00}}content{{/if}}")

	// Very long expressions
	longExpr := "{{#if "
	for i := 0; i < 100; i++ {
		longExpr += "github.event.pull_request.head.repo."
	}
	longExpr += "name}}content{{/if}}"
	f.Add(longExpr)

	// Complex markdown structures
	f.Add(`# Header
{{#if github.actor}}
## Conditional section
{{/if}}`)
	f.Add("{{#if github.actor}}**bold**{{/if}}")
	f.Add("{{#if github.actor}}\n- list\n- items\n{{/if}}")

	// Mixed valid and edge cases
	f.Add("Before {{#if github.actor}}middle{{/if}} after")
	f.Add("{{#if github.actor}}{{#if github.repository}}nested{{/if}}{{/if}}")

	f.Fuzz(func(t *testing.T, input string) {
		// The fuzzer will generate variations of the seed corpus
		// and random strings to test the wrapper

		// This should never panic, even on malformed input
		result := wrapExpressionsInTemplateConditionals(input)

		// Basic sanity checks
		if result == "" && input != "" {
			// Result should not be empty if input is not empty
			// (unless the input somehow gets completely removed, which shouldn't happen)
			t.Errorf("wrapExpressionsInTemplateConditionals returned empty string for non-empty input")
		}

		// If the input contains {{#if with a non-empty, non-special expression,
		// the result should contain ${{ }} wrapping
		if strings.Contains(input, "{{#if github.") && !strings.Contains(input, "${{") {
			if !strings.Contains(result, "${{") {
				t.Errorf("Expected result to contain ${{ }} wrapping for GitHub expression, input: %q, result: %q", input, result)
			}
		}

		// If the input contains already wrapped expressions, they should be preserved
		if strings.Contains(input, "${{ github.") {
			if !strings.Contains(result, "${{ github.") {
				t.Errorf("Already wrapped expression should be preserved, input: %q, result: %q", input, result)
			}
		}

		// If the input contains environment variables, they should not be wrapped with ${{ }}
		if strings.Contains(input, "${GH_AW_EXPR_") {
			// Count occurrences before and after
			beforeCount := strings.Count(input, "${GH_AW_EXPR_")
			afterCount := strings.Count(result, "${GH_AW_EXPR_")
			if beforeCount != afterCount {
				t.Errorf("Environment variable references should not be modified, input: %q, result: %q", input, result)
			}
		}

		// If the input contains placeholder references, they should not be wrapped with ${{ }}
		if strings.Contains(input, "__") && strings.Contains(input, "{{#if __") {
			// The result should still contain the __ prefix in the conditional
			if !strings.Contains(result, "{{#if __") {
				t.Errorf("Placeholder references should not be wrapped, input: %q, result: %q", input, result)
			}
		}

		// If the input has empty expression {{#if }}, it should be wrapped as ${{ false }}
		if strings.Contains(input, "{{#if }}") || strings.Contains(input, "{{#if   }}") || strings.Contains(input, "{{#if\t}}") {
			if !strings.Contains(result, "${{ false }}") {
				t.Errorf("Empty expression should be wrapped as ${{ false }}, input: %q, result: %q", input, result)
			}
		}
	})
}
