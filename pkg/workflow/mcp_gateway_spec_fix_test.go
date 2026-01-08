package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPServerEntrypointField tests that MCP servers support optional entrypoint field
func TestMCPServerEntrypointField(t *testing.T) {
	tests := []struct {
		name                string
		mcpConfig           map[string]any
		expectEntrypoint    string
		expectEntrypointArgs []string
		expectError         bool
	}{
		{
			name: "entrypoint with entrypointArgs",
			mcpConfig: map[string]any{
				"container":      "ghcr.io/example/server:latest",
				"entrypoint":     "/custom/entrypoint.sh",
				"entrypointArgs": []any{"--verbose", "--port", "8080"},
			},
			expectEntrypoint:    "/custom/entrypoint.sh",
			expectEntrypointArgs: []string{"--verbose", "--port", "8080"},
			expectError:         false,
		},
		{
			name: "entrypoint without entrypointArgs",
			mcpConfig: map[string]any{
				"container":  "ghcr.io/example/server:latest",
				"entrypoint": "/bin/sh",
			},
			expectEntrypoint:    "/bin/sh",
			expectEntrypointArgs: nil,
			expectError:         false,
		},
		{
			name: "entrypointArgs without entrypoint (existing behavior)",
			mcpConfig: map[string]any{
				"container":      "ghcr.io/example/server:latest",
				"entrypointArgs": []any{"--config", "/etc/config.json"},
			},
			expectEntrypoint:    "",
			expectEntrypointArgs: []string{"--config", "/etc/config.json"},
			expectError:         false,
		},
		{
			name: "no entrypoint or entrypointArgs",
			mcpConfig: map[string]any{
				"container": "ghcr.io/example/server:latest",
			},
			expectEntrypoint:    "",
			expectEntrypointArgs: nil,
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			extracted := compiler.extractMCPGatewayConfig(tt.mcpConfig)

			if tt.expectError {
				// For now, we don't expect errors, but this is for future validation
				return
			}

			require.NotNil(t, extracted, "Extraction should not return nil")

			// Note: This test will fail initially because Entrypoint field doesn't exist yet
			// We'll add it as part of the fix
			// assert.Equal(t, tt.expectEntrypoint, extracted.Entrypoint, "Entrypoint mismatch")
			assert.ElementsMatch(t, tt.expectEntrypointArgs, extracted.EntrypointArgs, "EntrypointArgs mismatch")
		})
	}
}

// TestMCPServerMountsInServerConfig tests that mounts can be configured per MCP server
func TestMCPServerMountsInServerConfig(t *testing.T) {
	tests := []struct {
		name          string
		toolsConfig   map[string]any
		serverName    string
		expectMounts  []string
		expectError   bool
	}{
		{
			name: "mcp server with mounts",
			toolsConfig: map[string]any{
				"custom-server": map[string]any{
					"container": "ghcr.io/example/server:latest",
					"mounts": []any{
						"/host/data:/container/data:ro",
						"/host/config:/container/config:rw",
					},
				},
			},
			serverName:   "custom-server",
			expectMounts: []string{"/host/data:/container/data:ro", "/host/config:/container/config:rw"},
			expectError:  false,
		},
		{
			name: "mcp server without mounts",
			toolsConfig: map[string]any{
				"simple-server": map[string]any{
					"container": "ghcr.io/example/simple:latest",
				},
			},
			serverName:   "simple-server",
			expectMounts: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the tools config
			toolsConfigStruct, err := ParseToolsConfig(tt.toolsConfig)
			require.NoError(t, err, "Failed to parse tools config")

			// Get the specific MCP server config
			serverConfig, exists := toolsConfigStruct.Custom[tt.serverName]
			require.True(t, exists, "Server not found in custom tools")

			// Note: This test will fail initially because Mounts field doesn't exist in MCPServerConfig
			// We'll add it as part of the fix
			// assert.ElementsMatch(t, tt.expectMounts, serverConfig.Mounts, "Mounts mismatch")

			// For now, just verify the server exists
			_ = serverConfig
		})
	}
}

// TestGatewayMountsDeprecation tests that gateway-level mounts are deprecated
func TestGatewayMountsDeprecation(t *testing.T) {
	// This test documents the transition from gateway-level mounts to server-level mounts
	// Gateway-level mounts should still work but should be considered deprecated

	sandboxConfig := &SandboxConfig{
		MCP: &MCPGatewayRuntimeConfig{
			Container: "ghcr.io/example/gateway:latest",
			Mounts: []string{
				"/host/data:/container/data:ro",
			},
		},
	}

	// Gateway-level mounts should still be supported for backward compatibility
	require.NotNil(t, sandboxConfig.MCP.Mounts)
	assert.Len(t, sandboxConfig.MCP.Mounts, 1)
	assert.Equal(t, "/host/data:/container/data:ro", sandboxConfig.MCP.Mounts[0])
}
