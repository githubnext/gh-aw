package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServerConfig represents a parsed MCP server configuration
type MCPServerConfig struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`      // stdio, http, docker
	Command   string            `json:"command"`   // for stdio
	Args      []string          `json:"args"`      // for stdio
	Container string            `json:"container"` // for docker
	URL       string            `json:"url"`       // for http
	Headers   map[string]string `json:"headers"`   // for http
	Env       map[string]string `json:"env"`       // environment variables
	Allowed   []string          `json:"allowed"`   // allowed tools
}

// MCPServerInfo contains the inspection results for an MCP server
type MCPServerInfo struct {
	Config    MCPServerConfig
	Connected bool
	Error     error
	Tools     []*mcp.Tool
	Resources []*mcp.Resource
	Roots     []*mcp.Root
}

// BuiltInMCPServer represents configuration for built-in MCP servers
type BuiltInMCPServer struct {
	Name            string
	DefaultImage    string
	DefaultEnvVars  map[string]string
	EnforcedEnvVars map[string]string // These cannot be overridden
}

// getBuiltInMCPServers returns configuration for built-in MCP servers
func getBuiltInMCPServers() map[string]BuiltInMCPServer {
	return map[string]BuiltInMCPServer{
		"github": {
			Name:         "github",
			DefaultImage: "ghcr.io/github/github-mcp-server:sha-09deac4",
			DefaultEnvVars: map[string]string{
				"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}",
			},
		},
		"playwright": {
			Name:         "playwright",
			DefaultImage: "mcr.microsoft.com/playwright-mcp:latest",
			DefaultEnvVars: map[string]string{
				"HEADLESS": "true",
			},
			EnforcedEnvVars: map[string]string{
				"HEADLESS": "true", // Always headless - cannot be overridden
			},
		},
	}
}

// buildDockerMCPConfig creates a Docker-based MCP configuration for built-in servers
func buildDockerMCPConfig(serverInfo BuiltInMCPServer) MCPServerConfig {
	config := MCPServerConfig{
		Name:    serverInfo.Name,
		Type:    "docker",
		Command: "docker",
		Args:    []string{"run", "-i", "--rm"},
		Env:     make(map[string]string),
	}

	// Add environment variables to Docker args and config
	for envVar := range serverInfo.DefaultEnvVars {
		config.Args = append(config.Args, "-e", envVar)
	}

	// Set environment variable values
	for key, value := range serverInfo.DefaultEnvVars {
		// Special handling for GitHub token
		if key == "GITHUB_PERSONAL_ACCESS_TOKEN" {
			if githubToken, err := GetGitHubToken(); err == nil {
				config.Env[key] = githubToken
			} else {
				config.Env[key] = value // Use placeholder
			}
		} else {
			config.Env[key] = value
		}
	}

	// Add Docker image
	config.Args = append(config.Args, serverInfo.DefaultImage)

	return config
}

// ExtractMCPConfigurations extracts MCP server configurations from workflow frontmatter
func ExtractMCPConfigurations(frontmatter map[string]any, serverFilter string) ([]MCPServerConfig, error) {
	var configs []MCPServerConfig

	// Get tools section from frontmatter
	toolsSection, hasTools := frontmatter["tools"]
	if !hasTools {
		return configs, nil // No tools configured
	}

	tools, ok := toolsSection.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tools section is not a valid map")
	}

	builtInServers := getBuiltInMCPServers()

	for toolName, toolValue := range tools {
		// Apply server filter first
		if serverFilter != "" && !strings.Contains(strings.ToLower(toolName), strings.ToLower(serverFilter)) {
			continue
		}

		// Handle built-in MCP servers (github, playwright)
		if serverInfo, isBuiltIn := builtInServers[toolName]; isBuiltIn {
			config := buildDockerMCPConfig(serverInfo)

			// Apply tool-specific customizations
			if toolConfig, ok := toolValue.(map[string]any); ok {
				// Parse allowed tools
				if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
					if allowedSlice, ok := allowed.([]any); ok {
						for _, item := range allowedSlice {
							if str, ok := item.(string); ok {
								config.Allowed = append(config.Allowed, str)
							}
						}
					}
				}

				// Handle custom MCP configuration (not allowed for Playwright)
				if mcpSection, hasMcp := toolConfig["mcp"]; hasMcp && toolName != "playwright" {
					customConfig, err := ParseMCPConfig(toolName, mcpSection, toolConfig)
					if err != nil {
						return nil, fmt.Errorf("failed to parse custom MCP config for %s: %w", toolName, err)
					}
					customConfig.Name = toolName

					// Apply enforced environment variables (cannot be overridden)
					if customConfig.Env == nil {
						customConfig.Env = make(map[string]string)
					}
					for key, value := range serverInfo.EnforcedEnvVars {
						customConfig.Env[key] = value
					}

					config = customConfig
				}

				// Handle GitHub-specific customizations
				if toolName == "github" {
					if version, exists := toolConfig["docker_image_version"]; exists {
						if versionStr, ok := version.(string); ok {
							dockerImage := "ghcr.io/github/github-mcp-server:" + versionStr
							// Update the Docker image in args
							for i, arg := range config.Args {
								if strings.HasPrefix(arg, "ghcr.io/github/github-mcp-server:") {
									config.Args[i] = dockerImage
									break
								}
							}
						}
					}
				}

				// Handle Playwright-specific customizations
				if toolName == "playwright" {
					if version, exists := toolConfig["docker_image_version"]; exists {
						if versionStr, ok := version.(string); ok {
							dockerImage := "mcr.microsoft.com/playwright-mcp:" + versionStr
							// Update the Docker image in args
							for i, arg := range config.Args {
								if strings.HasPrefix(arg, "mcr.microsoft.com/playwright-mcp:") {
									config.Args[i] = dockerImage
									break
								}
							}
						}
					}
				}
			}

			configs = append(configs, config)
		} else {
			// Handle custom MCP tools
			toolConfig, ok := toolValue.(map[string]any)
			if !ok {
				continue
			}

			// Check if it has MCP configuration
			mcpSection, hasMcp := toolConfig["mcp"]
			if !hasMcp {
				continue
			}

			config, err := ParseMCPConfig(toolName, mcpSection, toolConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to parse MCP config for %s: %w", toolName, err)
			}

			configs = append(configs, config)
		}
	}

	return configs, nil
}

// ParseMCPConfig parses MCP configuration from various formats (map or JSON string)
func ParseMCPConfig(toolName string, mcpSection any, toolConfig map[string]any) (MCPServerConfig, error) {
	config := MCPServerConfig{
		Name: toolName,
		Env:  make(map[string]string),
	}

	// Parse allowed tools
	if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
		if allowedSlice, ok := allowed.([]any); ok {
			for _, item := range allowedSlice {
				if str, ok := item.(string); ok {
					config.Allowed = append(config.Allowed, str)
				}
			}
		}
	}

	var mcpConfig map[string]any

	// Handle different MCP section formats
	switch v := mcpSection.(type) {
	case map[string]any:
		mcpConfig = v
	case string:
		// Parse JSON string
		if err := json.Unmarshal([]byte(v), &mcpConfig); err != nil {
			return config, fmt.Errorf("invalid JSON in mcp configuration: %w", err)
		}
	default:
		return config, fmt.Errorf("invalid mcp configuration format")
	}

	// Extract type (required)
	if typeVal, hasType := mcpConfig["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			config.Type = typeStr
		} else {
			return config, fmt.Errorf("type must be a string")
		}
	} else {
		return config, fmt.Errorf("missing required 'type' field")
	}

	// Extract configuration based on type
	switch config.Type {
	case "stdio":
		// Handle container field (simplified Docker run)
		if container, hasContainer := mcpConfig["container"]; hasContainer {
			if containerStr, ok := container.(string); ok {
				config.Container = containerStr
				config.Command = "docker"
				config.Args = []string{"run", "--rm", "-i"}

				// Add environment variables
				if env, hasEnv := mcpConfig["env"]; hasEnv {
					if envMap, ok := env.(map[string]any); ok {
						for key, value := range envMap {
							if valueStr, ok := value.(string); ok {
								config.Args = append(config.Args, "-e", key)
								config.Env[key] = valueStr
							}
						}
					}
				}

				config.Args = append(config.Args, containerStr)
			}
		} else {
			// Handle command and args
			if command, hasCommand := mcpConfig["command"]; hasCommand {
				if commandStr, ok := command.(string); ok {
					config.Command = commandStr
				} else {
					return config, fmt.Errorf("command must be a string")
				}
			} else {
				return config, fmt.Errorf("stdio type requires 'command' or 'container' field")
			}

			if args, hasArgs := mcpConfig["args"]; hasArgs {
				if argsSlice, ok := args.([]any); ok {
					for _, arg := range argsSlice {
						if argStr, ok := arg.(string); ok {
							config.Args = append(config.Args, argStr)
						}
					}
				}
			}
		}

		// Extract environment variables for stdio
		if env, hasEnv := mcpConfig["env"]; hasEnv {
			if envMap, ok := env.(map[string]any); ok {
				for key, value := range envMap {
					if valueStr, ok := value.(string); ok {
						config.Env[key] = valueStr
					}
				}
			}
		}

	case "http":
		if url, hasURL := mcpConfig["url"]; hasURL {
			if urlStr, ok := url.(string); ok {
				config.URL = urlStr
			} else {
				return config, fmt.Errorf("url must be a string")
			}
		} else {
			return config, fmt.Errorf("http type requires 'url' field")
		}

		// Extract headers
		if headers, hasHeaders := mcpConfig["headers"]; hasHeaders {
			if headersMap, ok := headers.(map[string]any); ok {
				config.Headers = make(map[string]string)
				for key, value := range headersMap {
					if valueStr, ok := value.(string); ok {
						config.Headers[key] = valueStr
					}
				}
			}
		}

	default:
		return config, fmt.Errorf("unsupported MCP type: %s", config.Type)
	}

	return config, nil
}
