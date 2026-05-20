//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputAddReviewerAllowlistsCodemod(t *testing.T) {
	codemod := getSafeOutputAddReviewerAllowlistsCodemod()

	t.Run("renames reviewers and team-reviewers to allowed-reviewers and allowed-team-reviewers", func(t *testing.T) {
		content := `---
safe-outputs:
  add-reviewer:
    reviewers: [user1, copilot]
    team-reviewers: [platform-team]
    max: 3
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"add-reviewer": map[string]any{
					"reviewers":      []string{"user1", "copilot"},
					"team-reviewers": []string{"platform-team"},
					"max":            3,
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "allowed-reviewers:")
		assert.Contains(t, result, "allowed-team-reviewers:")
		assert.NotContains(t, result, "\n    reviewers:")
		assert.NotContains(t, result, "\n    team-reviewers:")
	})

	t.Run("renames only reviewers when team-reviewers absent", func(t *testing.T) {
		content := `---
safe-outputs:
  add-reviewer:
    reviewers: [user1]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"add-reviewer": map[string]any{
					"reviewers": []string{"user1"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "allowed-reviewers:")
		assert.NotContains(t, result, "\n    reviewers:")
	})

	t.Run("does not modify when new fields already present", func(t *testing.T) {
		content := `---
safe-outputs:
  add-reviewer:
    allowed-reviewers: [user1]
    allowed-team-reviewers: [platform-team]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"add-reviewer": map[string]any{
					"allowed-reviewers":      []string{"user1"},
					"allowed-team-reviewers": []string{"platform-team"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not affect create-pull-request reviewers", func(t *testing.T) {
		content := `---
safe-outputs:
  create-pull-request:
    reviewers: [user1]
  add-reviewer:
    reviewers: [user2]
---
`
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"create-pull-request": map[string]any{
					"reviewers": []string{"user1"},
				},
				"add-reviewer": map[string]any{
					"reviewers": []string{"user2"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		// create-pull-request.reviewers must remain unchanged
		assert.Contains(t, result, "  create-pull-request:\n    reviewers:")
		// only add-reviewer gets renamed
		assert.Contains(t, result, "allowed-reviewers:")
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
