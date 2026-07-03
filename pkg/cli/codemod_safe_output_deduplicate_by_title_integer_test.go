//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputDeduplicateByTitleIntegerCodemod(t *testing.T) {
	codemod := getSafeOutputDeduplicateByTitleIntegerCodemod()

	assert.Equal(t, "safe-output-deduplicate-by-title-integer-to-boolean", codemod.ID)
	assert.Equal(t, "Convert legacy deduplicate-by-title integers to booleans", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSafeOutputDeduplicateByTitleIntegerCodemod_ConvertsCreateIssue(t *testing.T) {
	codemod := getSafeOutputDeduplicateByTitleIntegerCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    deduplicate-by-title: 3
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"deduplicate-by-title": 3,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "deduplicate-by-title: true")
	assert.NotContains(t, result, "deduplicate-by-title: 3")
}

func TestSafeOutputDeduplicateByTitleIntegerCodemod_PreservesComment(t *testing.T) {
	codemod := getSafeOutputDeduplicateByTitleIntegerCodemod()

	content := `---
safe-outputs:
  create-issue:
    deduplicate-by-title: 1  # legacy fuzzy matching
---
`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"deduplicate-by-title": 1,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "deduplicate-by-title: true  # legacy fuzzy matching")
}

func TestSafeOutputDeduplicateByTitleIntegerCodemod_NoChangeForBoolean(t *testing.T) {
	codemod := getSafeOutputDeduplicateByTitleIntegerCodemod()

	content := `---
safe-outputs:
  create-issue:
    deduplicate-by-title: true
---
`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"deduplicate-by-title": true,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
