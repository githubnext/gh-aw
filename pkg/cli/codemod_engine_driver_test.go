//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineCopilotSDKDriverToDriverCodemod(t *testing.T) {
	codemod := getEngineCopilotSDKDriverToDriverCodemod()

	t.Run("renames copilot-sdk-driver to driver in engine block", func(t *testing.T) {
		content := `---
engine:
  id: copilot
  copilot-sdk-driver: .github/drivers/my_driver.cjs
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":                 "copilot",
				"copilot-sdk-driver": ".github/drivers/my_driver.cjs",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  driver: .github/drivers/my_driver.cjs")
		assert.NotContains(t, result, "copilot-sdk-driver")
	})

	t.Run("does not apply when engine has no copilot-sdk-driver field", func(t *testing.T) {
		content := `---
engine:
  id: copilot
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id": "copilot",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not apply when driver field already present", func(t *testing.T) {
		content := `---
engine:
  id: copilot
  driver: .github/drivers/my_driver.cjs
  copilot-sdk-driver: .github/drivers/my_driver.cjs
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":                 "copilot",
				"driver":             ".github/drivers/my_driver.cjs",
				"copilot-sdk-driver": ".github/drivers/my_driver.cjs",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not apply when no engine block", func(t *testing.T) {
		content := `---
name: My Workflow
---

# Test Workflow
`
		frontmatter := map[string]any{
			"name": "My Workflow",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not apply when engine is a scalar", func(t *testing.T) {
		content := `---
engine: copilot
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("preserves indentation when renaming", func(t *testing.T) {
		content := `---
engine:
  id: copilot
  copilot-sdk: true
  copilot-sdk-driver: .github/drivers/copilot_sdk_driver_sample_typescript.ts
strict: true
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":                 "copilot",
				"copilot-sdk":        true,
				"copilot-sdk-driver": ".github/drivers/copilot_sdk_driver_sample_typescript.ts",
			},
			"strict": true,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  driver: .github/drivers/copilot_sdk_driver_sample_typescript.ts")
		assert.NotContains(t, result, "copilot-sdk-driver")
		assert.Contains(t, result, "  copilot-sdk: true")
	})
}
