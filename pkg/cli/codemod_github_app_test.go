//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hasDeprecatedAppFieldInContent returns true if any line in the content has 'app:' as its YAML key
// (i.e., trimmed content starts with "app:" – matches the field name, not app-id: or github-app:)
func hasDeprecatedAppFieldInContent(content string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "app:" || strings.HasPrefix(trimmed, "app: ") || strings.HasPrefix(trimmed, "app:\t") {
			return true
		}
	}
	return false
}

func TestGitHubAppCodemod(t *testing.T) {
	codemod := getGitHubAppCodemod()

	t.Run("renames app to github-app under tools.github", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "remote",
					"app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error when applying codemod")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "github-app:", "Should contain github-app field")
		assert.False(t, hasDeprecatedAppFieldInContent(result), "Should not contain old app field")
	})

	t.Run("renames app to github-app under safe-outputs", func(t *testing.T) {
		content := `---
engine: copilot
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-issue:
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"safe-outputs": map[string]any{
				"app": map[string]any{
					"app-id":      "${{ vars.APP_ID }}",
					"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
				},
				"create-issue": nil,
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error when applying codemod")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "github-app:", "Should contain github-app field")
		assert.False(t, hasDeprecatedAppFieldInContent(result), "Should not contain old app field")
	})

	t.Run("renames app to github-app under checkout", func(t *testing.T) {
		content := `---
engine: copilot
checkout:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"checkout": map[string]any{
				"app": map[string]any{
					"app-id":      "${{ vars.APP_ID }}",
					"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error when applying codemod")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "github-app:", "Should contain github-app field")
		assert.False(t, hasDeprecatedAppFieldInContent(result), "Should not contain old app field")
	})

	t.Run("does not modify workflows without app field", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    github-token: ${{ secrets.MY_TOKEN }}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode":         "remote",
					"github-token": "${{ secrets.MY_TOKEN }}",
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content without app field")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not modify app field outside target sections", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "remote",
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content when no app field in target sections")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("renames app in all three sections", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-issue:
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "remote",
					"app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
			"safe-outputs": map[string]any{
				"app": map[string]any{
					"app-id":      "${{ vars.APP_ID }}",
					"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
				},
				"create-issue": nil,
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error when applying codemod")
		assert.True(t, modified, "Should modify content")
		// Both app fields should be renamed
		assert.False(t, hasDeprecatedAppFieldInContent(result), "Should not contain any old app fields")
	})

	t.Run("does not rename already migrated github-app field", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    github-app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "remote",
					"github-app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content with github-app field (already migrated)")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("renames app to github-app inside checkout array item", func(t *testing.T) {
		content := `---
engine: copilot
checkout:
  - repo: org/other-repo
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"checkout": []any{
				map[string]any{
					"repo": "org/other-repo",
					"app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error when applying codemod")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "github-app:", "Should contain github-app field")
		assert.False(t, hasDeprecatedAppFieldInContent(result), "Should not contain old app field")
	})

	t.Run("preserves comments and formatting", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    # GitHub App for token minting
    app:  # Use a GitHub App
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "remote",
					"app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "# GitHub App for token minting", "Should preserve comment")
		assert.Contains(t, result, "github-app:  # Use a GitHub App", "Should preserve inline comment")
	})
}
