//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestTargetCheckoutFalseCodemod(t *testing.T) {
	codemod := getPullRequestTargetCheckoutFalseCodemod()

	t.Run("adds checkout false after on block when missing", func(t *testing.T) {
		content := `---
on:
  pull_request_target:
description: Review PR metadata
---

# Prompt
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.True(t, applied, "codemod should apply when checkout is missing")
		assert.Contains(t, result, "checkout: false", "codemod should add checkout: false")
		assert.Contains(t, result, "pull_request_target:\ncheckout: false\ndescription:", "checkout should be inserted after on block")
	})

	t.Run("normalizes checkout true to false", func(t *testing.T) {
		content := `---
on:
  pull_request_target:
checkout: true
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
			"checkout": true,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.True(t, applied, "codemod should apply when checkout is true")
		assert.Contains(t, result, "checkout: false", "codemod should set checkout to false")
		assert.NotContains(t, result, "checkout: true", "codemod should remove checkout: true")
	})

	t.Run("preserves inline comment spacing when normalizing checkout", func(t *testing.T) {
		content := `---
on:
  pull_request_target:
checkout: true # keep-comment
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
			"checkout": true,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.True(t, applied, "codemod should apply when checkout is true")
		assert.Contains(t, result, "checkout: false # keep-comment", "codemod should keep space before inline comment")
		assert.NotContains(t, result, "checkout: false# keep-comment", "codemod should not collapse comment spacing")
	})

	t.Run("does not modify when checkout false already exists", func(t *testing.T) {
		content := `---
on:
  pull_request_target:
checkout: false
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
			"checkout": false,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.False(t, applied, "codemod should not apply when checkout is already false")
		assert.Equal(t, content, result, "content should remain unchanged")
	})

	t.Run("does not modify non pull_request_target workflow", func(t *testing.T) {
		content := `---
on:
  pull_request:
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.False(t, applied, "codemod should not apply without pull_request_target")
		assert.Equal(t, content, result, "content should remain unchanged")
	})

	t.Run("does not modify when explicit checkout command exists", func(t *testing.T) {
		content := `---
on:
  pull_request_target:
---

Run gh pr checkout ${{ github.event.pull_request.number }} before tests.
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.False(t, applied, "codemod should not apply when explicit checkout command exists")
		assert.Equal(t, content, result, "content should remain unchanged")
	})

	t.Run("does not modify when git checkout uses tab separator", func(t *testing.T) {
		content := "---\non:\n  pull_request_target:\n---\n\nUse git checkout\tfeature-branch before tests.\n"
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.False(t, applied, "codemod should not apply when git checkout is present with tab separator")
		assert.Equal(t, content, result, "content should remain unchanged")
	})

	t.Run("does not modify when strict is explicitly false", func(t *testing.T) {
		content := `---
on:
  pull_request_target:
strict: false
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_target": map[string]any{},
			},
			"strict": false,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not return an error")
		assert.False(t, applied, "codemod should not apply when strict is false")
		assert.Equal(t, content, result, "content should remain unchanged")
	})
}
