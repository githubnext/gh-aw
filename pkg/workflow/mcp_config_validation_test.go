//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateMCPMountsSyntax tests the MCP mount syntax validation function.
func TestValidateMCPMountsSyntax(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		mountsRaw any
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid []string - ro mount",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/data:/data:ro"},
			wantErr:   false,
		},
		{
			name:      "valid []string - rw mount",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/data:/data:rw"},
			wantErr:   false,
		},
		{
			name:     "valid []any with string items",
			toolName: "my-tool",
			mountsRaw: []any{
				"/host/data:/data:ro",
				"/usr/bin/tool:/usr/bin/tool:ro",
			},
			wantErr: false,
		},
		{
			name:      "empty []string",
			toolName:  "my-tool",
			mountsRaw: []string{},
			wantErr:   false,
		},
		{
			name:      "invalid type — neither []any nor []string",
			toolName:  "my-tool",
			mountsRaw: "not-an-array",
			wantErr:   true,
			errMsg:    "must be an array of strings",
		},
		{
			name:      "invalid format — too few parts",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/path:/container/path"},
			wantErr:   true,
			errMsg:    "must follow 'source:destination:mode' format",
		},
		{
			name:      "invalid format — too many parts",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/path:/container/path:ro:extra"},
			wantErr:   true,
			errMsg:    "must follow 'source:destination:mode' format",
		},
		{
			name:      "invalid mode value",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/path:/container/path:invalid"},
			wantErr:   true,
			errMsg:    "mode must be 'ro' or 'rw'",
		},
		{
			name:      "invalid mode uppercase — case sensitive",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/path:/container/path:RO"},
			wantErr:   true,
			errMsg:    "mode must be 'ro' or 'rw'",
		},
		{
			name:      "error message includes tool name",
			toolName:  "special-tool",
			mountsRaw: []string{"/host/path:/container/path"},
			wantErr:   true,
			errMsg:    "special-tool",
		},
		{
			name:     "error message includes mount index",
			toolName: "my-tool",
			mountsRaw: []string{
				"/host/data:/data:ro",
				"/invalid/mount",
			},
			wantErr: true,
			errMsg:  "mounts[1]",
		},
		{
			name:     "[]any with non-string items are silently skipped",
			toolName: "my-tool",
			mountsRaw: []any{
				123,                   // non-string, skipped
				"/host/data:/data:ro", // valid string
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPMountsSyntax(tt.toolName, tt.mountsRaw)

			if tt.wantErr {
				require.Error(t, err, "expected an error")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg,
						"error message should contain %q", tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
