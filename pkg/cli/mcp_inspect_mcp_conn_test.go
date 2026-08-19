//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/types"
)

// TestBuildConnectionString exercises all branches of buildConnectionString:
// stdio with container, stdio with args, stdio with only command, http, and
// an unrecognized/default type.
func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name   string
		config parser.RegistryMCPServerConfig
		want   string
	}{
		{
			name: "stdio with container",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio", Container: "my-container"},
			},
			want: "docker: my-container",
		},
		{
			name: "stdio with command and args",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio", Command: "npx", Args: []string{"-y", "server"}},
			},
			want: "cmd: npx -y server",
		},
		{
			name: "stdio with command only",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio", Command: "my-binary"},
			},
			want: "cmd: my-binary",
		},
		{
			name: "stdio prefers container over command/args",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio", Command: "my-binary", Args: []string{"arg"}, Container: "my-container"},
			},
			want: "docker: my-container",
		},
		{
			name: "http returns URL",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "http", URL: "https://example.com/mcp"},
			},
			want: "https://example.com/mcp",
		},
		{
			name: "unrecognized type returns type itself",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "local"},
			},
			want: "local",
		},
		{
			name: "empty type returns empty string",
			config: parser.RegistryMCPServerConfig{
				BaseMCPServerConfig: types.BaseMCPServerConfig{Type: ""},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConnectionString(tt.config)
			if got != tt.want {
				t.Fatalf("buildConnectionString() = %q, want %q", got, tt.want)
			}
		})
	}
}
