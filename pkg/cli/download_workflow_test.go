//go:build !integration

package cli

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeBase64FileContent(t *testing.T) {
	tests := []struct {
		name     string
		input    func() string // build the raw API-style input
		expected string
		wantErr  bool
	}{
		{
			name: "plain base64 without newlines",
			input: func() string {
				return base64.StdEncoding.EncodeToString([]byte("hello world"))
			},
			expected: "hello world",
		},
		{
			name: "GitHub API style with embedded newlines every 60 chars",
			input: func() string {
				encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
				// Simulate GitHub API line-wrapping at 60 characters
				var sb strings.Builder
				for i, c := range encoded {
					if i > 0 && i%60 == 0 {
						sb.WriteByte('\n')
					}
					sb.WriteRune(c)
				}
				return sb.String()
			},
			expected: "hello world",
		},
		{
			name: "leading and trailing whitespace stripped",
			input: func() string {
				return "  " + base64.StdEncoding.EncodeToString([]byte("trim me")) + "\n"
			},
			expected: "trim me",
		},
		{
			name: "binary content round-trips correctly",
			input: func() string {
				data := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
				return base64.StdEncoding.EncodeToString(data)
			},
			expected: string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE}),
		},
		{
			name:    "invalid base64 returns error",
			input:   func() string { return "!!!not-valid-base64!!!" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBase64FileContent(tt.input())
			if tt.wantErr {
				assert.Error(t, err, "expected an error for invalid base64 input")
				return
			}
			require.NoError(t, err, "unexpected error decoding base64 content")
			assert.Equal(t, tt.expected, string(got), "decoded content should match expected")
		})
	}
}
