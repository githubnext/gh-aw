//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubReposToAllowedReposCodemod(t *testing.T) {
	codemod := getGitHubReposToAllowedReposCodemod()

	t.Run("renames repos to allowed-repos under tools.github", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
    repos: "all"
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode":          "remote",
					"toolsets":      []any{"default"},
					"repos":         "all",
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "allowed-repos: \"all\"", "Should rename repos to allowed-repos")
		assert.NotContains(t, result, "\n    repos: ", "Should not contain old repos: field")
	})

	t.Run("renames repos array to allowed-repos under tools.github", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    toolsets: [default]
    repos:
      - "myorg/*"
      - "partner/shared-repo"
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets":      []any{"default"},
					"repos":         []any{"myorg/*", "partner/shared-repo"},
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "allowed-repos:", "Should rename repos to allowed-repos")
		assert.NotContains(t, result, "    repos:\n", "Should not contain old repos: field")
	})

	t.Run("does not modify workflows without repos field", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    toolsets: [default]
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets": []any{"default"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not have applied the codemod")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("does not modify workflows without tools.github section", func(t *testing.T) {
		content := `---
engine: copilot
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not have applied the codemod")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("does not rename already-migrated allowed-repos field", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    toolsets: [default]
    allowed-repos: "all"
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets":      []any{"default"},
					"allowed-repos": "all",
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not have applied the codemod when already migrated")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("does not rename repos when allowed-repos already present", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    toolsets: [default]
    allowed-repos: "all"
    repos: "all"
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets":      []any{"default"},
					"allowed-repos": "all",
					"repos":         "all",
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply codemod when allowed-repos already present alongside repos")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("does not rename repos in toolsets list", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    toolsets: [repos, issues]
    repos: "all"
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets":      []any{"repos", "issues"},
					"repos":         "all",
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "allowed-repos: \"all\"", "Should rename repos to allowed-repos")
		// toolsets value 'repos' should remain unchanged
		assert.Contains(t, result, "toolsets: [repos, issues]", "Should not rename toolsets values")
	})

	t.Run("does not rename repos in comments", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    # repos: specifies the repository scope
    repos: "all"
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"repos":         "all",
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "allowed-repos: \"all\"", "Should rename repos key")
		// Comment should remain unchanged
		assert.Contains(t, result, "# repos: specifies the repository scope", "Should not rename repos in comments")
	})
}
