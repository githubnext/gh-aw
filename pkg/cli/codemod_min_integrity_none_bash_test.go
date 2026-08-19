//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinIntegrityNoneRequiresBashCodemod(t *testing.T) {
	t.Parallel()
	codemod := getMinIntegrityNoneRequiresBashCodemod()

	assert.Equal(t, "min-integrity-none-requires-bash", codemod.ID)
	assert.NotEmpty(t, codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.NotEmpty(t, codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)

	t.Run("inserts bash: false when min-integrity is none and bash is absent", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools:
  github:
    min-integrity: none
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{"min-integrity": "none"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  bash: false")
		assert.Contains(t, result, "  bash: false\n  github:")
	})

	t.Run("does nothing when bash is already specified", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools:
  bash: ["cat", "ls"]
  github:
    min-integrity: none
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash":   []any{"cat", "ls"},
				"github": map[string]any{"min-integrity": "none"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does nothing when min-integrity is not none", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools:
  github:
    min-integrity: approved
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{"min-integrity": "approved"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does nothing when tools.github is absent", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
engine: copilot
---

# Test
`
		frontmatter := map[string]any{"engine": "copilot"}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does nothing when min-integrity is absent", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools:
  github:
    allowed-repos: all
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{"allowed-repos": "all"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})
}
