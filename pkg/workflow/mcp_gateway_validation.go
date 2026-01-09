package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/types"
)

var mcpGatewayValidationLog = logger.New("workflow:mcp_gateway_validation")

// MCPGatewayConfigValidation represents the root configuration structure for the MCP gateway
// as defined in the MCP Gateway Specification v1.0.0 section 4.1
// This type is specifically for validation and uses a simpler structure than the full MCPServerConfig
type MCPGatewayConfigValidation struct {
	MCPServers map[string]MCPServerConfigValidation `json:"mcpServers"`
	Gateway    *GatewayConfigValidation             `json:"gateway,omitempty"`
}

// MCPServerConfigValidation represents a single MCP server configuration for validation
// Supports both stdio (containerized) and http transports per spec section 3.2
// This embeds BaseMCPServerConfig and adds validation-specific fields
type MCPServerConfigValidation struct {
	types.BaseMCPServerConfig

	// Additional fields for validation
	Tools     []string `json:"tools,omitempty"`      // Allowed tools (copilot specific)
	ProxyArgs []string `json:"proxy-args,omitempty"` // Proxy arguments
	Registry  string   `json:"registry,omitempty"`   // Registry URL
	Allowed   []string `json:"allowed,omitempty"`    // Allowed tools list
	Toolsets  []string `json:"toolsets,omitempty"`   // GitHub MCP toolsets
}

// GatewayConfigValidation represents the optional gateway section
// as defined in spec section 4.1.3
type GatewayConfigValidation struct {
	Port           int    `json:"port"`                     // HTTP server port
	APIKey         string `json:"apiKey,omitempty"`         // API key for authentication
	Domain         string `json:"domain,omitempty"`         // Gateway domain
	StartupTimeout int    `json:"startupTimeout,omitempty"` // Server startup timeout in seconds
	ToolTimeout    int    `json:"toolTimeout,omitempty"`    // Tool invocation timeout in seconds
}

// ValidateMCPGatewayJSON validates that a JSON string conforms to the MCP Gateway Specification v1.0.0
// Returns an error if the configuration is invalid
func ValidateMCPGatewayJSON(jsonStr string) error {
	mcpGatewayValidationLog.Print("Validating MCP gateway JSON configuration")

	// Parse JSON into config structure
	var config MCPGatewayConfigValidation
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		mcpGatewayValidationLog.Printf("JSON parsing failed: %v", err)
		return fmt.Errorf("invalid JSON format: %w. The configuration must be valid JSON conforming to the MCP Gateway Specification", err)
	}

	// Validate mcpServers section exists
	if config.MCPServers == nil {
		return fmt.Errorf("missing required 'mcpServers' section in configuration")
	}

	// Validate each server configuration
	for serverName, serverConfig := range config.MCPServers {
		if err := validateServerConfig(serverName, serverConfig); err != nil {
			return err
		}
	}

	// Validate gateway configuration if present
	if config.Gateway != nil {
		if err := validateGatewayConfig(config.Gateway); err != nil {
			return err
		}
	}

	mcpGatewayValidationLog.Print("MCP gateway JSON validation successful")
	return nil
}

// validateServerConfig validates a single MCP server configuration
func validateServerConfig(serverName string, config MCPServerConfigValidation) error {
	mcpGatewayValidationLog.Printf("Validating server config: %s", serverName)

	// Infer type if not explicitly set
	inferredType := config.Type
	if inferredType == "" {
		if config.URL != "" {
			inferredType = "http"
		} else if config.Container != "" || config.Command != "" {
			inferredType = "stdio"
		} else {
			return fmt.Errorf("server '%s': unable to determine type (stdio or http). Must specify 'type', 'url', 'container', or 'command'", serverName)
		}
		mcpGatewayValidationLog.Printf("Inferred type for server '%s': %s", serverName, inferredType)
	}

	// Normalize "local" to "stdio"
	if inferredType == "local" {
		inferredType = "stdio"
	}

	// Validate based on type
	switch inferredType {
	case "stdio":
		return validateStdioServer(serverName, config)
	case "http":
		return validateHTTPServer(serverName, config)
	default:
		return fmt.Errorf("server '%s': unsupported type '%s'. Valid types are: stdio, http", serverName, config.Type)
	}
}

// validateStdioServer validates a stdio (containerized) MCP server configuration
// Per MCP Gateway Specification v1.0.0 section 3.2.1: stdio servers MUST be containerized
func validateStdioServer(serverName string, config MCPServerConfigValidation) error {
	mcpGatewayValidationLog.Printf("Validating stdio server: %s", serverName)

	// Check for docker command pattern (transformed container configuration)
	hasDockerCommand := config.Command == "docker" && len(config.Args) > 0
	hasContainer := config.Container != ""

	// If it has neither container nor docker command, check if it's invalid direct command execution
	if !hasContainer && !hasDockerCommand {
		// Check for command-based execution (not supported)
		if config.Command != "" {
			return fmt.Errorf(
				"server '%s': direct command execution is NOT supported per MCP Gateway Specification v1.0.0 section 3.2.1. "+
					"Stdio-based MCP servers MUST be containerized. "+
					"Please specify a 'container' field with a Docker image instead of using 'command'. "+
					"Example: \"container\": \"node:lts-alpine\"",
				serverName,
			)
		}

		// No container, no docker command, no command - missing required field
		return fmt.Errorf(
			"server '%s': stdio type requires 'container' field. "+
				"Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized. "+
				"Example: \"container\": \"ghcr.io/example/mcp-server:latest\"",
			serverName,
		)
	}

	// Validate mount format if present (format: "source:dest:mode" where mode is "ro" or "rw")
	for _, mount := range config.Mounts {
		parts := strings.Split(mount, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return fmt.Errorf(
				"server '%s': invalid mount format '%s'. Expected format: \"source:dest\" or \"source:dest:mode\" where mode is 'ro' or 'rw'",
				serverName, mount,
			)
		}
		if len(parts) == 3 && parts[2] != "ro" && parts[2] != "rw" {
			return fmt.Errorf(
				"server '%s': invalid mount mode '%s' in '%s'. Mode must be 'ro' (read-only) or 'rw' (read-write)",
				serverName, parts[2], mount,
			)
		}
	}

	return nil
}

// validateHTTPServer validates an HTTP MCP server configuration
func validateHTTPServer(serverName string, config MCPServerConfigValidation) error {
	mcpGatewayValidationLog.Printf("Validating HTTP server: %s", serverName)

	// URL is required for HTTP servers
	if config.URL == "" {
		return fmt.Errorf(
			"server '%s': http type requires 'url' field. "+
				"HTTP MCP servers must specify a URL endpoint. "+
				"Example:\n"+
				"  \"%s\": {\n"+
				"    \"type\": \"http\",\n"+
				"    \"url\": \"https://api.example.com/mcp\",\n"+
				"    \"headers\": {\n"+
				"      \"Authorization\": \"Bearer ${API_KEY}\"\n"+
				"    }\n"+
				"  }",
			serverName, serverName,
		)
	}

	// Validate URL format (basic check for http/https)
	if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		return fmt.Errorf(
			"server '%s': invalid URL '%s'. URLs must start with http:// or https://",
			serverName, config.URL,
		)
	}

	return nil
}

// validateGatewayConfig validates the optional gateway configuration section
func validateGatewayConfig(config *GatewayConfigValidation) error {
	mcpGatewayValidationLog.Print("Validating gateway configuration")

	// Port must be valid if specified
	if config.Port != 0 && (config.Port < 1 || config.Port > 65535) {
		return fmt.Errorf("gateway: invalid port %d. Port must be between 1 and 65535", config.Port)
	}

	// Timeouts must be positive if specified
	if config.StartupTimeout < 0 {
		return fmt.Errorf("gateway: invalid startupTimeout %d. Timeout must be non-negative", config.StartupTimeout)
	}
	if config.ToolTimeout < 0 {
		return fmt.Errorf("gateway: invalid toolTimeout %d. Timeout must be non-negative", config.ToolTimeout)
	}

	return nil
}

// ValidateMCPGatewayConfig validates an already-parsed MCPGatewayConfigValidation structure
// This is useful when the config has been programmatically constructed rather than parsed from JSON
func ValidateMCPGatewayConfig(config *MCPGatewayConfigValidation) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate mcpServers section exists
	if config.MCPServers == nil {
		return fmt.Errorf("missing required 'mcpServers' section in configuration")
	}

	// Validate each server configuration
	for serverName, serverConfig := range config.MCPServers {
		if err := validateServerConfig(serverName, serverConfig); err != nil {
			return err
		}
	}

	// Validate gateway configuration if present
	if config.Gateway != nil {
		if err := validateGatewayConfig(config.Gateway); err != nil {
			return err
		}
	}

	return nil
}
