//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxRunsToMaxTurnsCodemod_Metadata(t *testing.T) {
	codemod := getMaxRunsToMaxTurnsCodemod()

	assert.Equal(t, "max-runs-to-max-turns", codemod.ID)
	assert.Equal(t, "Rename top-level max-runs to max-turns", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.76", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestMaxRunsToMaxTurnsCodemod_NoOpWhenAbsent(t *testing.T) {
	codemod := getMaxRunsToMaxTurnsCodemod()

	content := `---
on: push
max-turns: 10
---`
	frontmatter := map[string]any{
		"on":        "push",
		"max-turns": 10,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMaxRunsToMaxTurnsCodemod_RenamesField(t *testing.T) {
	codemod := getMaxRunsToMaxTurnsCodemod()

	content := `---
on: push
max-runs: 42 # cap
engine: copilot
---`
	frontmatter := map[string]any{
		"on":       "push",
		"max-runs": 42,
		"engine":   "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-turns: 42 # cap")
	assert.NotContains(t, result, "\nmax-runs:")
}

func TestMaxRunsToMaxTurnsCodemod_RemovesDeprecatedWhenBothPresent(t *testing.T) {
	codemod := getMaxRunsToMaxTurnsCodemod()

	content := `---
max-turns: 15
max-runs: 42
---`
	frontmatter := map[string]any{
		"max-turns": 15,
		"max-runs":  42,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-turns: 15")
	assert.NotContains(t, result, "max-runs:")
}
