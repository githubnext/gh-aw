package lint

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/rhysd/actionlint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptNeedsAvailability(t *testing.T) {
	for _, tt := range []struct {
		name, config, prompt string
		finding              bool
	}{
		{"post activation", "jobs: {release: {needs: activation, outputs: {release_id: public}}}", "${{ needs.release.outputs.release_id }}", true},
		{"brackets and case", "jobs: {release: {needs: activation, outputs: {release_id: public}}}", "${{ NEEDS['RELEASE']['OUTPUTS']['RELEASE_ID'] }}", true},
		{"compound", "jobs: {release: {needs: activation, outputs: {release_id: public}}}", "${{ inputs.override || needs.release.outputs.release_id }}", true},
		{"no explicit dependencies", "jobs: {release: {outputs: {release_id: public}}}", "${{ needs.release.outputs.release_id }}", false},
		{"before activation", "jobs: {release: {needs: pre_activation, outputs: {release_id: public}}}", "${{ needs.release.outputs.release_id }}", false},
		{"on needs", "on: {needs: [release]}\njobs: {release: {needs: prepare, outputs: {release_id: public}}, prepare: {}}", "${{ needs.release.outputs.release_id }}", false},
		{"unknown output", "jobs: {release: {needs: pre_activation, outputs: {release_id: public}}}", "${{ needs.release.outputs.PRIVATE_VALUE }}", true},
		{"unknown job", "", "${{ needs.PRIVATE_VALUE.outputs.value }}", true},
		{"imported job", "imports: [shared.md]", "${{ needs.imported.outputs.value }}", false},
		{"quoted text", "", "${{ 'needs.PRIVATE_VALUE.outputs.value' }}", false},
		{"ordinary text", "", "needs.PRIVATE_VALUE.outputs.value", false},
		{"bare if unavailable", "jobs: {release: {needs: activation, outputs: {release_id: public}}}", "{{#if needs.release.outputs.release_id}}yes{{/if}}", true},
		{"bare elseif unavailable", "jobs: {release: {needs: activation, outputs: {release_id: public}}}", "{{#if false}}no{{#elseif needs.release.outputs.release_id}}yes{{/if}}", true},
		{"bare if unknown output", "jobs: {release: {outputs: {release_id: public}}}", "{{#if needs.release.outputs.PRIVATE_VALUE}}yes{{/if}}", true},
		{"bare elseif unknown output", "jobs: {release: {outputs: {release_id: public}}}", "{{#if false}}no{{elseif needs.release.outputs.PRIVATE_VALUE}}yes{{/if}}", true},
		{"bare else-if unknown job", "", "{{#else-if NEEDS['PRIVATE_VALUE'].outputs.value}}", true},
		{"bare else_if before activation", "jobs: {release: {outputs: {release_id: public}}}", "{{else_if needs.release.outputs.release_id}}", false},
		{"bare condition compound", "", "{{#if inputs.override || needs.PRIVATE_VALUE.outputs.value}}", true},
		{"bare condition literal", "", "{{#if 'needs.PRIVATE_VALUE.outputs.value'}}", false},
		{"bare condition environment", "", "{{#if ${needs.PRIVATE_VALUE.outputs.value}}}", false},
		{"bare condition placeholder", "", "{{#elseif __needs.PRIVATE_VALUE.outputs.value__}}", false},
		{"pre activation", "", "${{ needs.pre_activation.outputs.activated }}", false},
		{"sanitized", "", "${{ needs.activation.outputs.text }}", false},
		{"scalar trigger", "on: issues", "Ordinary prompt", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := policyTestInput(t, tt.config)
			input.Workflow = nil
			input.Document.Markdown = "Heading\n" + tt.prompt
			before, err := json.Marshal(input)
			require.NoError(t, err)
			diagnostics, err := NeedsConsistencyRule().Check(context.Background(), input)
			require.NoError(t, err)
			if tt.finding {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, "prompt", diagnostics[0].Path)
				assert.Equal(t, 2, diagnostics[0].Line)
				assert.NotContains(t, diagnostics[0].Message, "PRIVATE_VALUE")
			} else {
				assert.Empty(t, diagnostics)
			}
			after, err := json.Marshal(input)
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}

func TestPromptNeedsCompiledGraph(t *testing.T) {
	for _, late := range []bool{false, true} {
		input := policyTestInput(t, "jobs: {release: {needs: pre_activation, outputs: {release_id: public}}}")
		input.Document.Markdown = "${{ needs.release.outputs.release_id }}"
		source := `on: workflow_dispatch
jobs:
  activation:
    runs-on: ubuntu-latest
    steps: [{run: echo activation}]
  release:
    needs: activation
    runs-on: ubuntu-latest
    outputs: {release_id: public}
    steps: [{run: echo release}]
  agent:
    needs: [activation, release]
    runs-on: ubuntu-latest
    steps:
`
		if late {
			source += `      - id: ` + string(constants.DeferredPromptStepID) + `
        name: ` + constants.DeferredPromptStepName + `
        uses: actions/github-script@0123456789abcdef0123456789abcdef01234567
        env:
          GH_AW_DEFERRED_1234: ${{ needs.release.outputs.release_id }}
        with:
          script: |
            const substitutePlaceholders = require('./substitute_placeholders.cjs');
            await substitutePlaceholders({ file: process.env.GH_AW_PROMPT, substitutions: { GH_AW_DEFERRED_1234: process.env.GH_AW_DEFERRED_1234 } });
`
		} else {
			source += "      - run: echo agent\n"
		}
		workflow, err := actions.Parse([]byte(source))
		require.NoError(t, err)
		input.Workflow = workflow
		diagnostics, err := NeedsConsistencyRule().Check(context.Background(), input)
		require.NoError(t, err)
		if late {
			assert.Empty(t, diagnostics)
		} else {
			require.Len(t, diagnostics, 1, "compiled graph must override authored dependencies")
			assert.Equal(t, "prompt", diagnostics[0].Path)
		}
	}
}

func TestPromptNeedsMalformedExpressionsPreserveGraphDiagnostics(t *testing.T) {
	for _, prompt := range []string{
		"${{ needs.release.outputs. }}",
		"{{#if needs.release.outputs.}}yes{{/if}}",
		"{{#elseif needs.release.outputs.}}yes{{/if}}",
	} {
		t.Run(prompt, func(t *testing.T) {
			input := policyTestInput(t, "jobs: {build: {needs: missing}}")
			input.Workflow = nil
			input.Document.Markdown = prompt
			diagnostics, err := NeedsConsistencyRule().Check(context.Background(), input)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			assert.Equal(t, "jobs.build.needs", diagnostics[0].Path)
		})
	}
	input := policyTestInput(t, "jobs: {build: {needs: missing}}")
	input.Workflow = nil
	input.Document.Markdown = "${{ needs.release.outputs. }}"
	diagnostics, err := PromptExpressionRule().Check(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1, "prompt-expression retains ownership of syntax diagnostics")
}

func TestPromptNeedsMalformedConditionRetainsLaterChecks(t *testing.T) {
	for _, prompt := range []string{
		"{{#if needs.broken.}}\nignored\n{{#elseif needs.unknown.outputs.value}}",
		"{{#if needs.broken.\nignored\n{{#elseif needs.unknown.outputs.value}}",
	} {
		input := policyTestInput(t, "")
		input.Workflow = nil
		input.Document.Markdown = prompt
		diagnostics, err := NeedsConsistencyRule().Check(context.Background(), input)
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, 3, diagnostics[0].Line)
	}
}

func TestPromptNeedsConditionsCancellation(t *testing.T) {
	for _, prompt := range []string{
		"{{#if needs.release.outputs.release_id}}",
		"{{#elseif needs.release.outputs.release_id}}",
		"{{#if needs.release.outputs.}}",
	} {
		t.Run(prompt, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := visitPromptConditions(ctx, prompt, func(_ int, _ actionlint.ExprNode) error {
				t.Fatal("cancelled visitor must not run")
				return nil
			})
			require.ErrorIs(t, err, context.Canceled)
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	err := visitPromptConditions(ctx, "{{#if needs.release.outputs.release_id}}", func(_ int, _ actionlint.ExprNode) error {
		cancel()
		return ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)
	failure := errors.New("visitor failed")
	err = visitPromptConditions(context.Background(), "{{#elseif needs.release.outputs.release_id}}", func(_ int, _ actionlint.ExprNode) error {
		return failure
	})
	require.ErrorIs(t, err, failure)
}

func TestPromptNeedsDeferredBindingContract(t *testing.T) {
	for _, tt := range []struct {
		name, id, key, value, prompt string
		finding                      bool
	}{
		{"stable id", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "${{ needs.release.outputs.release_id }}", false},
		{"wrong id", "custom", "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "${{ needs.release.outputs.release_id }}", true},
		{"missing id", "", "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "${{ needs.release.outputs.release_id }}", true},
		{"wrong binding key", string(constants.DeferredPromptStepID), "OTHER_VALUE", "${{ needs.release.outputs.release_id }}", "${{ needs.release.outputs.release_id }}", true},
		{"text binding", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "prefix ${{ needs.release.outputs.release_id }}", "${{ needs.release.outputs.release_id }}", true},
		{"compound binding", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id || 'fallback' }}", "${{ needs.release.outputs.release_id }}", true},
		{"compound prompt", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "${{ needs.release.outputs.release_id || 'fallback' }}", true},
		{"bare if prompt", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "{{#if needs.release.outputs.release_id}}", true},
		{"bare elseif prompt", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "{{#elseif needs.release.outputs.release_id}}", true},
		{"wrapped if prompt", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "{{#if ${{ needs.release.outputs.release_id }} }}", true},
		{"wrapped elseif prompt", string(constants.DeferredPromptStepID), "GH_AW_DEFERRED_1234", "${{ needs.release.outputs.release_id }}", "{{#elseif ${{ needs.release.outputs.release_id }} }}", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := policyTestInput(t, "")
			input.Document.Markdown = tt.prompt
			source := `on: workflow_dispatch
jobs:
  activation:
    runs-on: ubuntu-latest
    steps: [{run: echo activation}]
  release:
    needs: activation
    runs-on: ubuntu-latest
    outputs: {release_id: public}
    steps: [{run: echo release}]
  agent:
    needs: [activation, release]
    runs-on: ubuntu-latest
    steps:
      - id: "` + tt.id + `"
        name: ` + constants.DeferredPromptStepName + `
        env:
          ` + tt.key + `: ` + tt.value + `
        run: echo resolver implementation may change
`
			if tt.id == "" {
				source = strings.Replace(source, "- id: \"\"\n        name:", "- name:", 1)
			}
			workflow, err := actions.Parse([]byte(source))
			require.NoError(t, err)
			input.Workflow = workflow
			diagnostics, err := NeedsConsistencyRule().Check(context.Background(), input)
			require.NoError(t, err)
			if tt.finding {
				require.Len(t, diagnostics, 1)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestPromptNeedsReportsEachLine(t *testing.T) {
	input := policyTestInput(t, "jobs: {release: {needs: activation, outputs: {release_id: public}}}")
	input.Workflow = nil
	input.Document.Markdown = "${{ needs.release.outputs.release_id }}\n${{ needs.release.outputs.release_id }}"
	diagnostics, err := NeedsConsistencyRule().Check(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, 1, diagnostics[0].Line)
	assert.Equal(t, 2, diagnostics[1].Line)
}
