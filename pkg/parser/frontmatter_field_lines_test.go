//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTopLevelFieldLines(t *testing.T) {
	tests := []struct {
		name             string
		yamlContent      string
		frontmatterStart int
		wantFieldLines   map[string]int
	}{
		{
			name: "simple top-level keys",
			yamlContent: `engine: copilot
on: push
permissions: read`,
			frontmatterStart: 2,
			wantFieldLines: map[string]int{
				"engine":      2, // line 1 of YAML + 2 - 1 = 2
				"on":          3,
				"permissions": 4,
			},
		},
		{
			name: "keys with nested content",
			yamlContent: `engine: copilot
tools:
  github:
    toolsets: default
on: push`,
			frontmatterStart: 2,
			wantFieldLines: map[string]int{
				"engine": 2,
				"tools":  3,
				"on":     6,
			},
		},
		{
			name: "skips comments and blank lines",
			yamlContent: `# workflow config
engine: copilot

on: push`,
			frontmatterStart: 2,
			wantFieldLines: map[string]int{
				"engine": 3, // line 2 of YAML (comment is skipped, blank is skipped) + 2 - 1
				"on":     5,
			},
		},
		{
			name:             "empty YAML",
			yamlContent:      "",
			frontmatterStart: 2,
			wantFieldLines:   map[string]int{},
		},
		{
			name: "first occurrence wins for duplicate keys",
			yamlContent: `engine: copilot
engine: claude`,
			frontmatterStart: 2,
			wantFieldLines: map[string]int{
				"engine": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTopLevelFieldLines(tt.yamlContent, tt.frontmatterStart)
			assert.Equal(t, tt.wantFieldLines, got, "Field lines should match")
		})
	}
}

func TestFrontmatterResultFieldLines(t *testing.T) {
	t.Run("field lines populated from content", func(t *testing.T) {
		content := "---\nengine: copilot\ntools:\n  github:\n    toolsets: default\non: push\n---\n# My Workflow\n"
		result, err := ExtractFrontmatterFromContent(content)
		require.NoError(t, err, "Should parse frontmatter without error")
		require.NotNil(t, result.FieldLines, "FieldLines should not be nil")

		// engine is on line 2 (line 1 is '---')
		assert.Equal(t, 2, result.FieldLines["engine"], "engine should be on line 2")
		// tools is on line 3
		assert.Equal(t, 3, result.FieldLines["tools"], "tools should be on line 3")
		// on is on line 6
		assert.Equal(t, 6, result.FieldLines["on"], "on should be on line 6")
		// nested key should NOT appear as top-level
		assert.Zero(t, result.FieldLines["github"], "Nested keys should not appear in FieldLines")
	})

	t.Run("no frontmatter returns nil FieldLines", func(t *testing.T) {
		content := "# Just markdown\n\nNo frontmatter here.\n"
		result, err := ExtractFrontmatterFromContent(content)
		require.NoError(t, err, "Should not error on content without frontmatter")
		// FieldLines is nil when there is no frontmatter
		assert.Nil(t, result.FieldLines, "FieldLines should be nil when there is no frontmatter")
	})

	t.Run("preserves blank frontmatter lines while tracking top level fields", func(t *testing.T) {
		content := "---\nengine: copilot\n\n# comment\non: push\n---\n# Workflow\n"
		result, err := ExtractFrontmatterFromContent(content)
		require.NoError(t, err, "Should parse frontmatter without error")

		assert.Equal(t, []string{"engine: copilot", "", "# comment", "on: push"}, result.FrontmatterLines)
		assert.Equal(t, map[string]int{"engine": 2, "on": 5}, result.FieldLines)
	})
}
