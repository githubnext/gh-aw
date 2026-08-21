//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDiscussionTriggerCategoriesLowercaseCodemod(t *testing.T) {
	t.Parallel()
	codemod := getDiscussionTriggerCategoriesLowercaseCodemod()

	assert.Equal(t, "discussion-trigger-categories-lowercase", codemod.ID)
	assert.Equal(t, "Lowercase discussion trigger category values", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestDiscussionTriggerCategoriesCodemod_LowercasesMixedCaseValues(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestGetBlockMappingKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		line    string
		wantKey string
		wantOK  bool
	}{
		{name: "plain key", line: "on:", wantKey: "on", wantOK: true},
		{name: "quoted key", line: `"on":`, wantKey: "on", wantOK: true},
		{name: "inline comment with extra spaces", line: `"discussion":    # category`, wantKey: "discussion", wantOK: true},
		{name: "inline value", line: "on: push", wantKey: "", wantOK: false},
		{name: "list item", line: "- on:", wantKey: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotKey, gotOK := getBlockMappingKey(tt.line)
			assert.Equal(t, tt.wantKey, gotKey)
			assert.Equal(t, tt.wantOK, gotOK)
		})
	}
}

func TestLowercaseDiscussionTriggerTypesInLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		lines        []string
		wantModified bool
		wantLines    []string
	}{
		{
			name:         "empty input is not modified",
			lines:        []string{},
			wantModified: false,
			wantLines:    []string{},
		},
		{
			name: "no on block leaves lines untouched",
			lines: []string{
				"name: test",
				"jobs:",
				"  build:",
			},
			wantModified: false,
			wantLines: []string{
				"name: test",
				"jobs:",
				"  build:",
			},
		},
		{
			name: "on block without discussion trigger is untouched",
			lines: []string{
				"on:",
				"  push:",
				"    branches: [main]",
			},
			wantModified: false,
			wantLines: []string{
				"on:",
				"  push:",
				"    branches: [main]",
			},
		},
		{
			name: "discussion block-list types are lowercased",
			lines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - Created",
				"      - Answered",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - created",
				"      - answered",
			},
		},
		{
			name: "discussion inline array types are lowercased",
			lines: []string{
				"on:",
				"  discussion:",
				"    types: [Created, Answered]",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  discussion:",
				"    types: [created, answered]",
			},
		},
		{
			name: "discussion_comment trigger is also handled",
			lines: []string{
				"on:",
				"  discussion_comment:",
				"    types:",
				"      - Created",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  discussion_comment:",
				"    types:",
				"      - created",
			},
		},
		{
			name: "already lowercase types are not modified",
			lines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - created",
			},
			wantModified: false,
			wantLines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - created",
			},
		},
		{
			name: "comment and blank lines inside on block are ignored",
			lines: []string{
				"on:",
				"  # a comment",
				"",
				"  discussion:",
				"    types:",
				"      - Created",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  # a comment",
				"",
				"  discussion:",
				"    types:",
				"      - created",
			},
		},
		{
			name: "trigger block ends when a sibling key is reached",
			lines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - Created",
				"  push:",
				"    branches: [main]",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - created",
				"  push:",
				"    branches: [main]",
			},
		},
		{
			name: "on block ends when a top level key is reached",
			lines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - Created",
				"jobs:",
				"  build:",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - created",
				"jobs:",
				"  build:",
			},
		},
		{
			name: "empty types key switches into list-item mode",
			lines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - Answered",
				"      - Created",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				"  discussion:",
				"    types:",
				"      - answered",
				"      - created",
			},
		},
		{
			name: "quoted trigger keys are recognized",
			lines: []string{
				"on:",
				`  "discussion":`,
				"    types:",
				"      - Created",
			},
			wantModified: true,
			wantLines: []string{
				"on:",
				`  "discussion":`,
				"    types:",
				"      - created",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotLines, gotModified := lowercaseDiscussionTriggerTypesInLines(tt.lines)
			assert.Equal(t, tt.wantModified, gotModified)
			assert.Equal(t, tt.wantLines, gotLines)
		})
	}
}
