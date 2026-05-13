//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDiscussionTriggerCategoriesLowercaseCodemod(t *testing.T) {
	codemod := getDiscussionTriggerCategoriesLowercaseCodemod()

	assert.Equal(t, "discussion-trigger-categories-lowercase", codemod.ID)
	assert.Equal(t, "Lowercase discussion trigger category values", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestDiscussionTriggerCategoriesCodemod_LowercasesMixedCaseValues(t *testing.T) {
	codemod := getDiscussionTriggerCategoriesLowercaseCodemod()

	content := `---
on:
  discussion:
    types:
      - Agentic Workflows
  discussion_comment:
    types: [General]
---

# Test`

	frontmatter := map[string]any{
		"on": map[string]any{
			"discussion": map[string]any{
				"types": []any{"Agentic Workflows"},
			},
			"discussion_comment": map[string]any{
				"types": []any{"General"},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "- agentic workflows")
	assert.Contains(t, result, "types: [general]")
	assert.NotContains(t, result, "- Agentic Workflows")
	assert.NotContains(t, result, "types: [General]")
}

func TestDiscussionTriggerCategoriesCodemod_NoOpWhenAlreadyLowercase(t *testing.T) {
	codemod := getDiscussionTriggerCategoriesLowercaseCodemod()

	content := `---
on:
  discussion:
    types:
      - agentic workflows
  discussion_comment:
    types: [general]
---

# Test`

	frontmatter := map[string]any{
		"on": map[string]any{
			"discussion": map[string]any{
				"types": []any{"agentic workflows"},
			},
			"discussion_comment": map[string]any{
				"types": []any{"general"},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestDiscussionTriggerCategoriesCodemod_LowercasesQuotedOnAndTriggerKeys(t *testing.T) {
	codemod := getDiscussionTriggerCategoriesLowercaseCodemod()

	content := `---
"on": # workflow triggers
  "discussion": # category trigger
    types:
      - Agentic Workflows
  "discussion_comment": # comment category
    types: [General]
---

# Test`

	frontmatter := map[string]any{
		"on": map[string]any{
			"discussion": map[string]any{
				"types": []any{"Agentic Workflows"},
			},
			"discussion_comment": map[string]any{
				"types": []any{"General"},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "- agentic workflows")
	assert.Contains(t, result, "types: [general]")
	assert.NotContains(t, result, "- Agentic Workflows")
	assert.NotContains(t, result, "types: [General]")
}
