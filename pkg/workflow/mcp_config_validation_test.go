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
			name:      "valid []string - workspace ro mount",
			toolName:  "my-tool",
			mountsRaw: []string{"${GITHUB_WORKSPACE}/data:/data:ro"},
			wantErr:   false,
		},
		{
			name:      "valid []string - runtime rw mount",
			toolName:  "my-tool",
			mountsRaw: []string{"${RUNNER_TEMP}/gh-aw/data:/data:rw"},
			wantErr:   false,
		},
		{
			name:     "valid []any with string items",
			toolName: "my-tool",
			mountsRaw: []any{
				`\${GITHUB_WORKSPACE}/data:/data:ro`,
				"cache-volume:/cache:rw",
			},
			wantErr: false,
		},
		{
			name:      "valid tmp gh-aw mount",
			toolName:  "my-tool",
			mountsRaw: []string{"/tmp/gh-aw/cache:/cache:rw"},
			wantErr:   false,
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
			name:      "empty source path",
			toolName:  "my-tool",
			mountsRaw: []string{":/container/path:ro"},
			wantErr:   true,
			errMsg:    "source path cannot be empty",
		},
		{
			name:      "empty destination path",
			toolName:  "my-tool",
			mountsRaw: []string{"/host/path::ro"},
			wantErr:   true,
			errMsg:    "destination path cannot be empty",
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
				"${GITHUB_WORKSPACE}/data:/data:ro",
				"/invalid/mount",
			},
			wantErr: true,
			errMsg:  "mounts[1]",
		},
		{
			name:     "[]any with non-string items are silently skipped",
			toolName: "my-tool",
			mountsRaw: []any{
				123,                                 // non-string, skipped
				"${GITHUB_WORKSPACE}/data:/data:ro", // valid string
			},
			wantErr: false,
		},
		{
			name:      "reject filesystem root source",
			toolName:  "my-tool",
			mountsRaw: []string{"/:/host_root:rw"},
			wantErr:   true,
			errMsg:    "host mount source \"/\" is not permitted",
		},
		{
			name:      "reject arbitrary absolute source",
			toolName:  "my-tool",
			mountsRaw: []string{"/etc:/host_etc:ro"},
			wantErr:   true,
			errMsg:    "custom MCP server mounts may not use arbitrary absolute host paths",
		},
		{
			name:      "reject docker socket source",
			toolName:  "my-tool",
			mountsRaw: []string{"/var/run/docker.sock:/var/run/docker.sock:rw"},
			wantErr:   true,
			errMsg:    "host mount source \"/var/run/docker.sock\" is not permitted",
		},
		{
			name:      "reject source under docker socket",
			toolName:  "my-tool",
			mountsRaw: []string{"/var/run/docker.sock/child:/sock:ro"},
			wantErr:   true,
			errMsg:    "host mount source \"/var/run/docker.sock/child\" is not permitted",
		},
		{
			name:      "reject workspace traversal",
			toolName:  "my-tool",
			mountsRaw: []string{"${GITHUB_WORKSPACE}/..:/host_parent:ro"},
			wantErr:   true,
			errMsg:    "host mount source \"${GITHUB_WORKSPACE}/..\" is not permitted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPMountsSyntax(tt.toolName, tt.mountsRaw)

			if tt.wantErr {
				require.Error(t, err, "expected an error")
				if tt.errMsg != "" {
					require.ErrorContains(t, err, tt.errMsg,
						"error message should contain %q", tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
