//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateGitHubAppJSON verifies that validateGitHubAppJSON accepts well-formed
// GitHub App configuration JSON with the required identity and key fields, and
// rejects empty, malformed, or incomplete input.
func TestValidateGitHubAppJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		appJSON string
		want    string
	}{
		{
			name:    "empty string",
			appJSON: "",
			want:    "",
		},
		{
			name:    "literal null",
			appJSON: "null",
			want:    "",
		},
		{
			name:    "invalid JSON",
			appJSON: "{not valid json",
			want:    "",
		},
		{
			name:    "JSON array instead of object",
			appJSON: `["client-id", "private-key"]`,
			want:    "",
		},
		{
			name:    "valid with client-id and private-key",
			appJSON: `{"client-id":"abc123","private-key":"-----BEGIN KEY-----"}`,
			want:    `{"client-id":"abc123","private-key":"-----BEGIN KEY-----"}`,
		},
		{
			name:    "valid with app-id and private-key",
			appJSON: `{"app-id":"123456","private-key":"-----BEGIN KEY-----"}`,
			want:    `{"app-id":"123456","private-key":"-----BEGIN KEY-----"}`,
		},
		{
			name:    "valid with both client-id and app-id",
			appJSON: `{"client-id":"abc","app-id":"123","private-key":"key"}`,
			want:    `{"client-id":"abc","app-id":"123","private-key":"key"}`,
		},
		{
			name:    "missing client-id and app-id",
			appJSON: `{"private-key":"-----BEGIN KEY-----"}`,
			want:    "",
		},
		{
			name:    "missing private-key",
			appJSON: `{"client-id":"abc123"}`,
			want:    "",
		},
		{
			name:    "empty object",
			appJSON: `{}`,
			want:    "",
		},
		{
			name:    "client-id null is not a valid credential",
			appJSON: `{"client-id":null,"private-key":"key"}`,
			want:    "",
		},
		{
			name:    "non-string private-key is rejected",
			appJSON: `{"client-id":"abc","private-key":123}`,
			want:    "",
		},
		{
			name:    "unicode values preserved",
			appJSON: `{"client-id":"客户端","private-key":"密钥"}`,
			want:    `{"client-id":"客户端","private-key":"密钥"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := validateGitHubAppJSON(tt.appJSON)
			assert.Equal(t, tt.want, got)
		})
	}
}
