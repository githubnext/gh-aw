//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRateLimitToUserRateLimitCodemod(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	assert.Equal(t, "rate-limit-to-user-rate-limit", codemod.ID)
	assert.Equal(t, "Rename 'rate-limit' to 'user-rate-limit'", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.44", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestRateLimitToUserRateLimitCodemod_RenamesRateLimitAndMaxRuns(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  max-runs: 5
  window: 60
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"max-runs": 5,
			"window":   60,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.Contains(t, result, "user-rate-limit:")
	assert.Contains(t, result, "max-runs-per-window: 5")
	assert.NotContains(t, result, "\nrate-limit:")
	assert.NotContains(t, result, "\n  max-runs:")
}

func TestRateLimitToUserRateLimitCodemod_RenamesLegacyMaxKey(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  max: 5
  window: 60
---`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"max":    5,
			"window": 60,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.Contains(t, result, "user-rate-limit:")
	assert.Contains(t, result, "max-runs-per-window: 5")
	assert.NotContains(t, result, "\n  max:")
}

func TestRateLimitToUserRateLimitCodemod_NoRateLimitField(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	content := `---
on: workflow_dispatch
engine: copilot
---

# No rate limit`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied, "Codemod should not be applied when rate-limit is absent")
	assert.Equal(t, content, result)
}

func TestRateLimitToUserRateLimitCodemod_SkipsWhenBothKeysPresent(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	content := `---
rate-limit:
  max-runs: 5
user-rate-limit:
  max-runs-per-window: 4
---`

	frontmatter := map[string]any{
		"rate-limit": map[string]any{
			"max-runs": 5,
		},
		"user-rate-limit": map[string]any{
			"max-runs-per-window": 4,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestRateLimitToUserRateLimitCodemod_DoesNotRenameOtherMaxRuns(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	content := `---
rate-limit:
  max-runs: 5
  window: 60
concurrency:
  max-runs: 2
---`

	frontmatter := map[string]any{
		"rate-limit": map[string]any{
			"max-runs": 5,
			"window":   60,
		},
		"concurrency": map[string]any{
			"max-runs": 2,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "user-rate-limit:")
	assert.Contains(t, result, "  max-runs-per-window: 5")
	assert.Contains(t, result, "concurrency:\n  max-runs: 2")
}

func TestRateLimitToUserRateLimitCodemod_DoesNotRenameNestedMaxRuns(t *testing.T) {
	codemod := getRateLimitToUserRateLimitCodemod()

	content := `---
rate-limit:
  rules:
    max-runs: 9
  max-runs: 5
  window: 60
---`

	frontmatter := map[string]any{
		"rate-limit": map[string]any{
			"rules": map[string]any{
				"max-runs": 9,
			},
			"max-runs": 5,
			"window":   60,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "user-rate-limit:")
	assert.Contains(t, result, "  max-runs-per-window: 5")
	assert.Contains(t, result, "  rules:\n    max-runs: 9")
	assert.NotContains(t, result, "    max-runs-per-window: 9")
}
