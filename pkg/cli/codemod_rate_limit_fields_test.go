//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRateLimitFieldsCodemod(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	assert.Equal(t, "rate-limit-fields-migration", codemod.ID)
	assert.Equal(t, "Rename rate-limit fields for clarity", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.20.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestRateLimitFieldsCodemod_RenamesMaxAndWindow(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  max: 5
  window: 60
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"max":    5,
			"window": 60,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.True(t, applied, "Codemod should be applied")
	assert.Contains(t, result, "max-runs-per-user: 5")
	assert.Contains(t, result, "max-runs-per-user-window: 60")
	assert.NotContains(t, result, "  max: 5")
	assert.NotContains(t, result, "  window: 60")
}

func TestRateLimitFieldsCodemod_RenamesMaxOnly(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  max: 3
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"max": 3,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.True(t, applied, "Codemod should be applied")
	assert.Contains(t, result, "max-runs-per-user: 3")
	assert.NotContains(t, result, "  max: 3")
}

func TestRateLimitFieldsCodemod_RenamesWindowOnly(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  window: 30
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"window": 30,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.True(t, applied, "Codemod should be applied")
	assert.Contains(t, result, "max-runs-per-user-window: 30")
	assert.NotContains(t, result, "  window: 30")
}

func TestRateLimitFieldsCodemod_NoRateLimitBlock(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.False(t, applied, "Codemod should not be applied")
	assert.Equal(t, content, result, "Content should be unchanged")
}

func TestRateLimitFieldsCodemod_AlreadyRenamed(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  max-runs-per-user: 5
  max-runs-per-user-window: 60
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"max-runs-per-user":        5,
			"max-runs-per-user-window": 60,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.False(t, applied, "Codemod should not be applied when fields already renamed")
	assert.Equal(t, content, result, "Content should be unchanged")
}

func TestRateLimitFieldsCodemod_PreservesOtherFields(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	content := `---
on: workflow_dispatch
rate-limit:
  max: 5
  window: 60
  events:
    - workflow_dispatch
  ignored-roles:
    - admin
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"rate-limit": map[string]any{
			"max":           5,
			"window":        60,
			"events":        []any{"workflow_dispatch"},
			"ignored-roles": []any{"admin"},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.True(t, applied, "Codemod should be applied")
	assert.Contains(t, result, "max-runs-per-user: 5")
	assert.Contains(t, result, "max-runs-per-user-window: 60")
	assert.Contains(t, result, "events:")
	assert.Contains(t, result, "ignored-roles:")
}

func TestRateLimitFieldsCodemod_PreservesFieldsOutsideRateLimitBlock(t *testing.T) {
	codemod := getRateLimitFieldsCodemod()

	// "max" outside the rate-limit block should NOT be renamed
	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    max: 5
rate-limit:
  max: 3
  window: 60
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{"max": 5},
		},
		"rate-limit": map[string]any{
			"max":    3,
			"window": 60,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Should not return an error")
	assert.True(t, applied, "Codemod should be applied")
	// max inside safe-outputs.create-issue is preserved
	assert.Contains(t, result, "    max: 5", "max outside rate-limit block should be unchanged")
	// max inside rate-limit is renamed
	assert.Contains(t, result, "max-runs-per-user: 3")
	assert.Contains(t, result, "max-runs-per-user-window: 60")
}

func TestRateLimitFieldsCodemod_IsRegistered(t *testing.T) {
	codemods := GetAllCodemods()
	var found bool
	for _, c := range codemods {
		if c.ID == "rate-limit-fields-migration" {
			found = true
			break
		}
	}
	assert.True(t, found, "rate-limit-fields-migration codemod should be registered in GetAllCodemods")
}
