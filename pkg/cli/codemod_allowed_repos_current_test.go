//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedReposCurrentToGitHubRepositoryCodemod(t *testing.T) {
	codemod := getAllowedReposCurrentToGitHubRepositoryCodemod()

	t.Run("metadata is populated", func(t *testing.T) {
		assert.Equal(t, "allowed-repos-current-to-github-repository", codemod.ID)
		assert.NotEmpty(t, codemod.Name)
		assert.NotEmpty(t, codemod.Description)
		assert.NotEmpty(t, codemod.IntroducedIn)
		require.NotNil(t, codemod.Apply)
	})

	t.Run("rewrites unquoted current value", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    toolsets: [default]
    allowed-repos: current
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets":      []any{"default"},
					"allowed-repos": "current",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, `allowed-repos: "${{ github.repository }}"`, "Should rewrite current to the github.repository expression")
		assert.NotContains(t, result, "allowed-repos: current", "Should not contain the old current value")
	})

	t.Run("rewrites quoted current value", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    allowed-repos: "current"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": "current",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, `allowed-repos: "${{ github.repository }}"`, "Should rewrite current to the github.repository expression")
	})

	t.Run("no-op when allowed-repos is already an expression", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    allowed-repos: "${{ github.repository }}"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": "${{ github.repository }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when already migrated")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("no-op when allowed-repos is an array", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    allowed-repos:
      - "myorg/*"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": []any{"myorg/*"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when allowed-repos is an array")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("no-op when allowed-repos is set to all", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    allowed-repos: "all"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": "all",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when allowed-repos is 'all'")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("preserves trailing comments", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    allowed-repos: current # legacy alias
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": "current",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, `allowed-repos: "${{ github.repository }}" # legacy alias`, "Should preserve trailing comment")
	})

	t.Run("only treats whitespace-preceded hash as a comment marker", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    allowed-repos: current # see docs on "#current" alias
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": "current",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, `allowed-repos: "${{ github.repository }}" # see docs on "#current" alias`, "Should preserve the full comment including embedded hash")
	})
}
