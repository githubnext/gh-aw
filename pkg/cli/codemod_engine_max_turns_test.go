//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineMaxTurnsToTopLevelCodemod_Metadata(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

	assert.Equal(t, "engine-max-turns-to-top-level", codemod.ID)
	assert.Equal(t, "Move engine.max-turns to top-level max-turns", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.68.4", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestEngineMaxTurnsToTopLevelCodemod_NoOp(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

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

func TestEngineMaxTurnsToTopLevelCodemod_IdempotentWhenAlreadyMigrated(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

	content := `---
max-turns: "${{ inputs.max-turns }}"
engine:
  id: codex
---
`
	frontmatter := map[string]any{
		"max-turns": "${{ inputs.max-turns }}",
		"engine": map[string]any{
			"id": "codex",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEngineMaxTurnsToTopLevelCodemod_MigratesField(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: copilot
  max-turns: 42
---

# Body`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":        "copilot",
			"max-turns": 42,
		},
	}

	want := `---
on: push
max-turns: 42
engine:
  id: copilot
---

# Body`

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, want, result)
}

func TestEngineMaxTurnsToTopLevelCodemod_PreservesExpressionCommentsAndBody(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

	content := `---
on: workflow_dispatch
engine:
  id: copilot
  max-turns: "${{ inputs.max-turns }}" # runtime override
---

# Body
Keep this content.`
	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"engine": map[string]any{
			"id":        "copilot",
			"max-turns": "${{ inputs.max-turns }}",
		},
	}

	want := `---
on: workflow_dispatch
max-turns: "${{ inputs.max-turns }}" # runtime override
engine:
  id: copilot
---

# Body
Keep this content.`

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, want, result)
}

func TestEngineMaxTurnsToTopLevelCodemod_RespectsExistingTopLevel(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

	content := `---
max-turns: 10
engine:
  id: copilot
  max-turns: 42
---
`
	frontmatter := map[string]any{
		"max-turns": 10,
		"engine": map[string]any{
			"id":        "copilot",
			"max-turns": 42,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-turns: 10")
	assert.NotContains(t, result, "max-turns: 42")
	assert.NotContains(t, result, "\n  max-turns:")
}

func TestEngineMaxTurnsToTopLevelCodemod_InlineEngineMapNoOp(t *testing.T) {
	codemod := getEngineMaxTurnsToTopLevelCodemod()

	content := `---
on: push
engine: { id: copilot, max-turns: "${{ inputs.max-turns }}" }
---
`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":        "copilot",
			"max-turns": "${{ inputs.max-turns }}",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
