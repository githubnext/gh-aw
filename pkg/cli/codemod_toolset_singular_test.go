//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsetSingularToToolsetsCodemod(t *testing.T) {
	t.Parallel()
	codemod := getToolsetSingularToToolsetsCodemod()

	t.Run("metadata is populated", func(t *testing.T) {
		assert.Equal(t, "toolset-singular-to-toolsets", codemod.ID)
		assert.NotEmpty(t, codemod.Name)
		assert.NotEmpty(t, codemod.Description)
		assert.Equal(t, "0.85.5", codemod.IntroducedIn)
		require.NotNil(t, codemod.Apply)
	})

	t.Run("renames toolset to toolsets under tools.github", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: remote
    toolset: [default]
    allowed-repos: "all"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode":          "remote",
					"toolset":       []any{"default"},
					"allowed-repos": "all",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "toolsets: [default]", "Should rename toolset to toolsets")
		assert.NotContains(t, result, "\n    toolset: ", "Should not contain old toolset: field")
	})

	t.Run("no-op when toolsets already present", func(t *testing.T) {
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
		assert.False(t, applied, "Should not apply when already migrated")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("no-op when tools.github is absent", func(t *testing.T) {
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
		assert.False(t, applied, "Should not apply when tools.github is absent")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not rename toolset in comments", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    # toolset: legacy comment
    toolset: default
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"toolset": "default",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "toolsets: default", "Should rename toolset key")
		assert.Contains(t, result, "# toolset: legacy comment", "Should not rename toolset in comments")
	})

	t.Run("no-op when toolset appears outside tools.github", func(t *testing.T) {
		content := `---
engine: copilot
other:
  toolset: some-value
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"other": map[string]any{
				"toolset": "some-value",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not rename toolset outside tools.github")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not rename nested custom github toolset", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  custom:
    github:
      toolset: custom-value
  github:
    toolset: default
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"custom": map[string]any{
					"github": map[string]any{
						"toolset": "custom-value",
					},
				},
				"github": map[string]any{
					"toolset": "default",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should rename tools.github.toolset")
		assert.Contains(t, result, "      toolset: custom-value", "Should preserve nested custom github toolset")
		assert.Contains(t, result, "    toolsets: default", "Should rename direct tools.github toolset")
	})
}
