//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExpandBotNames verifies that "copilot" is expanded to the full set of
// GitHub Copilot bot identifiers and that other bot names pass through unchanged.
func TestExpandBotNames(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty list",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "nil list",
			input:    nil,
			expected: nil,
		},
		{
			name:     "copilot alias expands to all copilot bot names",
			input:    []string{"copilot"},
			expected: []string{"copilot-swe-agent", "Copilot", "copilot"},
		},
		{
			name:     "non-copilot bots pass through unchanged",
			input:    []string{"dependabot[bot]", "renovate[bot]"},
			expected: []string{"dependabot[bot]", "renovate[bot]"},
		},
		{
			name:     "copilot mixed with other bots deduplicates",
			input:    []string{"dependabot[bot]", "copilot", "renovate[bot]"},
			expected: []string{"dependabot[bot]", "copilot-swe-agent", "Copilot", "copilot", "renovate[bot]"},
		},
		{
			name:     "copilot-swe-agent explicit does not double-expand",
			input:    []string{"copilot", "copilot-swe-agent"},
			expected: []string{"copilot-swe-agent", "Copilot", "copilot"},
		},
		{
			name:     "Copilot explicit does not double-expand",
			input:    []string{"copilot", "Copilot"},
			expected: []string{"copilot-swe-agent", "Copilot", "copilot"},
		},
		{
			name:     "no copilot alias — list unchanged",
			input:    []string{"github-actions[bot]"},
			expected: []string{"github-actions[bot]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandBotNames(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
