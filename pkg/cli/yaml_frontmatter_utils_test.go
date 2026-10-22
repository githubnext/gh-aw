//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatterLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		content       string
		expectedLines []string
		expectedMd    string
		shouldErr     bool
	}{
		{
			name: "valid frontmatter",
			content: `---
on: workflow_dispatch
permissions:
  contents: read
---

# Test Workflow

Content here.`,
			expectedLines: []string{
				"on: workflow_dispatch",
				"permissions:",
				"  contents: read",
			},
			expectedMd: "# Test Workflow\n\nContent here.",
			shouldErr:  false,
		},
		{
			name:          "no frontmatter",
			content:       "Just markdown content",
			expectedLines: []string{},
			expectedMd:    "Just markdown content",
			shouldErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, md, err := parseFrontmatterLines(tt.content)

			if tt.shouldErr {
				assert.Error(t, err, "Expected error parsing invalid content")
				return
			}

			require.NoError(t, err, "Should parse valid frontmatter")
			assert.Equal(t, tt.expectedLines, lines, "Frontmatter lines should match")
			assert.Equal(t, tt.expectedMd, md, "Markdown content should match")
		})
	}
}

func TestGetIndentation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "no indentation",
			line:     "key: value",
			expected: "",
		},
		{
			name:     "two space indentation",
			line:     "  key: value",
			expected: "  ",
		},
		{
			name:     "four space indentation",
			line:     "    key: value",
			expected: "    ",
		},
		{
			name:     "tab indentation",
			line:     "\tkey: value",
			expected: "\t",
		},
		{
			name:     "mixed indentation",
			line:     "  \t  key: value",
			expected: "  \t  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIndentation(tt.line)
			assert.Equal(t, tt.expected, result, "Indentation should match")
		})
	}
}

func TestIsTopLevelKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "top level key",
			line:     "on: workflow_dispatch",
			expected: true,
		},
		{
			name:     "indented key",
			line:     "  contents: read",
			expected: false,
		},
		{
			name:     "comment",
			line:     "# This is a comment",
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			expected: false,
		},
		{
			name:     "no colon",
			line:     "just text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTopLevelKey(tt.line)
			assert.Equal(t, tt.expected, result, "Top-level key detection should match")
		})
	}
}

func TestApplyFrontmatterLineTransform(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		content     string
		transform   func([]string) ([]string, bool)
		wantApplied bool
		wantContent string
		wantErr     bool
	}{
		{
			name: "transform applied",
			content: `---
on: workflow_dispatch
timeout_minutes: 30
---

# Test`,
			transform: func(lines []string) ([]string, bool) {
				result := make([]string, len(lines))
				modified := false
				for i, line := range lines {
					if strings.Contains(line, "timeout_minutes") {
						result[i] = strings.ReplaceAll(line, "timeout_minutes", "timeout-minutes")
						modified = true
					} else {
						result[i] = line
					}
				}
				return result, modified
			},
			wantApplied: true,
			wantContent: "timeout-minutes: 30",
		},
		{
			name: "no change returns original content",
			content: `---
on: workflow_dispatch
---

# Test`,
			transform: func(lines []string) ([]string, bool) {
				return lines, false
			},
			wantApplied: false,
		},
		{
			name:    "content with no frontmatter is handled gracefully",
			content: "not valid frontmatter at all",
			transform: func(lines []string) ([]string, bool) {
				// transform returns false because there are no lines to modify
				return lines, false
			},
			wantApplied: false,
		},
		{
			name: "markdown body preserved",
			content: `---
on: workflow_dispatch
---

# My Workflow

Some description here.`,
			transform: func(lines []string) ([]string, bool) {
				result := make([]string, len(lines))
				copy(result, lines)
				result[0] = "on: push"
				return result, true
			},
			wantApplied: true,
			wantContent: "# My Workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applied, err := applyFrontmatterLineTransform(tt.content, tt.transform)

			if tt.wantErr {
				assert.Error(t, err, "Expected error")
				return
			}

			require.NoError(t, err, "Should not return an error")
			assert.Equal(t, tt.wantApplied, applied, "Applied status should match")

			if tt.wantApplied {
				assert.Contains(t, result, tt.wantContent, "Transformed content should be present")
			} else {
				assert.Equal(t, tt.content, result, "Original content should be returned when not applied")
			}
		})
	}
}
