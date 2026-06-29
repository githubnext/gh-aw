//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputDispatchRepositoryKeyCodemod(t *testing.T) {
	codemod := getSafeOutputDispatchRepositoryKeyCodemod()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "safe-output-dispatch-repository-key", codemod.ID)
		assert.Equal(t, "Rename safe-outputs.dispatch_repository to dispatch-repository", codemod.Name)
		assert.Equal(t, "Renames deprecated safe-outputs.dispatch_repository to safe-outputs.dispatch-repository.", codemod.Description)
		assert.Equal(t, "1.0.65", codemod.IntroducedIn)
		require.NotNil(t, codemod.Apply)
	})

	t.Run("renames safe-outputs dispatch_repository key", func(t *testing.T) {
		content := `---
on: workflow_dispatch
safe-outputs:
  dispatch_repository:
    relay:
      workflow: router.yml
      event_type: dispatch
      repository: github/gh-aw
---

Body text.
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"safe-outputs": map[string]any{
				"dispatch_repository": map[string]any{
					"relay": map[string]any{
						"workflow":   "router.yml",
						"event_type": "dispatch",
						"repository": "github/gh-aw",
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  dispatch-repository:")
		assert.NotContains(t, result, "  dispatch_repository:")
		assert.Contains(t, result, "\n\nBody text.")
	})

	t.Run("preserves comments and indentation", func(t *testing.T) {
		content := `---
safe-outputs:
  # relay config
  dispatch_repository: # inline comment
    relay:
      workflow: router.yml
      event_type: dispatch
      repository: github/gh-aw
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"dispatch_repository": map[string]any{
					"relay": map[string]any{
						"workflow":   "router.yml",
						"event_type": "dispatch",
						"repository": "github/gh-aw",
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  dispatch-repository: # inline comment")
		assert.Contains(t, result, "  # relay config")
	})

	t.Run("no-op when deprecated key absent", func(t *testing.T) {
		content := `---
safe-outputs:
  dispatch-repository:
    relay:
      workflow: router.yml
      event_type: dispatch
      repository: github/gh-aw
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"dispatch-repository": map[string]any{
					"relay": map[string]any{
						"workflow":   "router.yml",
						"event_type": "dispatch",
						"repository": "github/gh-aw",
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("no-op when both keys already exist", func(t *testing.T) {
		content := `---
safe-outputs:
  dispatch-repository:
    canonical:
      workflow: router.yml
      event_type: dispatch
      repository: github/gh-aw
  dispatch_repository:
    alias:
      workflow: router.yml
      event_type: dispatch
      repository: github/gh-aw
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"dispatch-repository": map[string]any{},
				"dispatch_repository": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("no-op when dispatch_repository appears only in a description value", func(t *testing.T) {
		content := `---
safe-outputs:
  some-tool:
    description: "Triggers via dispatch_repository: mechanism"
    workflow: router.yml
    event_type: dispatch
    repository: github/gh-aw
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"some-tool": map[string]any{
					"description": "Triggers via dispatch_repository: mechanism",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})
}
