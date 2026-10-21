//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpiresIntegerToStringCodemod(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	assert.Equal(t, "expires-integer-to-string", codemod.ID)
	assert.Equal(t, "Convert expires integer to day string", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.13.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestExpiresIntegerCodemod_ConvertsCreateIssue(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    expires: 7
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"expires": 7,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assertPolicyCodemodValue(t, result, "7d", "safe-outputs", "create-issue", "expires")
}

func TestExpiresIntegerCodemod_ConvertsCreateDiscussion(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-discussion:
    expires: 30
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-discussion": map[string]any{
				"expires": 30,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assertPolicyCodemodValue(t, result, "30d", "safe-outputs", "create-discussion", "expires")
}

func TestExpiresIntegerCodemod_ConvertsCreatePullRequest(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-pull-request:
    expires: 14
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-pull-request": map[string]any{
				"expires": 14,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assertPolicyCodemodValue(t, result, "14d", "safe-outputs", "create-pull-request", "expires")
}

func TestExpiresIntegerCodemod_AlreadyStringFormat_NoChange(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    expires: 7d
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"expires": "7d",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestExpiresIntegerCodemod_HourStringFormat_NoChange(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    expires: 24h
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"expires": "24h",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestExpiresIntegerCodemod_NoSafeOutputs_NoChange(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

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

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestExpiresIntegerCodemod_PreservesComment(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    expires: 7  # expire after one week
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"expires": 7,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assertPolicyCodemodValue(t, result, "7d", "safe-outputs", "create-issue", "expires")
	assert.Contains(t, result, "# expire after one week")
}

func TestExpiresIntegerCodemod_PreservesOtherFields(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    max: 5
    expires: 14
    labels:
      - bug
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"max":     5,
				"expires": 14,
				"labels":  []any{"bug"},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assertPolicyCodemodValue(t, result, "14d", "safe-outputs", "create-issue", "expires")
	assert.Contains(t, result, "max: 5")
	assert.Contains(t, result, "- bug")
}

func TestExpiresIntegerCodemod_MultipleOutputTypes(t *testing.T) {
	t.Parallel()
	codemod := getExpiresIntegerToDayStringCodemod()

	content := `---
on: workflow_dispatch
safe-outputs:
  create-issue:
    expires: 7
  create-discussion:
    expires: 30
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"expires": 7,
			},
			"create-discussion": map[string]any{
				"expires": 30,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assertPolicyCodemodValue(t, result, "7d", "safe-outputs", "create-issue", "expires")
	assertPolicyCodemodValue(t, result, "30d", "safe-outputs", "create-discussion", "expires")
}
