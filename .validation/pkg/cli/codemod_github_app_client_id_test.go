//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubAppClientIDCodemod(t *testing.T) {
	codemod := getGitHubAppClientIDCodemod()

	t.Run("renames app-id under github-app blocks", func(t *testing.T) {
		content := `---
github-app:
  app-id: ${{ vars.TOP_LEVEL_APP_ID }}
  private-key: ${{ secrets.TOP_LEVEL_APP_PRIVATE_KEY }}
on:
  github-app:
    app-id: ${{ vars.ACTIVATION_APP_ID }}
    private-key: ${{ secrets.ACTIVATION_APP_PRIVATE_KEY }}
checkout:
  - repository: org/repo
    github-app:
      app-id: ${{ vars.CHECKOUT_APP_ID }}
      private-key: ${{ secrets.CHECKOUT_APP_PRIVATE_KEY }}
---
`
		frontmatter := map[string]any{
			"github-app": map[string]any{
				"app-id":      "${{ vars.TOP_LEVEL_APP_ID }}",
				"private-key": "${{ secrets.TOP_LEVEL_APP_PRIVATE_KEY }}",
			},
			"on": map[string]any{
				"github-app": map[string]any{
					"app-id":      "${{ vars.ACTIVATION_APP_ID }}",
					"private-key": "${{ secrets.ACTIVATION_APP_PRIVATE_KEY }}",
				},
			},
			"checkout": []any{
				map[string]any{
					"repository": "org/repo",
					"github-app": map[string]any{
						"app-id":      "${{ vars.CHECKOUT_APP_ID }}",
						"private-key": "${{ secrets.CHECKOUT_APP_PRIVATE_KEY }}",
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error when applying codemod")
		assert.True(t, modified, "Should modify content")
		assert.NotContains(t, result, "app-id:", "Should remove deprecated app-id key from github-app blocks")
		assert.Contains(t, result, "client-id: ${{ vars.TOP_LEVEL_APP_ID }}", "Should migrate top-level github-app")
		assert.Contains(t, result, "client-id: ${{ vars.ACTIVATION_APP_ID }}", "Should migrate on.github-app")
		assert.Contains(t, result, "client-id: ${{ vars.CHECKOUT_APP_ID }}", "Should migrate checkout github-app")
	})

	t.Run("does not modify content without github-app.app-id", func(t *testing.T) {
		content := `---
github-app:
  client-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
---
`
		frontmatter := map[string]any{
			"github-app": map[string]any{
				"client-id":   "${{ vars.APP_ID }}",
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify already migrated content")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not rename app-id outside github-app blocks", func(t *testing.T) {
		content := `---
engine:
  provider:
    auth:
      client-id: APP_CLIENT_ID
tools:
  custom:
    app-id: value
---
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"provider": map[string]any{
					"auth": map[string]any{
						"client-id": "APP_CLIENT_ID",
					},
				},
			},
			"tools": map[string]any{
				"custom": map[string]any{
					"app-id": "value",
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify non github-app app-id keys")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})
}
