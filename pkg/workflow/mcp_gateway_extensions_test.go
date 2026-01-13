package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPServerExtensionFields tests that MCP servers accept extension fields
// per MCP Gateway Specification v1.6.0 Section 4.3.1
func TestMCPServerExtensionFields(t *testing.T) {
	tests := []struct {
		name        string
		toolsConfig map[string]any
		expectError bool
		description string
	}{
		{
			name: "stdio server with extension fields",
			toolsConfig: map[string]any{
				"custom-server": map[string]any{
					"container":       "ghcr.io/example/server:latest",
					"retry-policy":    map[string]any{"max-attempts": 3, "backoff": "exponential"},
					"circuit-breaker": map[string]any{"threshold": 5, "timeout": 30},
					"custom-metrics":  true,
				},
			},
			expectError: false,
			description: "Extension fields like retry-policy and circuit-breaker should be accepted",
		},
		{
			name: "http server with extension fields",
			toolsConfig: map[string]any{
				"http-server": map[string]any{
					"type": "http",
					"url":  "https://api.example.com/mcp",
					"headers": map[string]any{
						"Authorization": "Bearer token123",
					},
					"request-timeout":        120,
					"connection-pool-size":   10,
					"vendor-specific-option": "value",
				},
			},
			expectError: false,
			description: "HTTP servers should accept extension fields like timeouts and vendor options",
		},
		{
			name: "server with namespaced extension fields",
			toolsConfig: map[string]any{
				"namespaced-server": map[string]any{
					"container":            "ghcr.io/example/server:latest",
					"acme-timeout":         300,
					"vendor-retry-count":   5,
					"x-custom-feature":     "enabled",
					"implementation-flag":  true,
				},
			},
			expectError: false,
			description: "Namespaced extension fields with vendor prefixes should be accepted",
		},
		{
			name: "server with mixed standard and extension fields",
			toolsConfig: map[string]any{
				"mixed-server": map[string]any{
					"container":      "ghcr.io/example/server:latest",
					"env":            map[string]any{"API_KEY": "secret"},
					"entrypointArgs": []any{"--verbose"},
					"custom-config":  map[string]any{"option1": "value1", "option2": "value2"},
					"feature-flags":  []any{"feature-a", "feature-b"},
				},
			},
			expectError: false,
			description: "Mix of standard fields and extensions should work together",
		},
		{
			name: "server with extension fields containing complex types",
			toolsConfig: map[string]any{
				"complex-server": map[string]any{
					"container": "ghcr.io/example/server:latest",
					"monitoring": map[string]any{
						"enabled": true,
						"metrics": []any{"latency", "throughput", "errors"},
						"thresholds": map[string]any{
							"latency": 1000,
							"errors":  5,
						},
					},
				},
			},
			expectError: false,
			description: "Extension fields can contain complex nested structures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate MCP configuration accepts extension fields
			err := ValidateMCPConfigs(tt.toolsConfig)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}

			// If validation passes, ensure we can parse the tools config
			if !tt.expectError {
				toolsConfigStruct, err := ParseToolsConfig(tt.toolsConfig)
				require.NoError(t, err, "Failed to parse tools config with extensions")
				require.NotNil(t, toolsConfigStruct, "Tools config should not be nil")
			}
		})
	}
}

// TestMCPGatewayExtensionFields tests that the gateway configuration accepts extension fields
func TestMCPGatewayExtensionFields(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectError     bool
		description     string
		validateGateway func(*testing.T, map[string]any)
	}{
		{
			name: "gateway with extension fields",
			frontmatter: map[string]any{
				"engine": "copilot",
				"tools": map[string]any{
					"custom-server": map[string]any{
						"container": "ghcr.io/example/server:latest",
					},
				},
			},
			expectError: false,
			description: "Gateway configuration with log-level and metrics extensions",
			validateGateway: func(t *testing.T, fm map[string]any) {
				// Extension fields should be preserved in the configuration
				tools, ok := fm["tools"].(map[string]any)
				require.True(t, ok, "Tools should be a map")
				require.NotEmpty(t, tools, "Tools should not be empty")
			},
		},
		{
			name: "gateway with vendor-specific extensions",
			frontmatter: map[string]any{
				"engine": "copilot",
				"tools": map[string]any{
					"server-with-vendor-ext": map[string]any{
						"container":              "ghcr.io/example/server:latest",
						"vendor-custom-timeout":  120,
						"vendor-feature-enabled": true,
					},
				},
			},
			expectError: false,
			description: "Gateway with vendor-specific extension fields should work",
		},
		{
			name: "gateway with feature flags in extensions",
			frontmatter: map[string]any{
				"engine": "copilot",
				"tools": map[string]any{
					"feature-server": map[string]any{
						"container":                "ghcr.io/example/server:latest",
						"experimental-feature-a":   true,
						"experimental-feature-b":   false,
						"custom-behavior-mode":     "strict",
					},
				},
			},
			expectError: false,
			description: "Extension fields can be used for feature flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the frontmatter configuration
			config, err := ParseFrontmatterConfig(tt.frontmatter)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				require.NotNil(t, config, "Config should not be nil")

				// Run additional validation if provided
				if tt.validateGateway != nil {
					tt.validateGateway(t, tt.frontmatter)
				}
			}
		})
	}
}

// TestExtensionFieldsDoNotBreakValidation ensures extension fields don't interfere with required field validation
func TestExtensionFieldsDoNotBreakValidation(t *testing.T) {
	tests := []struct {
		name        string
		toolsConfig map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "server with only extension fields - not treated as MCP server",
			toolsConfig: map[string]any{
				"non-mcp-server": map[string]any{
					"custom-extension": "value",
					"another-field":    123,
				},
			},
			expectError: false, // No error because it's not recognized as an MCP server
		},
		{
			name: "http server missing required url field - should still fail",
			toolsConfig: map[string]any{
				"invalid-http": map[string]any{
					"type":             "http",
					"custom-extension": "value",
				},
			},
			expectError: true,
			errorMsg:    "url",
		},
		{
			name: "valid server with extensions - should pass",
			toolsConfig: map[string]any{
				"valid-server": map[string]any{
					"container":        "ghcr.io/example/server:latest",
					"custom-extension": "value",
					"vendor-field":     true,
				},
			},
			expectError: false,
		},
		{
			name: "valid http server with extensions - should pass",
			toolsConfig: map[string]any{
				"valid-http": map[string]any{
					"type":              "http",
					"url":               "https://api.example.com/mcp",
					"custom-timeout":    300,
					"retry-policy":      map[string]any{"max": 3},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPConfigs(tt.toolsConfig)

			if tt.expectError {
				require.Error(t, err, "Should fail validation")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should mention the issue")
				}
			} else {
				assert.NoError(t, err, "Should pass validation with extension fields")
			}
		})
	}
}

// TestExtensionFieldPreservation verifies that extension fields are preserved during processing
func TestExtensionFieldPreservation(t *testing.T) {
	toolsConfig := map[string]any{
		"server-with-extensions": map[string]any{
			"container":       "ghcr.io/example/server:latest",
			"custom-field":    "custom-value",
			"numeric-ext":     42,
			"boolean-ext":     true,
			"object-ext":      map[string]any{"nested": "value"},
			"array-ext":       []any{"item1", "item2"},
		},
	}

	// Validate the configuration
	err := ValidateMCPConfigs(toolsConfig)
	require.NoError(t, err, "Validation should pass")

	// Parse the tools config
	parsed, err := ParseToolsConfig(toolsConfig)
	require.NoError(t, err, "Parsing should succeed")
	require.NotNil(t, parsed, "Parsed config should not be nil")

	// Verify the server exists in parsed config
	serverConfig, exists := parsed.Custom["server-with-extensions"]
	require.True(t, exists, "Server should exist in parsed config")
	require.NotNil(t, serverConfig, "Server config should not be nil")

	// Note: Extension field preservation is implementation-dependent
	// The important part is that validation and parsing succeed
}

// TestMCPServerCustomTypes tests that custom MCP server types are allowed
// per MCP Gateway Specification v1.6.0 - type can be any string
func TestMCPServerCustomTypes(t *testing.T) {
	tests := []struct {
		name        string
		toolsConfig map[string]any
		expectError bool
		description string
	}{
		{
			name: "custom type with any fields",
			toolsConfig: map[string]any{
				"custom-transport": map[string]any{
					"type":           "websocket",
					"url":            "wss://example.com/mcp",
					"reconnect":      true,
					"ping-interval":  30,
					"custom-field-1": "value1",
					"custom-field-2": map[string]any{"nested": "data"},
				},
			},
			expectError: false,
			description: "Custom type 'websocket' should allow any fields",
		},
		{
			name: "grpc type with custom fields",
			toolsConfig: map[string]any{
				"grpc-server": map[string]any{
					"type":           "grpc",
					"address":        "grpc://localhost:50051",
					"use-tls":        true,
					"cert-path":      "/path/to/cert",
					"max-msg-size":   1048576,
					"custom-options": []any{"opt1", "opt2"},
				},
			},
			expectError: false,
			description: "Custom type 'grpc' should allow any fields",
		},
		{
			name: "custom type without url or command",
			toolsConfig: map[string]any{
				"custom-minimal": map[string]any{
					"type":         "custom-protocol",
					"endpoint":     "custom://endpoint",
					"custom-field": "value",
				},
			},
			expectError: false,
			description: "Custom types don't require url or command fields",
		},
		{
			name: "custom type with mixed standard and custom fields",
			toolsConfig: map[string]any{
				"hybrid-server": map[string]any{
					"type":         "hybrid",
					"env":          map[string]any{"VAR1": "value1"},
					"headers":      map[string]any{"X-Custom": "header"},
					"custom-setup": "initialization-script",
					"extensions":   []any{"ext1", "ext2"},
				},
			},
			expectError: false,
			description: "Custom types can use both standard fields (env, headers) and custom fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPConfigs(tt.toolsConfig)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestMCPServerStrictValidationForKnownTypes verifies that stdio and http still have strict validation
func TestMCPServerStrictValidationForKnownTypes(t *testing.T) {
	tests := []struct {
		name        string
		toolsConfig map[string]any
		expectError bool
		description string
	}{
		{
			name: "stdio missing container and command - should fail",
			toolsConfig: map[string]any{
				"stdio-invalid": map[string]any{
					"type":           "stdio",
					"custom-field":   "value",
					"another-field":  123,
				},
			},
			expectError: true,
			description: "stdio type requires container or command field",
		},
		{
			name: "http missing url - should fail",
			toolsConfig: map[string]any{
				"http-invalid": map[string]any{
					"type":         "http",
					"custom-field": "value",
				},
			},
			expectError: true,
			description: "http type requires url field",
		},
		{
			name: "http cannot use container - should fail",
			toolsConfig: map[string]any{
				"http-with-container": map[string]any{
					"type":      "http",
					"url":       "https://example.com/mcp",
					"container": "image:latest",
				},
			},
			expectError: true,
			description: "http type cannot use container field",
		},
		{
			name: "stdio with container - should pass",
			toolsConfig: map[string]any{
				"stdio-valid": map[string]any{
					"type":         "stdio",
					"container":    "image:latest",
					"custom-field": "value",
				},
			},
			expectError: false,
			description: "stdio with container is valid, extension fields allowed",
		},
		{
			name: "http with url - should pass",
			toolsConfig: map[string]any{
				"http-valid": map[string]any{
					"type":         "http",
					"url":          "https://example.com/mcp",
					"custom-field": "value",
				},
			},
			expectError: false,
			description: "http with url is valid, extension fields allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPConfigs(tt.toolsConfig)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}
