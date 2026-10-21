package lint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
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
			source += `      - name: Resolve deferred prompt expressions
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
