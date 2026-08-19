//go:build !integration

package cli

import (
	"os"
	"path/filepath"
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

func TestMinIntegrityNoneRequiresBashCodemod_InlineToolsMapping(t *testing.T) {
	t.Parallel()
	codemod := getMinIntegrityNoneRequiresBashCodemod()

	t.Run("inserts bash: false into an inline tools mapping", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools: {github: {min-integrity: none}}
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
		assert.Contains(t, result, "tools: {bash: false, github: {min-integrity: none}}")
	})

	t.Run("preserves spacing of an inline tools mapping", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools: { github: { min-integrity: none } }
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
		assert.Contains(t, result, "tools: {bash: false, github: { min-integrity: none } }")
	})

	t.Run("skips multi-line inline tools mappings", func(t *testing.T) {
		t.Parallel()
		content := `---
on: workflow_dispatch
tools: {
  github: {min-integrity: none}
}
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
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})
}

// TestMinIntegrityNoneRequiresBash_SingleFixPassAlsoDisablesCLIProxy verifies that the
// registry order lets one fix pass emit both 'bash: false' and the 'cli-proxy: false'
// that strict mode requires once bash is disabled.
func TestMinIntegrityNoneRequiresBash_SingleFixPassAlsoDisablesCLIProxy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{
			name: "block mapping",
			content: `---
on: workflow_dispatch
tools:
  github:
    min-integrity: none
---

# Test
`,
		},
		{
			name: "inline mapping",
			content: `---
on: workflow_dispatch
tools: {github: {min-integrity: none}}
---

# Test
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowFile := filepath.Join(tmpDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(workflowFile, []byte(tc.content), 0644))

			fixed, _, err := processWorkflowFileWithInfo(workflowFile, GetAllCodemods(), true, false)
			require.NoError(t, err)
			require.True(t, fixed)

			result, err := os.ReadFile(workflowFile)
			require.NoError(t, err)
			assert.Contains(t, string(result), "bash: false")
			assert.Contains(t, string(result), "cli-proxy: false")
		})
	}
}
