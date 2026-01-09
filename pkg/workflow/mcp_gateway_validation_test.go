package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMCPGatewayJSON_ValidConfigurations(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		reason string
	}{
		{
			name: "valid stdio server with container",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp-server:latest",
						"entrypointArgs": ["--verbose"],
						"env": {
							"API_KEY": "${MY_API_KEY}"
						}
					}
				}
			}`,
			reason: "Basic containerized stdio server with environment variables",
		},
		{
			name: "valid http server",
			json: `{
				"mcpServers": {
					"remote": {
						"type": "http",
						"url": "https://api.example.com/mcp",
						"headers": {
							"Authorization": "Bearer ${TOKEN}"
						}
					}
				}
			}`,
			reason: "HTTP server with authentication headers",
		},
		{
			name: "mixed stdio and http servers",
			json: `{
				"mcpServers": {
					"local-server": {
						"container": "ghcr.io/example/python-mcp:latest",
						"entrypointArgs": ["--config", "/app/config.json"],
						"type": "stdio"
					},
					"remote-server": {
						"type": "http",
						"url": "https://api.example.com/mcp"
					}
				}
			}`,
			reason: "Multiple servers with different transports",
		},
		{
			name: "server with volume mounts",
			json: `{
				"mcpServers": {
					"data-server": {
						"container": "ghcr.io/example/data-mcp:latest",
						"entrypoint": "/custom/entrypoint.sh",
						"entrypointArgs": ["--config", "/app/config.json"],
						"mounts": [
							"/host/data:/container/data:ro",
							"/host/config:/container/config:rw"
						],
						"type": "stdio"
					}
				}
			}`,
			reason: "Stdio server with custom entrypoint and volume mounts",
		},
		{
			name: "server with gateway configuration",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest"
					}
				},
				"gateway": {
					"port": 8080,
					"apiKey": "secret-key",
					"domain": "localhost",
					"startupTimeout": 60,
					"toolTimeout": 120
				}
			}`,
			reason: "Configuration with gateway settings",
		},
		{
			name: "local type (normalized to stdio)",
			json: `{
				"mcpServers": {
					"example": {
						"type": "local",
						"container": "ghcr.io/example/mcp:latest"
					}
				}
			}`,
			reason: "Type 'local' should be normalized to 'stdio'",
		},
		{
			name: "stdio server type explicit",
			json: `{
				"mcpServers": {
					"example": {
						"type": "stdio",
						"command": "docker",
						"args": ["run", "--rm", "-i", "ghcr.io/example/mcp:latest"]
					}
				}
			}`,
			reason: "Explicit stdio type with docker command pattern (transformed container)",
		},
		{
			name: "copilot-style config with tools field",
			json: `{
				"mcpServers": {
					"github": {
						"type": "http",
						"url": "http://localhost:8080/mcp/github",
						"headers": {
							"Authorization": "Bearer token123"
						},
						"tools": ["*"]
					}
				}
			}`,
			reason: "Copilot-specific configuration with tools field",
		},
		{
			name: "mount without mode (valid)",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest",
						"mounts": ["/host/path:/container/path"]
					}
				}
			}`,
			reason: "Volume mount without explicit mode is valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPGatewayJSON(tt.json)
			assert.NoError(t, err, "Expected valid configuration: %s", tt.reason)
		})
	}
}

func TestValidateMCPGatewayJSON_InvalidConfigurations(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedError string
		reason        string
	}{
		{
			name:          "invalid JSON syntax",
			json:          `{"mcpServers": {`,
			expectedError: "invalid JSON format",
			reason:        "Malformed JSON should be caught",
		},
		{
			name:          "missing mcpServers section",
			json:          `{}`,
			expectedError: "missing required 'mcpServers' section",
			reason:        "mcpServers is a required top-level field",
		},
		{
			name: "stdio server with command instead of container",
			json: `{
				"mcpServers": {
					"example": {
						"command": "node server.js",
						"args": ["--port", "3000"]
					}
				}
			}`,
			expectedError: "direct command execution is NOT supported",
			reason:        "Per spec section 3.2.1, stdio servers must be containerized",
		},
		{
			name: "http server missing url",
			json: `{
				"mcpServers": {
					"example": {
						"type": "http",
						"headers": {
							"Authorization": "Bearer token"
						}
					}
				}
			}`,
			expectedError: "http type requires 'url' field",
			reason:        "HTTP servers must specify a URL endpoint",
		},
		{
			name: "server without type or identifiable fields",
			json: `{
				"mcpServers": {
					"example": {
						"env": {
							"KEY": "value"
						}
					}
				}
			}`,
			expectedError: "unable to determine type",
			reason:        "Cannot infer type without container, command, or url",
		},
		{
			name: "unsupported server type",
			json: `{
				"mcpServers": {
					"example": {
						"type": "websocket",
						"url": "ws://localhost:8080"
					}
				}
			}`,
			expectedError: "unsupported type 'websocket'",
			reason:        "Only stdio and http types are supported",
		},
		{
			name: "invalid mount format",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest",
						"mounts": ["invalid-mount-format"]
					}
				}
			}`,
			expectedError: "invalid mount format",
			reason:        "Mount format must be 'source:dest' or 'source:dest:mode'",
		},
		{
			name: "invalid mount mode",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest",
						"mounts": ["/host:/container:invalid"]
					}
				}
			}`,
			expectedError: "invalid mount mode",
			reason:        "Mount mode must be 'ro' or 'rw'",
		},
		{
			name: "invalid URL protocol",
			json: `{
				"mcpServers": {
					"example": {
						"type": "http",
						"url": "ftp://api.example.com/mcp"
					}
				}
			}`,
			expectedError: "URLs must start with http:// or https://",
			reason:        "Only HTTP and HTTPS protocols are supported",
		},
		{
			name: "invalid gateway port",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest"
					}
				},
				"gateway": {
					"port": 70000
				}
			}`,
			expectedError: "invalid port 70000",
			reason:        "Port must be between 1 and 65535",
		},
		{
			name: "negative startup timeout",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest"
					}
				},
				"gateway": {
					"port": 8080,
					"startupTimeout": -10
				}
			}`,
			expectedError: "invalid startupTimeout",
			reason:        "Timeouts must be non-negative",
		},
		{
			name: "negative tool timeout",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest"
					}
				},
				"gateway": {
					"port": 8080,
					"toolTimeout": -5
				}
			}`,
			expectedError: "invalid toolTimeout",
			reason:        "Timeouts must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPGatewayJSON(tt.json)
			require.Error(t, err, "Expected error for: %s", tt.reason)
			assert.Contains(t, err.Error(), tt.expectedError, "Error message should contain expected text")
		})
	}
}

func TestValidateMCPGatewayConfig_StructValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        *MCPGatewayConfigValidation
		expectedError string
	}{
		{
			name:          "nil config",
			config:        nil,
			expectedError: "config cannot be nil",
		},
		{
			name: "valid config",
			config: &MCPGatewayConfigValidation{
				MCPServers: map[string]MCPServerConfigValidation{
					"example": {
						BaseMCPServerConfig: types.BaseMCPServerConfig{
							Container: "ghcr.io/example/mcp:latest",
						},
					},
				},
			},
			expectedError: "",
		},
		{
			name: "config with nil mcpServers",
			config: &MCPGatewayConfigValidation{
				MCPServers: nil,
			},
			expectedError: "missing required 'mcpServers' section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPGatewayConfig(tt.config)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestValidateServerConfig_TypeInference(t *testing.T) {
	tests := []struct {
		name         string
		config       MCPServerConfigValidation
		shouldError  bool
		expectedType string
	}{
		{
			name: "infer stdio from container",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Container: "ghcr.io/example/mcp:latest",
				},
			},
			shouldError:  false,
			expectedType: "stdio",
		},
		{
			name: "infer stdio from command",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "docker",
					Args:    []string{"run", "ghcr.io/example/mcp:latest"},
				},
			},
			shouldError:  false,
			expectedType: "stdio",
		},
		{
			name: "infer http from url",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					URL: "https://api.example.com/mcp",
				},
			},
			shouldError:  false,
			expectedType: "http",
		},
		{
			name: "explicit stdio type",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Type:      "stdio",
					Container: "ghcr.io/example/mcp:latest",
				},
			},
			shouldError:  false,
			expectedType: "stdio",
		},
		{
			name: "normalize local to stdio",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Type:      "local",
					Container: "ghcr.io/example/mcp:latest",
				},
			},
			shouldError:  false,
			expectedType: "stdio",
		},
		{
			name: "cannot infer type",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Env: map[string]string{"KEY": "value"},
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerConfig("test-server", tt.config)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMCPGatewayJSON_RealWorldExamples(t *testing.T) {
	// Test examples from the MCP Gateway Specification Appendix A
	tests := []struct {
		name string
		json string
	}{
		{
			name: "Appendix A.1 - Basic Containerized Stdio Server",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp-server:latest",
						"entrypointArgs": ["--verbose"],
						"env": {
							"API_KEY": "${MY_API_KEY}"
						}
					}
				},
				"gateway": {
					"port": 8080,
					"apiKey": "gateway-secret-token"
				}
			}`,
		},
		{
			name: "Appendix A.2 - Server with Volume Mounts and Custom Entrypoint",
			json: `{
				"mcpServers": {
					"data-server": {
						"container": "ghcr.io/example/data-mcp:latest",
						"entrypoint": "/custom/entrypoint.sh",
						"entrypointArgs": ["--config", "/app/config.json"],
						"mounts": [
							"/host/data:/container/data:ro",
							"/host/config:/container/config:rw"
						],
						"type": "stdio"
					}
				},
				"gateway": {
					"port": 8080,
					"apiKey": "gateway-secret-token"
				}
			}`,
		},
		{
			name: "Appendix A.3 - Mixed Transport Configuration",
			json: `{
				"mcpServers": {
					"local-server": {
						"container": "ghcr.io/example/python-mcp:latest",
						"entrypointArgs": ["--config", "/app/config.json"],
						"type": "stdio"
					},
					"remote-server": {
						"type": "http",
						"url": "https://api.example.com/mcp"
					}
				},
				"gateway": {
					"port": 8080,
					"startupTimeout": 60,
					"toolTimeout": 120
				}
			}`,
		},
		{
			name: "Appendix A.4 - GitHub MCP Server (Containerized)",
			json: `{
				"mcpServers": {
					"github": {
						"container": "ghcr.io/github/github-mcp-server:latest",
						"env": {
							"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
						}
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPGatewayJSON(tt.json)
			assert.NoError(t, err, "Real-world example from spec should be valid")
		})
	}
}

func TestValidateStdioServer_ContainerRequirement(t *testing.T) {
	// Test that stdio servers MUST be containerized per spec section 3.2.1
	tests := []struct {
		name          string
		config        MCPServerConfigValidation
		shouldError   bool
		errorContains string
	}{
		{
			name: "valid - has container",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Container: "ghcr.io/example/mcp:latest",
				},
			},
			shouldError: false,
		},
		{
			name: "valid - transformed docker command",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "docker",
					Args:    []string{"run", "--rm", "-i", "ghcr.io/example/mcp:latest"},
				},
			},
			shouldError: false,
		},
		{
			name: "invalid - command without container",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "node",
					Args:    []string{"server.js"},
				},
			},
			shouldError:   true,
			errorContains: "direct command execution is NOT supported",
		},
		{
			name: "invalid - neither container nor docker command",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Env: map[string]string{"KEY": "value"},
				},
			},
			shouldError:   true,
			errorContains: "requires 'container' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStdioServer("test-server", tt.config)
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHTTPServer_URLRequirement(t *testing.T) {
	tests := []struct {
		name          string
		config        MCPServerConfigValidation
		shouldError   bool
		errorContains string
	}{
		{
			name: "valid - https URL",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					URL: "https://api.example.com/mcp",
				},
			},
			shouldError: false,
		},
		{
			name: "valid - http URL",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					URL: "http://localhost:8080/mcp",
				},
			},
			shouldError: false,
		},
		{
			name: "invalid - missing URL",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Headers: map[string]string{"Authorization": "Bearer token"},
				},
			},
			shouldError:   true,
			errorContains: "requires 'url' field",
		},
		{
			name: "invalid - wrong protocol",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					URL: "ws://localhost:8080",
				},
			},
			shouldError:   true,
			errorContains: "must start with http:// or https://",
		},
		{
			name: "invalid - no protocol",
			config: MCPServerConfigValidation{
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					URL: "localhost:8080",
				},
			},
			shouldError:   true,
			errorContains: "must start with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPServer("test-server", tt.config)
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGatewayConfig_Constraints(t *testing.T) {
	tests := []struct {
		name          string
		config        *GatewayConfigValidation
		shouldError   bool
		errorContains string
	}{
		{
			name: "valid - all fields",
			config: &GatewayConfigValidation{
				Port:           8080,
				APIKey:         "secret",
				Domain:         "localhost",
				StartupTimeout: 30,
				ToolTimeout:    60,
			},
			shouldError: false,
		},
		{
			name: "valid - minimal",
			config: &GatewayConfigValidation{
				Port: 8080,
			},
			shouldError: false,
		},
		{
			name: "invalid - port too low",
			config: &GatewayConfigValidation{
				Port: 0,
			},
			shouldError:   true,
			errorContains: "invalid port",
		},
		{
			name: "invalid - port too high",
			config: &GatewayConfigValidation{
				Port: 70000,
			},
			shouldError:   true,
			errorContains: "invalid port",
		},
		{
			name: "invalid - negative startup timeout",
			config: &GatewayConfigValidation{
				Port:           8080,
				StartupTimeout: -10,
			},
			shouldError:   true,
			errorContains: "invalid startupTimeout",
		},
		{
			name: "invalid - negative tool timeout",
			config: &GatewayConfigValidation{
				Port:        8080,
				ToolTimeout: -5,
			},
			shouldError:   true,
			errorContains: "invalid toolTimeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGatewayConfig(tt.config)
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateMCPGatewayJSON_ErrorMessages verifies that error messages are helpful and actionable
func TestValidateMCPGatewayJSON_ErrorMessages(t *testing.T) {
	tests := []struct {
		name             string
		json             string
		expectedKeywords []string
	}{
		{
			name: "command execution error mentions spec",
			json: `{
				"mcpServers": {
					"example": {
						"command": "node server.js"
					}
				}
			}`,
			expectedKeywords: []string{"NOT supported", "MCP Gateway Specification", "v1.0.0", "section 3.2.1", "containerized", "container"},
		},
		{
			name: "missing url provides example",
			json: `{
				"mcpServers": {
					"example": {
						"type": "http"
					}
				}
			}`,
			expectedKeywords: []string{"requires 'url'", "Example", "type", "url", "headers"},
		},
		{
			name: "mount format error explains format",
			json: `{
				"mcpServers": {
					"example": {
						"container": "ghcr.io/example/mcp:latest",
						"mounts": ["badformat"]
					}
				}
			}`,
			expectedKeywords: []string{"invalid mount format", "source:dest", "mode"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPGatewayJSON(tt.json)
			require.Error(t, err)
			errMsg := err.Error()
			for _, keyword := range tt.expectedKeywords {
				assert.Contains(t, strings.ToLower(errMsg), strings.ToLower(keyword),
					"Error message should contain keyword '%s'", keyword)
			}
		})
	}
}
