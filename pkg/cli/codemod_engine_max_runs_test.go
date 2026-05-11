//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineMaxRunsToTopLevelCodemod_Metadata(t *testing.T) {
	codemod := getEngineMaxRunsToTopLevelCodemod()

	assert.Equal(t, "engine-max-runs-to-top-level", codemod.ID)
	assert.Equal(t, "Move engine.max-runs to top-level max-runs", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.17.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestEngineMaxRunsToTopLevelCodemod_NoOp(t *testing.T) {
	codemod := getEngineMaxRunsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: copilot
---
`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id": "copilot",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEngineMaxRunsToTopLevelCodemod_MigratesField(t *testing.T) {
	codemod := getEngineMaxRunsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: copilot
  max-runs: 42
---

# Body`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":       "copilot",
			"max-runs": 42,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "\nmax-runs: 42\nengine:")
	assert.NotContains(t, result, "\n  max-runs:")
}

func TestEngineMaxRunsToTopLevelCodemod_RespectsExistingTopLevel(t *testing.T) {
	codemod := getEngineMaxRunsToTopLevelCodemod()

	content := `---
max-runs: 10
engine:
  id: copilot
  max-runs: 42
---
`
	frontmatter := map[string]any{
		"max-runs": 10,
		"engine": map[string]any{
			"id":       "copilot",
			"max-runs": 42,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-runs: 10")
	assert.NotContains(t, result, "max-runs: 42")
	assert.NotContains(t, result, "\n  max-runs:")
}

func TestEngineMaxRunsToTopLevelCodemod_InlineEngineMapNoOp(t *testing.T) {
	codemod := getEngineMaxRunsToTopLevelCodemod()

	content := `---
on: push
engine: { id: copilot, max-runs: 42 }
---
`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":       "copilot",
			"max-runs": 42,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
