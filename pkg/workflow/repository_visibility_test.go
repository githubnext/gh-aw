//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRepositoryVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{
			name:     "visibility field public",
			payload:  `{"visibility":"public","private":false}`,
			expected: "public",
		},
		{
			name:     "visibility field internal",
			payload:  `{"visibility":"internal","private":true}`,
			expected: "internal",
		},
		{
			name:     "fallback to private boolean true",
			payload:  `{"private":true}`,
			expected: "private",
		},
		{
			name:     "fallback to private boolean false",
			payload:  `{"private":false}`,
			expected: "public",
		},
		{
			name:     "invalid payload",
			payload:  `not-json`,
			expected: "",
		},
		{
			name:     "valid json with invalid field types",
			payload:  `{"visibility":123,"private":"false"}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseRepositoryVisibility([]byte(tt.payload)))
		})
	}
}
