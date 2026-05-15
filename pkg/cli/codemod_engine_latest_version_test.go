//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPinEngineLatestVersionCodemod(t *testing.T) {
	codemod := getPinEngineLatestVersionCodemod()

	t.Run("removes engine version latest", func(t *testing.T) {
		content := `---
engine:
  id: copilot
  version: latest
on: push
---
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":      "copilot",
				"version": "latest",
			},
			"on": "push",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "id: copilot", "engine id should be preserved")
		assert.NotContains(t, result, "version: latest", "latest version should be removed")
	})

	t.Run("no-op when version already pinned", func(t *testing.T) {
		content := `---
engine:
  id: copilot
  version: v1.2.3
---
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":      "copilot",
				"version": "v1.2.3",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not error in no-op case")
		assert.False(t, applied, "codemod should not apply")
		assert.Equal(t, content, result, "content should remain unchanged")
	})
}
