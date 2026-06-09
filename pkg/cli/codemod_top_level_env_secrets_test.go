//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopLevelEnvSecretsGuidedErrorCodemod(t *testing.T) {
	codemod := getTopLevelEnvSecretsGuidedErrorCodemod()

	t.Run("returns guided error when top-level env contains a secret", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
			},
		}

		_, applied, err := codemod.Apply(content, frontmatter)
		require.Error(t, err, "should return an error for top-level env secrets")
		assert.False(t, applied, "should not modify the file")
		assert.Contains(t, err.Error(), "top-level env: contains secrets")
		assert.Contains(t, err.Error(), "${{ secrets.GITHUB_TOKEN }}")
		assert.Contains(t, err.Error(), "Manual fix required")
		assert.Contains(t, err.Error(), "https://github.github.com/gh-aw/reference/engines/")
	})

	t.Run("returns deduplicated guided error with multiple secret references", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  PAT: ${{ secrets.GITHUB_PERSONAL_ACCESS_TOKEN || secrets.GITHUB_TOKEN }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"PAT": "${{ secrets.GITHUB_PERSONAL_ACCESS_TOKEN || secrets.GITHUB_TOKEN }}",
			},
		}

		_, applied, err := codemod.Apply(content, frontmatter)
		require.Error(t, err)
		assert.False(t, applied)
		assert.Contains(t, err.Error(), "top-level env: contains secrets")
		assert.Equal(t, 1, strings.Count(err.Error(), "${{ secrets.GITHUB_PERSONAL_ACCESS_TOKEN || secrets.GITHUB_TOKEN }}"))
	})

	t.Run("no-op when top-level env has no secrets", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  LOG_LEVEL: debug
  NODE_ENV: production
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"LOG_LEVEL": "debug",
				"NODE_ENV":  "production",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "should not error when env has no secrets")
		assert.False(t, applied, "should not apply when no secrets in env")
		assert.Equal(t, content, result)
	})

	t.Run("no-op when there is no top-level env section", func(t *testing.T) {
		content := `---
on: workflow_dispatch
engine:
  id: copilot
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"engine": map[string]any{
				"id": "copilot",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("no-op when env only uses vars not secrets", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  API_URL: ${{ vars.API_URL }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"API_URL": "${{ vars.API_URL }}",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "vars references are not secrets")
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not modify content even when secret found", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  TOKEN: ${{ secrets.MY_TOKEN }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"TOKEN": "${{ secrets.MY_TOKEN }}",
			},
		}

		result, _, _ := codemod.Apply(content, frontmatter)
		assert.Equal(t, content, result, "content must remain unchanged (guided error, not auto-fix)")
	})
}
