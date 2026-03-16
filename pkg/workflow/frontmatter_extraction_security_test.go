//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractFirewallConfig tests the extraction of firewall configuration from frontmatter
func TestExtractFirewallConfig(t *testing.T) {
	compiler := &Compiler{}

	t.Run("extracts ssl-bump boolean field", func(t *testing.T) {
		firewallObj := map[string]any{
			"ssl-bump": true,
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.True(t, config.Enabled, "Should be enabled")
		assert.True(t, config.SSLBump, "Should have ssl-bump enabled")
	})

	t.Run("extracts allow-urls string array", func(t *testing.T) {
		firewallObj := map[string]any{
			"ssl-bump": true,
			"allow-urls": []any{
				"https://github.com/githubnext/*",
				"https://api.github.com/repos/*",
			},
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.True(t, config.SSLBump, "Should have ssl-bump enabled")
		assert.Len(t, config.AllowURLs, 2, "Should have 2 allow-urls")
		assert.Equal(t, "https://github.com/githubnext/*", config.AllowURLs[0], "First URL should match")
		assert.Equal(t, "https://api.github.com/repos/*", config.AllowURLs[1], "Second URL should match")
	})

	t.Run("extracts all fields together", func(t *testing.T) {
		firewallObj := map[string]any{
			"args":       []any{"--custom-arg", "value"},
			"version":    "v1.0.0",
			"log-level":  "debug",
			"ssl-bump":   true,
			"allow-urls": []any{"https://example.com/*"},
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.True(t, config.Enabled, "Should be enabled")
		assert.Len(t, config.Args, 2, "Should have 2 args")
		assert.Equal(t, "v1.0.0", config.Version, "Should extract version")
		assert.Equal(t, "debug", config.LogLevel, "Should extract log-level")
		assert.True(t, config.SSLBump, "Should have ssl-bump enabled")
		assert.Len(t, config.AllowURLs, 1, "Should have 1 allow-url")
		assert.Equal(t, "https://example.com/*", config.AllowURLs[0], "Should extract allow-url")
	})

	t.Run("ssl-bump defaults to false when not specified", func(t *testing.T) {
		firewallObj := map[string]any{
			"version": "v1.0.0",
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.False(t, config.SSLBump, "ssl-bump should default to false")
	})

	t.Run("allow-urls defaults to empty when not specified", func(t *testing.T) {
		firewallObj := map[string]any{
			"ssl-bump": true,
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.Nil(t, config.AllowURLs, "allow-urls should be nil when not specified")
	})

	t.Run("handles non-string values in allow-urls gracefully", func(t *testing.T) {
		firewallObj := map[string]any{
			"allow-urls": []any{
				"https://github.com/*",
				123, // Invalid: number instead of string
				"https://api.github.com/*",
			},
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.Len(t, config.AllowURLs, 2, "Should skip non-string values")
		assert.Equal(t, "https://github.com/*", config.AllowURLs[0], "First valid URL should be extracted")
		assert.Equal(t, "https://api.github.com/*", config.AllowURLs[1], "Second valid URL should be extracted")
	})

	t.Run("handles non-boolean ssl-bump gracefully", func(t *testing.T) {
		firewallObj := map[string]any{
			"ssl-bump": "true", // String instead of boolean
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.False(t, config.SSLBump, "Should ignore non-boolean ssl-bump value")
	})

	t.Run("handles non-array allow-urls gracefully", func(t *testing.T) {
		firewallObj := map[string]any{
			"allow-urls": "https://github.com/*", // String instead of array
		}

		config := compiler.extractFirewallConfig(firewallObj)
		require.NotNil(t, config, "Should extract firewall config")
		assert.Nil(t, config.AllowURLs, "Should ignore non-array allow-urls value")
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
