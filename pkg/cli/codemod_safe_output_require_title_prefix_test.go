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
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "require-title-prefix: \"[bot] \"")
		assert.NotContains(t, result, "\n    title-prefix:")
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
}
