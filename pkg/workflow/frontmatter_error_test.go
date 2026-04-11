//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindFrontmatterFieldLine verifies that the helper correctly locates a
// named key within frontmatter lines and handles edge cases.
func TestFindFrontmatterFieldLine(t *testing.T) {
	tests := []struct {
		name             string
		frontmatterLines []string
		frontmatterStart int
		fieldName        string
		expectedDocLine  int
		description      string
	}{
		{
			name:             "field found at first line",
			frontmatterLines: []string{"engine: copilot", "on: push"},
			frontmatterStart: 2,
			fieldName:        "engine",
			expectedDocLine:  2, // frontmatterStart + 0
			description:      "engine: on the first frontmatter line maps to document line 2",
		},
		{
			name:             "field found after other keys",
			frontmatterLines: []string{"on: push", "permissions:", "  contents: read", "engine: claude"},
			frontmatterStart: 2,
			fieldName:        "engine",
			expectedDocLine:  5, // frontmatterStart + 3
			description:      "engine: after other keys maps to correct document line",
		},
		{
			name:             "field not present returns zero",
			frontmatterLines: []string{"on: push", "permissions:", "  contents: read"},
			frontmatterStart: 2,
			fieldName:        "engine",
			expectedDocLine:  0,
			description:      "absent field should return 0",
		},
		{
			name:             "field with leading whitespace is not matched",
			frontmatterLines: []string{"on: push", "  engine: copilot"}, // indented — not a top-level key
			frontmatterStart: 2,
			fieldName:        "engine",
			expectedDocLine:  0,
			description:      "indented engine: should not be matched as top-level field",
		},
		{
			name:             "frontmatter starts later in the document",
			frontmatterLines: []string{"on: push", "engine: gemini"},
			frontmatterStart: 10,
			fieldName:        "engine",
			expectedDocLine:  11, // frontmatterStart + 1
			description:      "correct line number when frontmatter does not start at line 2",
		},
		{
			name:             "empty frontmatter lines returns zero",
			frontmatterLines: []string{},
			frontmatterStart: 2,
			fieldName:        "engine",
			expectedDocLine:  0,
			description:      "empty frontmatter should return 0",
		},
		{
			name:             "field name that is a prefix of another key is not confused",
			frontmatterLines: []string{"engine_custom: value", "engine: copilot"},
			frontmatterStart: 2,
			fieldName:        "engine",
			expectedDocLine:  3, // line 3 (frontmatterStart + 1), NOT line 2 which has engine_custom
			description:      "engine: should not match engine_custom: (prefix guard via colon suffix)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFrontmatterFieldLine(tt.frontmatterLines, tt.frontmatterStart, tt.fieldName)
			assert.Equal(t, tt.expectedDocLine, got, tt.description)
		})
	}
}

// TestReadSourceContextLines verifies that reading source context lines around a
// target line produces the expected context window for Rust-style error rendering.
func TestReadSourceContextLines(t *testing.T) {
	content := []byte("---\nengine: 123\non: push\n---\n# Workflow")

	tests := []struct {
		name       string
		targetLine int
		wantLen    int
		wantAny    string // at least this substring appears in the joined output
	}{
		{
			name:       "context around engine line",
			targetLine: 2,
			wantLen:    7,
			wantAny:    "engine: 123",
		},
		{
			name:       "context near start of file",
			targetLine: 1,
			wantLen:    7,
			wantAny:    "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := readSourceContextLines(content, tt.targetLine)
			assert.LessOrEqual(t, len(lines), tt.wantLen, "context should not exceed %d lines", tt.wantLen)
			assert.NotEmpty(t, lines, "context should not be empty")

			joined := strings.Join(lines, "\n")
			assert.Contains(t, joined, tt.wantAny,
				"context should contain the target line content")
		})
	}
}
