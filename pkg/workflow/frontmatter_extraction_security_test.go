//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAgentSandboxConfigVersion(t *testing.T) {
	compiler := &Compiler{}

	t.Run("extracts sandbox.agent.version from object format", func(t *testing.T) {
		agentObj := map[string]any{
			"id":      "awf",
			"version": "v0.30.1",
		}

		config := compiler.extractAgentSandboxConfig(agentObj)
		require.NotNil(t, config, "Should extract agent sandbox config")
		assert.Equal(t, "v0.30.1", config.Version, "Should extract sandbox.agent.version")
	})
}

// TestExtractMCPGatewayConfigPayloadFields tests extraction of payload-related fields
// from MCP gateway frontmatter configuration
func TestExtractMCPGatewayConfigPayloadFields(t *testing.T) {
	compiler := &Compiler{}

	t.Run("extracts payloadDir using camelCase key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":  "ghcr.io/github/gh-aw-mcpg",
			"payloadDir": "/custom/payloads",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, "/custom/payloads", config.PayloadDir, "Should extract payloadDir")
	})

	t.Run("extracts payloadDir using kebab-case key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":   "ghcr.io/github/gh-aw-mcpg",
			"payload-dir": "/custom/payloads",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, "/custom/payloads", config.PayloadDir, "Should extract payload-dir")
	})

	t.Run("extracts payloadPathPrefix using camelCase key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":         "ghcr.io/github/gh-aw-mcpg",
			"payloadPathPrefix": "/workspace/payloads",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, "/workspace/payloads", config.PayloadPathPrefix, "Should extract payloadPathPrefix")
	})

	t.Run("extracts payloadPathPrefix using kebab-case key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":           "ghcr.io/github/gh-aw-mcpg",
			"payload-path-prefix": "/workspace/payloads",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, "/workspace/payloads", config.PayloadPathPrefix, "Should extract payload-path-prefix")
	})

	t.Run("extracts payloadSizeThreshold using camelCase key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":            "ghcr.io/github/gh-aw-mcpg",
			"payloadSizeThreshold": 65536,
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, 65536, config.PayloadSizeThreshold, "Should extract payloadSizeThreshold")
	})

	t.Run("extracts payloadSizeThreshold using kebab-case key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":              "ghcr.io/github/gh-aw-mcpg",
			"payload-size-threshold": 65536,
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, 65536, config.PayloadSizeThreshold, "Should extract payload-size-threshold")
	})

	t.Run("extracts payloadSizeThreshold as float64 (YAML default numeric type)", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":            "ghcr.io/github/gh-aw-mcpg",
			"payloadSizeThreshold": float64(65536),
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, 65536, config.PayloadSizeThreshold, "Should extract payloadSizeThreshold from float64")
	})

	t.Run("extracts all payload fields together", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":            "ghcr.io/github/gh-aw-mcpg",
			"payloadDir":           "/custom/payloads",
			"payloadPathPrefix":    "/workspace/payloads",
			"payloadSizeThreshold": 1048576,
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, "/custom/payloads", config.PayloadDir, "Should extract payloadDir")
		assert.Equal(t, "/workspace/payloads", config.PayloadPathPrefix, "Should extract payloadPathPrefix")
		assert.Equal(t, 1048576, config.PayloadSizeThreshold, "Should extract payloadSizeThreshold")
	})

	t.Run("leaves payload fields zero/empty when not specified", func(t *testing.T) {
		mcpObj := map[string]any{
			"container": "ghcr.io/github/gh-aw-mcpg",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Empty(t, config.PayloadDir, "PayloadDir should be empty when not specified")
		assert.Empty(t, config.PayloadPathPrefix, "PayloadPathPrefix should be empty when not specified")
		assert.Equal(t, 0, config.PayloadSizeThreshold, "PayloadSizeThreshold should be 0 when not specified")
	})
}

// TestExtractMCPGatewayConfigTrustedBots tests extraction of trustedBots from MCP gateway frontmatter
func TestExtractMCPGatewayConfigTrustedBots(t *testing.T) {
	compiler := &Compiler{}

	t.Run("extracts trustedBots using camelCase key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":   "ghcr.io/github/gh-aw-mcpg",
			"trustedBots": []any{"github-actions[bot]", "copilot-swe-agent[bot]"},
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, []string{"github-actions[bot]", "copilot-swe-agent[bot]"}, config.TrustedBots, "Should extract trustedBots")
	})

	t.Run("extracts trustedBots using kebab-case key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":    "ghcr.io/github/gh-aw-mcpg",
			"trusted-bots": []any{"github-actions[bot]"},
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, []string{"github-actions[bot]"}, config.TrustedBots, "Should extract trusted-bots")
	})

	t.Run("leaves trustedBots nil when not specified", func(t *testing.T) {
		mcpObj := map[string]any{
			"container": "ghcr.io/github/gh-aw-mcpg",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Nil(t, config.TrustedBots, "TrustedBots should be nil when not specified")
	})
}

// TestExtractMCPGatewayConfigKeepaliveInterval tests extraction of keepalive-interval from MCP gateway frontmatter
func TestExtractMCPGatewayConfigKeepaliveInterval(t *testing.T) {
	compiler := &Compiler{}

	t.Run("extracts keepaliveInterval using camelCase key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":         "ghcr.io/github/gh-aw-mcpg",
			"keepaliveInterval": 300,
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, 300, config.KeepaliveInterval, "Should extract keepaliveInterval")
	})

	t.Run("extracts keepalive-interval using kebab-case key", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":          "ghcr.io/github/gh-aw-mcpg",
			"keepalive-interval": 600,
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, 600, config.KeepaliveInterval, "Should extract keepalive-interval")
	})

	t.Run("extracts -1 to disable keepalive", func(t *testing.T) {
		mcpObj := map[string]any{
			"container":         "ghcr.io/github/gh-aw-mcpg",
			"keepaliveInterval": -1,
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, -1, config.KeepaliveInterval, "Should extract -1 as keepalive disabled sentinel")
	})

	t.Run("leaves keepaliveInterval as 0 when not specified", func(t *testing.T) {
		mcpObj := map[string]any{
			"container": "ghcr.io/github/gh-aw-mcpg",
		}
		config := compiler.extractMCPGatewayConfig(mcpObj)
		require.NotNil(t, config, "Should extract MCP gateway config")
		assert.Equal(t, 0, config.KeepaliveInterval, "KeepaliveInterval should be 0 when not specified")
	})
}
