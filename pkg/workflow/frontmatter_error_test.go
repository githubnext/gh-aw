//go:build !integration

package workflow

import (
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
