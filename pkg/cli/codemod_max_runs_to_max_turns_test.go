//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxRunsToMaxTurnsCodemod_AWASTConflict(t *testing.T) {
	t.Parallel()
	content := "---\n{'max-runs': {nested: 42}, 'max-turns': null, nested: {max-runs: 12}} # keep\n---\n\n# Body\n"
	result, applied, err := getMaxRunsToMaxTurnsCodemod().Apply(content, nil)
	require.NoError(t, err)
	require.True(t, applied)
	document, err := aw.Parse(result)
	require.NoError(t, err)
	assert.NotContains(t, document.Frontmatter, "max-runs")
	assert.Contains(t, document.Frontmatter, "max-turns")
	assert.Nil(t, document.Frontmatter["max-turns"])
	assert.Equal(t, map[string]any{"max-runs": uint64(12)}, document.Frontmatter["nested"])
	assert.Contains(t, result, "# keep")
	assert.Contains(t, result, "\n---\n\n# Body\n")
}

func TestMaxRunsToMaxTurnsCodemod_Metadata(t *testing.T) {
	t.Parallel()
	codemod := getMaxRunsToMaxTurnsCodemod()

	assert.Equal(t, "max-runs-to-max-turns", codemod.ID)
	assert.Equal(t, "Rename top-level max-runs to max-turns", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.76", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestMaxRunsToMaxTurnsCodemod_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
