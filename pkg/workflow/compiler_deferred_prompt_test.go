package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteDeferredPromptExpressions(t *testing.T) {
	jobs := map[string]any{
		"release": map[string]any{"needs": "activation", "outputs": map[string]any{"release_id": "public"}},
		"report":  map[string]any{"needs": "agent", "outputs": map[string]any{"value": "public"}},
	}
	for _, tt := range []struct {
		expression string
		before     []string
		deferred   bool
	}{
		{"needs.release.outputs.release_id", nil, true},
		{"needs.release.result", nil, true},
		{"needs.release.outputs.release_id", []string{"release"}, false},
		{"needs.release.outputs.unknown", nil, false},
		{"needs.unknown.outputs.value", nil, false},
		{"needs.report.outputs.value", nil, false},
		{"needs.agent.outputs.value", nil, false},
		{"needs.release.outputs.release_id || 'fallback'", nil, false},
		{"github.repository", nil, false},
	} {
		t.Run(tt.expression+strings.Join(tt.before, ","), func(t *testing.T) {
			original := &ExpressionMapping{Original: "${{ " + tt.expression + " }}", Content: tt.expression, EnvVar: "GH_AW_VALUE"}
			got, deferred := rewriteDeferredPromptExpressions([]*ExpressionMapping{original, original}, jobs, tt.before, original.Original)
			require.Len(t, got, 2)
			assert.Equal(t, tt.expression, original.Content, "do not mutate the original mappings")
			if tt.deferred {
				require.Len(t, deferred, 1)
				assert.Equal(t, tt.expression, deferred[0].Content)
				assert.Equal(t, "'__"+deferred[0].EnvVar+"__'", got[0].Content)
				assert.Equal(t, original.EnvVar, got[0].EnvVar)
			} else {
				assert.Empty(t, deferred)
				assert.Equal(t, original, got[0])
			}
		})
	}
}

func TestDeferredPromptDoesNotChangeRendering(t *testing.T) {
	jobs := map[string]any{"release": map[string]any{
		"needs": "activation", "outputs": map[string]any{"release_id": "public"},
	}}
	for _, markdown := range []string{
		"${{ needs.release.outputs.release_id || 'fallback' }}",
		"{{#if ${{ needs.release.outputs.release_id }} }}yes{{/if}}",
		"{{#if needs.release.outputs.release_id}}yes{{/if}}",
	} {
		extractor := NewExpressionExtractor()
		mappings, err := extractor.ExtractExpressions(wrapExpressionsInTemplateConditionals(markdown))
		require.NoError(t, err)
		_, deferred := rewriteDeferredPromptExpressions(mappings, jobs, nil, markdown)
		assert.Empty(t, deferred, markdown)
	}
}

func TestDeferredPromptCompilation(t *testing.T) {
	for _, inline := range []bool{false, true} {
		t.Run(map[bool]string{false: "runtime", true: "inline"}[inline], func(t *testing.T) {
			dir := t.TempDir()
			source := `---
on: workflow_dispatch
engine: copilot
jobs:
  release:
    needs: activation
    runs-on: ubuntu-latest
    outputs:
      release_id: ${{ steps.release.outputs.id }}
    steps:
      - id: release
        run: echo "id=123" >> "$GITHUB_OUTPUT"
---
Release ID: ${{ needs.release.outputs.release_id }}
`
			path := filepath.Join(dir, "release.md")
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			compiler := NewCompiler()
			compiler.SetQuiet(true)
			compiler.inlinePrompt = inline
			require.NoError(t, compiler.CompileWorkflow(path))
			content, err := os.ReadFile(filepath.Join(dir, "release.lock.yml"))
			require.NoError(t, err)
			lock := string(content)
			agent := strings.Index(lock, "\n  agent:")
			require.Positive(t, agent)
			activation := lock[:agent]
			assert.NotContains(t, activation, "${{ needs.release.outputs.release_id }}")
			assert.Contains(t, activation, "GH_AW_DEFERRED_PLACEHOLDERS:")
			assert.Contains(t, lock[agent:], "Resolve deferred prompt expressions")
			assert.Contains(t, lock[agent:], "GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt")
			assert.Contains(t, lock[agent:], "${{ needs.release.outputs.release_id }}")
			assert.Less(t, strings.Index(lock[agent:], "Download activation artifact"), strings.Index(lock[agent:], "Resolve deferred prompt expressions"))
		})
	}
}
