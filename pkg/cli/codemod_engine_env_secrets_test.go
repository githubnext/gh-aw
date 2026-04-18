//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineEnvSecretsCodemod(t *testing.T) {
	codemod := getEngineEnvSecretsCodemod()

	t.Run("removes unsafe secret-bearing engine env keys and keeps allowed override", func(t *testing.T) {
		content := `---
on: workflow_dispatch
engine:
  id: codex
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    OPENAI_BASE_URL: ${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1
---
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"engine": map[string]any{
				"id": "codex",
				"env": map[string]any{
					"OPENAI_API_KEY":  "${{ secrets.OPENAI_API_KEY }}",
					"OPENAI_BASE_URL": "${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}", "allowed engine secret override should remain")
		assert.NotContains(t, result, "OPENAI_BASE_URL: ${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1", "unsafe engine env secret should be removed")
	})

	t.Run("removes empty env block after deleting unsafe entries", func(t *testing.T) {
		content := `---
on: workflow_dispatch
engine:
  id: codex
  env:
    OPENAI_BASE_URL: ${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1
---
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"engine": map[string]any{
				"id": "codex",
				"env": map[string]any{
					"OPENAI_BASE_URL": "${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.NotContains(t, result, "\n  env:\n", "empty env block should be removed")
		assert.Contains(t, result, "id: codex", "engine block should remain")
	})

	t.Run("no-op when only allowed engine secret override is used", func(t *testing.T) {
		content := `---
on: workflow_dispatch
engine:
  id: copilot
  env:
    COPILOT_GITHUB_TOKEN: ${{ secrets.MY_ORG_COPILOT_TOKEN }}
---
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"engine": map[string]any{
				"id": "copilot",
				"env": map[string]any{
					"COPILOT_GITHUB_TOKEN": "${{ secrets.MY_ORG_COPILOT_TOKEN }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not error in no-op case")
		assert.False(t, applied, "codemod should not apply")
		assert.Equal(t, content, result, "content should be unchanged")
	})

	t.Run("supports inline engine runtime.id for allowlist", func(t *testing.T) {
		content := `---
on: workflow_dispatch
engine:
  runtime:
    id: codex
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    OPENAI_BASE_URL: ${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1
---
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"engine": map[string]any{
				"runtime": map[string]any{
					"id": "codex",
				},
				"env": map[string]any{
					"OPENAI_API_KEY":  "${{ secrets.OPENAI_API_KEY }}",
					"OPENAI_BASE_URL": "${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}", "required engine secret should be preserved when using runtime.id")
		assert.NotContains(t, result, "OPENAI_BASE_URL: ${{ secrets.AZURE_OPENAI_ENDPOINT }}openai/v1", "unsafe secret-bearing key should still be removed")
	})
}
