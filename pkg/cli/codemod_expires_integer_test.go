//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpiresIntegerToStringCodemod(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

	assert.Equal(t, "expires-integer-to-string", codemod.ID)
	assert.Equal(t, "Convert expires integer to day string", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.13.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestExpiresIntegerCodemod_ConvertsCreateIssue(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

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
	assert.Contains(t, result, "expires: 7d")
}

func TestExpiresIntegerCodemod_ConvertsCreateDiscussion(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

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
	assert.Contains(t, result, "expires: 30d")
}

func TestExpiresIntegerCodemod_ConvertsCreatePullRequest(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

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
	assert.Contains(t, result, "expires: 14d")
}

func TestExpiresIntegerCodemod_AlreadyStringFormat_NoChange(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

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
	codemod := getExpiresIntegerToStringCodemod()

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
	codemod := getExpiresIntegerToStringCodemod()

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
	codemod := getExpiresIntegerToStringCodemod()

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
	assert.Contains(t, result, "expires: 7d  # expire after one week")
}

func TestExpiresIntegerCodemod_PreservesOtherFields(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

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
	assert.Contains(t, result, "expires: 14d")
	assert.Contains(t, result, "max: 5")
	assert.Contains(t, result, "- bug")
}

func TestExpiresIntegerCodemod_MultipleOutputTypes(t *testing.T) {
	codemod := getExpiresIntegerToStringCodemod()

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
	assert.Contains(t, result, "expires: 7d")
	assert.Contains(t, result, "expires: 30d")
}

func TestConvertExpiresLineToString_Integer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		changed  bool
	}{
		{
			name:     "simple integer",
			input:    "    expires: 7",
			expected: "    expires: 7d",
			changed:  true,
		},
		{
			name:     "integer with trailing comment",
			input:    "    expires: 7  # expire after one week",
			expected: "    expires: 7d  # expire after one week",
			changed:  true,
		},
		{
			name:     "already day string",
			input:    "    expires: 7d",
			expected: "    expires: 7d",
			changed:  false,
		},
		{
			name:     "hour string",
			input:    "    expires: 24h",
			expected: "    expires: 24h",
			changed:  false,
		},
		{
			name:     "week string",
			input:    "    expires: 2w",
			expected: "    expires: 2w",
			changed:  false,
		},
		{
			name:     "false value",
			input:    "    expires: false",
			expected: "    expires: false",
			changed:  false,
		},
		{
			name:     "no indentation",
			input:    "expires: 1",
			expected: "expires: 1d",
			changed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed := convertExpiresLineToString(tt.input)
			assert.Equal(t, tt.changed, changed, "changed flag should match")
			assert.Equal(t, tt.expected, result, "converted line should match")
		})
	}
}
