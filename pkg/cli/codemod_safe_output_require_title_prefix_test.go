//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputRequireTitlePrefixCodemod(t *testing.T) {
	codemod := getSafeOutputRequireTitlePrefixCodemod()

	t.Run("renames close and push constraint keys", func(t *testing.T) {
		content := `---
safe-outputs:
  close-issue:
    title-prefix: "[bot] "
  push-to-pull-request-branch:
    target: "*"
    title-prefix: "[bot] "
    labels: [automated]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"close-issue": map[string]any{
					"title-prefix": "[bot] ",
				},
				"push-to-pull-request-branch": map[string]any{
					"target":       "*",
					"title-prefix": "[bot] ",
					"labels":       []string{"automated"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required-title-prefix: \"[bot] \"")
		assert.Contains(t, result, "required-labels:")
		assert.NotContains(t, result, "\n    title-prefix:")
		assert.NotContains(t, result, "\n    labels:")
	})

	t.Run("does not rename create-issue title-prefix", func(t *testing.T) {
		content := `---
safe-outputs:
  create-issue:
    title-prefix: "[create] "
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{
					"title-prefix": "[create] ",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not modify when required field already present", func(t *testing.T) {
		content := `---
safe-outputs:
  close-issue:
    required-title-prefix: "[bot] "
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"close-issue": map[string]any{
					"required-title-prefix": "[bot] ",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("renames push labels when required-title-prefix already present", func(t *testing.T) {
		content := `---
safe-outputs:
  push-to-pull-request-branch:
    target: "*"
    required-title-prefix: "[bot] "
    labels: [automated]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"target":                "*",
					"required-title-prefix": "[bot] ",
					"labels":                []string{"automated"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required-labels:")
		assert.NotContains(t, result, "\n    labels:")
		assert.Contains(t, result, "required-title-prefix:")
		assert.NotContains(t, result, "\n    title-prefix:")
	})

	t.Run("renames nested push keys without losing active handler", func(t *testing.T) {
		content := `---
safe-outputs:
  push-to-pull-request-branch:
    target: "*"
    protected-files:
      - "README.md"
    title-prefix: "[bot] "
    labels: [automated]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"target":          "*",
					"title-prefix":    "[bot] ",
					"labels":          []string{"automated"},
					"protected-files": []string{"README.md"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required-title-prefix:")
		assert.Contains(t, result, "required-labels:")
		assert.NotContains(t, result, "\n    title-prefix:")
		assert.NotContains(t, result, "\n    labels:")
	})

	t.Run("only renames direct handler keys", func(t *testing.T) {
		content := `---
safe-outputs:
  push-to-pull-request-branch:
    metadata:
      labels: [nested]
      title-prefix: "[nested] "
    title-prefix: "[bot] "
    labels: [automated]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"title-prefix": "[bot] ",
					"labels":       []string{"automated"},
					"metadata": map[string]any{
						"labels":       []string{"nested"},
						"title-prefix": "[nested] ",
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "\n    required-title-prefix:")
		assert.Contains(t, result, "\n    required-labels:")
		assert.Contains(t, result, "\n      labels: [nested]")
		assert.Contains(t, result, "\n      title-prefix: \"[nested] \"")
	})
}
