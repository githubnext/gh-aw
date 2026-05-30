//go:build !integration

package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBotsToOnBotsCodemod(t *testing.T) {
	codemod := getBotsToOnBotsCodemod()

	assert.Equal(t, "bots-to-on-bots", codemod.ID)
	assert.Equal(t, "Move bots to on.bots", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.10.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestBotsToOnBotsCodemod_SingleLineArray(t *testing.T) {
	codemod := getBotsToOnBotsCodemod()

	content := `---
on:
  issues:
    types: [opened]
bots: [dependabot, renovate]
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"bots": []any{"dependabot", "renovate"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "on:")
	assert.Contains(t, result, "bots: [dependabot, renovate]")
	assert.NotContains(t, result, "\nbots: [dependabot, renovate]")
	// Ensure bots is nested under on:
	lines := strings.Split(result, "\n")
	foundOn := false
	foundBots := false
	for _, line := range lines {
		if line == "on:" {
			foundOn = true
		}
		if foundOn && strings.Contains(line, "bots:") {
			foundBots = true
			// Check that bots line has indentation (nested under on:)
			assert.Greater(t, len(line), len(strings.TrimSpace(line)), "bots should be indented under on:")
			break
		}
	}
	assert.True(t, foundBots, "Should find bots nested under on:")
}

func TestBotsToOnBotsCodemod_MultiLineArray(t *testing.T) {
	codemod := getBotsToOnBotsCodemod()

	content := `---
on:
  issues:
    types: [opened]
bots:
  - dependabot
  - renovate
  - github-actions
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"bots": []any{"dependabot", "renovate", "github-actions"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "on:")
	assert.Contains(t, result, "bots:")
	assert.Contains(t, result, "- dependabot")
	assert.Contains(t, result, "- renovate")
	assert.Contains(t, result, "- github-actions")
}

func TestBotsToOnBotsCodemod_NoOnBlock(t *testing.T) {
	codemod := getBotsToOnBotsCodemod()

	content := `---
bots: [dependabot, renovate]
engine: copilot
---

# Test workflow`

	frontmatter := map[string]any{
		"bots":   []any{"dependabot", "renovate"},
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "on:")
	assert.Contains(t, result, "bots: [dependabot, renovate]")
	// Ensure on: block is created
	lines := strings.Split(result, "\n")
	foundOn := slices.Contains(lines, "on:")
	assert.True(t, foundOn, "Should create new on: block")
}

func TestBotsToOnBotsCodemod_NoChange_NoBots(t *testing.T) {
	codemod := getBotsToOnBotsCodemod()

	content := `---
on:
  issues:
    types: [opened]
engine: copilot
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestBotsToOnBotsCodemod_NoChange_OnBotsExists(t *testing.T) {
	codemod := getBotsToOnBotsCodemod()

	content := `---
on:
  issues:
    types: [opened]
  bots: [dependabot, renovate]
bots: [dependabot, renovate, github-actions]
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
			"bots": []any{"dependabot", "renovate"},
		},
		"bots": []any{"dependabot", "renovate", "github-actions"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
