//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRunIDsFromStdin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single run ID",
			input:    "1234567890\n",
			expected: []string{"1234567890"},
		},
		{
			name:     "multiple run IDs",
			input:    "1234567890\n9876543210\n1111111111\n",
			expected: []string{"1234567890", "9876543210", "1111111111"},
		},
		{
			name:     "run URLs",
			input:    "https://github.com/owner/repo/actions/runs/1234567890\nhttps://github.com/owner/repo/actions/runs/9876543210\n",
			expected: []string{"https://github.com/owner/repo/actions/runs/1234567890", "https://github.com/owner/repo/actions/runs/9876543210"},
		},
		{
			name:     "blank lines are ignored",
			input:    "\n1234567890\n\n9876543210\n\n",
			expected: []string{"1234567890", "9876543210"},
		},
		{
			name:     "comment lines are ignored",
			input:    "# This is a comment\n1234567890\n# Another comment\n9876543210\n",
			expected: []string{"1234567890", "9876543210"},
		},
		{
			name:     "lines are trimmed",
			input:    "  1234567890  \n  9876543210  \n",
			expected: []string{"1234567890", "9876543210"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "only blank lines and comments",
			input:    "\n# comment\n\n# another\n",
			expected: nil,
		},
		{
			name:     "no trailing newline",
			input:    "1234567890",
			expected: []string{"1234567890"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := readRunIDsFromStdin(r)
			require.NoError(t, err, "readRunIDsFromStdin should not return an error")
			assert.Equal(t, tt.expected, got, "URLs should match expected values")
		})
	}
}
