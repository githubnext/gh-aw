//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineModelToTopLevelCodemod_Metadata(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

	assert.Equal(t, "engine-model-to-top-level", codemod.ID)
	assert.Equal(t, "Move engine.model to top-level model", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.78.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestEngineModelToTopLevelCodemod_NoOp(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

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

func TestEngineModelToTopLevelCodemod_IdempotentWhenAlreadyMigrated(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

	content := `---
model: gpt-5.4
engine:
  id: copilot
---
`
	frontmatter := map[string]any{
		"model": "gpt-5.4",
		"engine": map[string]any{
			"id": "copilot",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEngineModelToTopLevelCodemod_MigratesField(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

	content := `---
on: push
engine:
  id: copilot
  model: gpt-5.4
---

# Body`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":    "copilot",
			"model": "gpt-5.4",
		},
	}

	want := `---
on: push
model: gpt-5.4
engine:
  id: copilot
---

# Body`

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, want, result)
}

func TestEngineModelToTopLevelCodemod_PreservesCommentsAndBody(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

	content := `---
on: workflow_dispatch
engine:
  id: claude
  model: claude-3-5-sonnet-20241022 # pinned version
---

# Prompt body
Keep this content.`
	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"engine": map[string]any{
			"id":    "claude",
			"model": "claude-3-5-sonnet-20241022",
		},
	}

	want := `---
on: workflow_dispatch
model: claude-3-5-sonnet-20241022 # pinned version
engine:
  id: claude
---

# Prompt body
Keep this content.`

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, want, result)
}

func TestEngineModelToTopLevelCodemod_RespectsExistingTopLevel(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

	content := `---
model: gpt-5.4
engine:
  id: copilot
  model: gpt-4
---
`
	frontmatter := map[string]any{
		"model": "gpt-5.4",
		"engine": map[string]any{
			"id":    "copilot",
			"model": "gpt-4",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "model: gpt-5.4")
	assert.NotContains(t, result, "model: gpt-4")
	assert.NotContains(t, result, "\n  model:")
}

func TestEngineModelToTopLevelCodemod_InlineEngineMapNoOp(t *testing.T) {
	codemod := getEngineModelToTopLevelCodemod()

	content := `---
on: push
engine: { id: copilot, model: gpt-5.4 }
---
`
	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":    "copilot",
			"model": "gpt-5.4",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
