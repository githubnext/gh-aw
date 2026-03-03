//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerenaLocalModeCodemod(t *testing.T) {
	codemod := getSerenaLocalModeCodemod()

	t.Run("replaces mode: local with mode: docker", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  serena:
    mode: local
    languages:
      go: {}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"serena": map[string]any{
					"mode": "local",
					"languages": map[string]any{
						"go": map[string]any{},
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "mode: docker", "Should have mode: docker")
		assert.NotContains(t, result, "mode: local", "Should not contain mode: local")
	})

	t.Run("does not modify workflows without serena tool", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github: null
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": nil,
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not modify serena without mode field", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  serena:
    languages:
      go: {}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"serena": map[string]any{
					"languages": map[string]any{
						"go": map[string]any{},
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content when mode is not set")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not modify serena with mode: docker", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  serena:
    mode: docker
    languages:
      go: {}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"serena": map[string]any{
					"mode": "docker",
					"languages": map[string]any{
						"go": map[string]any{},
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content when mode is already docker")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("does not modify when tools section is absent", func(t *testing.T) {
		content := `---
engine: copilot
on: push
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"on":     "push",
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, modified, "Should not modify content without tools")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("preserves inline comments", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  serena:
    mode: local # deprecated
    languages:
      typescript: {}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"serena": map[string]any{
					"mode": "local",
					"languages": map[string]any{
						"typescript": map[string]any{},
					},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, modified, "Should modify content")
		assert.Contains(t, result, "mode: docker", "Should replace mode value")
		assert.NotContains(t, result, "mode: local", "Should not contain mode: local")
		assert.Contains(t, result, "# deprecated", "Should preserve inline comment")
	})

	t.Run("does not affect mode field outside tools.serena", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    mode: local
  serena:
    mode: local
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "local",
				},
				"serena": map[string]any{
					"mode": "local",
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, modified, "Should modify content")
		// GitHub tool mode should remain unchanged
		assert.Contains(t, result, "    mode: local", "GitHub tool mode should remain local")
		// Serena mode should be changed to docker
		assert.Contains(t, result, "mode: docker", "Serena mode should become docker")
	})
}
