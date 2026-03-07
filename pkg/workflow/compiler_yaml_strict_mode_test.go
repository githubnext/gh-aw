//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEffectiveStrictMode verifies the strict mode resolution priority:
// CLI flag (--strict) > frontmatter strict field > default (true)
func TestEffectiveStrictMode(t *testing.T) {
	tests := []struct {
		name        string
		cliStrict   bool
		frontmatter map[string]any
		expected    bool
	}{
		{
			name:        "CLI flag true overrides frontmatter false",
			cliStrict:   true,
			frontmatter: map[string]any{"strict": false},
			expected:    true,
		},
		{
			name:        "CLI flag true with no frontmatter strict field",
			cliStrict:   true,
			frontmatter: map[string]any{},
			expected:    true,
		},
		{
			name:        "CLI flag false, frontmatter strict true",
			cliStrict:   false,
			frontmatter: map[string]any{"strict": true},
			expected:    true,
		},
		{
			name:        "CLI flag false, frontmatter strict false",
			cliStrict:   false,
			frontmatter: map[string]any{"strict": false},
			expected:    false,
		},
		{
			name:        "CLI flag false, no frontmatter strict field defaults to true",
			cliStrict:   false,
			frontmatter: map[string]any{},
			expected:    true,
		},
		{
			name:        "CLI flag false, nil frontmatter defaults to true",
			cliStrict:   false,
			frontmatter: nil,
			expected:    true,
		},
		{
			name:        "CLI flag false, non-bool strict field in frontmatter defaults to true",
			cliStrict:   false,
			frontmatter: map[string]any{"strict": "yes"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{strictMode: tt.cliStrict}
			result := compiler.effectiveStrictMode(tt.frontmatter)
			assert.Equal(t, tt.expected, result, "effectiveStrictMode should return %v", tt.expected)
		})
	}
}
