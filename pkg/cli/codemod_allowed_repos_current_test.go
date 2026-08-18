//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedReposCurrentToGitHubRepositoryCodemod(t *testing.T) {
	t.Parallel()
	codemod := getAllowedReposCurrentToGitHubRepositoryCodemod()

	t.Run("metadata is populated", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "allowed-repos-current-to-github-repository", codemod.ID)
		assert.NotEmpty(t, codemod.Name)
		assert.NotEmpty(t, codemod.Description)
		assert.Equal(t, "0.85.5", codemod.IntroducedIn)
		require.NotNil(t, codemod.Apply)
	})

	t.Run("rewrites unquoted current value", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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

	t.Run("rewrites single-quoted current value", func(t *testing.T) {
		t.Parallel()
		content := `---
engine: copilot
tools:
  github:
    allowed-repos: 'current'
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
		assert.Contains(t, result, `allowed-repos: "${{ github.repository }}"`, "Should rewrite single-quoted current to the github.repository expression")
	})

	t.Run("no-op when allowed-repos is already an expression", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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

	t.Run("does not rewrite nested non-top-level tools github allowed-repos", func(t *testing.T) {
		t.Parallel()
		content := `---
engine: copilot
wrapper:
  tools:
    github:
      allowed-repos: current
tools:
  github:
    allowed-repos: current
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"wrapper": map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed-repos": "current",
					},
				},
			},
			"tools": map[string]any{
				"github": map[string]any{
					"allowed-repos": "current",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should rewrite direct top-level tools.github allowed-repos")
		assert.Contains(t, result, "      allowed-repos: current", "Should preserve nested non-top-level tools.github value")
		assert.Contains(t, result, `    allowed-repos: "${{ github.repository }}"`, "Should rewrite top-level tools.github value")
	})

	t.Run("does not rewrite nested custom github allowed-repos", func(t *testing.T) {
		t.Parallel()
		content := `---
engine: copilot
tools:
  custom:
    github:
      allowed-repos: current
  github:
    allowed-repos: current
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"custom": map[string]any{
					"github": map[string]any{
						"allowed-repos": "current",
					},
				},
				"github": map[string]any{
					"allowed-repos": "current",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should rewrite direct tools.github allowed-repos")
		assert.Contains(t, result, "      allowed-repos: current", "Should preserve nested custom github allowed-repos")
		assert.Contains(t, result, `    allowed-repos: "${{ github.repository }}"`, "Should rewrite direct tools.github value")
	})
}
