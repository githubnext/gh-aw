//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputMergePRConstraintsCodemod(t *testing.T) {
	codemod := getSafeOutputMergePRConstraintsCodemod()

	t.Run("renames allowed-labels to required-labels, leaves allowed-branches unchanged", func(t *testing.T) {
		content := `---
safe-outputs:
  merge-pull-request:
    allowed-labels: [release, automerge]
    allowed-branches: [release/*, main]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"merge-pull-request": map[string]any{
					"allowed-labels":   []string{"release", "automerge"},
					"allowed-branches": []string{"release/*", "main"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required-labels:")
		assert.NotContains(t, result, "allowed-labels:")
		// allowed-branches is NOT renamed by the codemod
		assert.Contains(t, result, "allowed-branches:")
	})

	t.Run("no-op when only allowed-branches present (no allowed-labels to migrate)", func(t *testing.T) {
		content := `---
safe-outputs:
  merge-pull-request:
    allowed-branches: [main]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"merge-pull-request": map[string]any{
					"allowed-branches": []string{"main"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied, "no migration needed when only allowed-branches is present")
		assert.Equal(t, content, result)
	})

	t.Run("does not modify when new fields already present", func(t *testing.T) {
		content := `---
safe-outputs:
  merge-pull-request:
    required-labels: [release]
    allowed-branches: [main]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"merge-pull-request": map[string]any{
					"required-labels":  []string{"release"},
					"allowed-branches": []string{"main"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not affect other safe-outputs handlers", func(t *testing.T) {
		content := `---
safe-outputs:
  close-issue:
    allowed-labels: [bot]
  merge-pull-request:
    allowed-labels: [automerge]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"close-issue": map[string]any{
					"allowed-labels": []string{"bot"},
				},
				"merge-pull-request": map[string]any{
					"allowed-labels": []string{"automerge"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		// close-issue.allowed-labels must remain unchanged
		assert.Contains(t, result, "  close-issue:\n    allowed-labels:")
		// only merge-pull-request gets renamed
		assert.Contains(t, result, "required-labels:")
	})

	t.Run("no-op when safe-outputs missing", func(t *testing.T) {
		content := `---
engine: copilot
---
`
		frontmatter := map[string]any{"engine": "copilot"}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})
}
