//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEffectiveTokensToAICreditsCodemod(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	assert.Equal(t, "effective-tokens-to-ai-credits", codemod.ID)
	assert.Equal(t, "Migrate obsolete effective-token limits to AI credits", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.47", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestEffectiveTokensToAICreditsCodemod_MigratesNumericValues(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: 5M # run budget
max-daily-effective-tokens: 10k  # daily budget
on: workflow_dispatch
---

# Workflow`

	frontmatter := map[string]any{
		"max-effective-tokens":       "5M",
		"max-daily-effective-tokens": "10k",
		"on":                         "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-ai-credits: 500 # run budget")
	assert.Contains(t, result, "max-daily-ai-credits: 1  # daily budget")
	assert.NotContains(t, result, "max-effective-tokens:")
	assert.NotContains(t, result, "max-daily-effective-tokens:")
	assert.Contains(t, result, "\n# Workflow")
}

func TestEffectiveTokensToAICreditsCodemod_NoOpWhenLegacyFieldsAbsent(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-ai-credits: 1000
max-daily-ai-credits: 5000
---`

	frontmatter := map[string]any{
		"max-ai-credits":       1000,
		"max-daily-ai-credits": 5000,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEffectiveTokensToAICreditsCodemod_IdempotentAfterMigration(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-ai-credits: 500
max-daily-ai-credits: 1
---`

	frontmatter := map[string]any{
		"max-ai-credits":       500,
		"max-daily-ai-credits": 1,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEffectiveTokensToAICreditsCodemod_SkipsExpressionValues(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: ${{ inputs.max-effective-tokens }}
max-daily-effective-tokens: 4M
---`

	frontmatter := map[string]any{
		"max-effective-tokens":       "${{ inputs.max-effective-tokens }}",
		"max-daily-effective-tokens": "4M",
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-effective-tokens: ${{ inputs.max-effective-tokens }}")
	assert.Contains(t, result, "max-daily-ai-credits: 400")
	assert.NotContains(t, result, "max-daily-effective-tokens:")
}

func TestEffectiveTokensToAICreditsCodemod_SkipsWhenTargetFieldExists(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: 7M
max-ai-credits: 2500
---`

	frontmatter := map[string]any{
		"max-effective-tokens": "7M",
		"max-ai-credits":       2500,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEffectiveTokensToAICreditsCodemod_MigratesDailyNegativeOne(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-daily-effective-tokens: -1 # disabled
---`

	frontmatter := map[string]any{
		"max-daily-effective-tokens": -1,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-daily-ai-credits: -1 # disabled")
	assert.NotContains(t, result, "max-daily-effective-tokens:")
}

func TestEffectiveTokensToAICreditsCodemod_MigratesRunNegativeOne(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: -1 # disabled
---`

	frontmatter := map[string]any{
		"max-effective-tokens": -1,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-ai-credits: -1 # disabled")
	assert.NotContains(t, result, "max-effective-tokens:")
}

func TestEffectiveTokensToAICreditsCodemod_SkipsValuesBelowOneCredit(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: 9999
max-daily-effective-tokens: 5000
---`

	frontmatter := map[string]any{
		"max-effective-tokens":       9999,
		"max-daily-effective-tokens": 5000,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestEffectiveTokensToAICreditsCodemod_PartialMigrationWhenOnlyOneValueConverts(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: 5000
max-daily-effective-tokens: 10k
---`

	frontmatter := map[string]any{
		"max-effective-tokens":       5000,
		"max-daily-effective-tokens": "10k",
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-effective-tokens: 5000")
	assert.Contains(t, result, "max-daily-ai-credits: 1")
	assert.NotContains(t, result, "max-daily-effective-tokens:")
}

func TestEffectiveTokensToAICreditsCodemod_PartialMigrationWhenOnlyRunValueConverts(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: 10k
max-daily-effective-tokens: 5000
---`

	frontmatter := map[string]any{
		"max-effective-tokens":       "10k",
		"max-daily-effective-tokens": 5000,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-ai-credits: 1")
	assert.NotContains(t, result, "max-effective-tokens:")
	assert.Contains(t, result, "max-daily-effective-tokens: 5000")
}

func TestEffectiveTokensToAICreditsCodemod_MigratesThresholdValue(t *testing.T) {
	codemod := getEffectiveTokensToAICreditsCodemod()

	content := `---
max-effective-tokens: 10000
---`

	frontmatter := map[string]any{
		"max-effective-tokens": 10000,
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "max-ai-credits: 1")
	assert.NotContains(t, result, "max-effective-tokens:")
}
