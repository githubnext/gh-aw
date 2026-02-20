//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEngineStepsToTopLevelCodemod_Metadata(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	assert.Equal(t, "engine-steps-to-top-level", codemod.ID)
	assert.Equal(t, "Move engine.steps to top-level steps", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.11.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestEngineStepsToTopLevelCodemod_NoEngine(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	content := `---
on: push
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "push",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Should not apply when no engine field")
	assert.Equal(t, content, result)
}

func TestEngineStepsToTopLevelCodemod_StringEngine(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	content := `---
on: push
engine: claude
---

# Test workflow`

	frontmatter := map[string]any{
		"on":     "push",
		"engine": "claude",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Should not apply when engine is a string (no steps)")
	assert.Equal(t, content, result)
}

func TestEngineStepsToTopLevelCodemod_EngineWithoutSteps(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: claude
  model: claude-3-5-sonnet
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":    "claude",
			"model": "claude-3-5-sonnet",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Should not apply when engine has no steps")
	assert.Equal(t, content, result)
}

func TestEngineStepsToTopLevelCodemod_SimpleSteps(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: codex
  steps:
    - name: Run step
      run: echo "hello"
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id": "codex",
			"steps": []any{
				map[string]any{"name": "Run step", "run": `echo "hello"`},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Should apply when engine has steps")

	// Should have top-level steps
	assert.Contains(t, result, "steps:")
	assert.Contains(t, result, "name: Run step")

	// Should not have engine.steps anymore
	assert.NotContains(t, result, "  steps:")

	// Engine block should still exist
	assert.Contains(t, result, "engine:")
	assert.Contains(t, result, "id: codex")
}

func TestEngineStepsToTopLevelCodemod_StepsRemovedFromEngine(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: codex
  model: gpt-4o
  steps:
    - name: Step 1
      run: echo "step1"
    - name: Step 2
      run: echo "step2"
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":    "codex",
			"model": "gpt-4o",
			"steps": []any{
				map[string]any{"name": "Step 1", "run": `echo "step1"`},
				map[string]any{"name": "Step 2", "run": `echo "step2"`},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)

	// Steps should be at top level
	assert.Contains(t, result, "steps:\n")
	assert.Contains(t, result, "name: Step 1")
	assert.Contains(t, result, "name: Step 2")

	// Engine should still have id and model
	assert.Contains(t, result, "id: codex")
	assert.Contains(t, result, "model: gpt-4o")

	// Engine should no longer have steps
	lines := strings.Split(result, "\n")
	inEngine := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "engine:") {
			inEngine = true
		} else if inEngine && len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inEngine = false
		}
		if inEngine && trimmed == "steps:" {
			t.Error("engine block should not contain 'steps:' after codemod")
		}
	}
}

func TestEngineStepsToTopLevelCodemod_MergeWithExistingSteps(t *testing.T) {
	codemod := getEngineStepsToTopLevelCodemod()

	content := `---
on: push
engine:
  id: codex
  steps:
    - name: Engine Step
      run: echo "engine"
steps:
  - name: Existing Step
    run: echo "existing"
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id": "codex",
			"steps": []any{
				map[string]any{"name": "Engine Step", "run": `echo "engine"`},
			},
		},
		"steps": []any{
			map[string]any{"name": "Existing Step", "run": `echo "existing"`},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)

	// Both steps should be present
	assert.Contains(t, result, "name: Engine Step")
	assert.Contains(t, result, "name: Existing Step")

	// Should have only one top-level "steps:" header
	stepsCount := strings.Count(result, "\nsteps:\n")
	assert.Equal(t, 1, stepsCount, "Should have exactly one top-level 'steps:' header")

	// Engine block should not have steps
	assert.NotContains(t, result, "  steps:")
}
