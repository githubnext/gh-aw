package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ensureLocalhostDomains ensures that localhost and 127.0.0.1 are always included
// in the allowed domains list for Playwright, even when custom domains are specified
func ensureLocalhostDomains(domains []string) []string {
	hasLocalhost := false
	hasLoopback := false

	for _, domain := range domains {
		if domain == "localhost" {
			hasLocalhost = true
		}
		if domain == "127.0.0.1" {
			hasLoopback = true
		}
	}

	result := make([]string, 0, len(domains)+2)

	// Always add localhost domains first
	if !hasLocalhost {
		result = append(result, "localhost")
	}
	if !hasLoopback {
		result = append(result, "127.0.0.1")
	}

	// Add the rest of the domains
	result = append(result, domains...)

	return result
}

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

// ExtractMCPConfigurations extracts MCP server configurations from workflow frontmatter
func ExtractMCPConfigurations(frontmatter map[string]any, serverFilter string) ([]MCPServerConfig, error) {
	var configs []MCPServerConfig

	// Check for safe-outputs configuration first (built-in MCP)
	if safeOutputsSection, hasSafeOutputs := frontmatter["safe-outputs"]; hasSafeOutputs {
		// Apply server filter if specified
		if serverFilter == "" || strings.Contains("safe-outputs", strings.ToLower(serverFilter)) {
			config := MCPServerConfig{
				Name: "safe-outputs",
				Type: "stdio",
				// Command and args will be set up dynamically when the server is started
				Command: "node",
				Env:     make(map[string]string),
			}

			// Parse safe-outputs configuration to determine enabled tools
			if safeOutputsMap, ok := safeOutputsSection.(map[string]any); ok {
				for toolType := range safeOutputsMap {
					// Convert tool types to the actual MCP tool names
					switch toolType {
					case "create-issue":
						config.Allowed = append(config.Allowed, "create-issue")
					case "create-discussion":
						config.Allowed = append(config.Allowed, "create-discussion")
					case "add-issue-comment":
						config.Allowed = append(config.Allowed, "add-issue-comment")
					case "create-pull-request":
						config.Allowed = append(config.Allowed, "create-pull-request")
					case "create-pull-request-review-comment":
						config.Allowed = append(config.Allowed, "create-pull-request-review-comment")
					case "create-code-scanning-alert":
						config.Allowed = append(config.Allowed, "create-code-scanning-alert")
					case "add-issue-labels":
						config.Allowed = append(config.Allowed, "add-issue-labels")
					case "update-issue":
						config.Allowed = append(config.Allowed, "update-issue")
					case "push-to-pr-branch":
						config.Allowed = append(config.Allowed, "push-to-pr-branch")
					case "missing-tool":
						config.Allowed = append(config.Allowed, "missing-tool")
					}
				}
			}

			configs = append(configs, config)
		}
	}

	// Get tools section from frontmatter
	toolsSection, hasTools := frontmatter["tools"]
	if !hasTools {
		return configs, nil // No tools configured, but we might have safe-outputs
	}

	tools, ok := toolsSection.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tools section is not a valid map")
	}

	for toolName, toolValue := range tools {
		// Handle built-in MCP tools (github and playwright)
		if toolName == "github" || toolName == "playwright" {
			// Apply server filter if specified
			if serverFilter != "" && !strings.Contains(strings.ToLower(toolName), strings.ToLower(serverFilter)) {
				continue
			}

			if toolName == "github" {
				// Handle GitHub MCP server - always use Docker by default
				config := MCPServerConfig{
					Name:    "github",
					Type:    "docker", // GitHub defaults to Docker (local containerized)
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
						"ghcr.io/github/github-mcp-server:sha-09deac4",
					},
					Env: make(map[string]string),
				}

				// Try to get GitHub token, but don't fail if it's not available
				// This allows tests to run without GitHub authentication
				if githubToken, err := GetGitHubToken(); err == nil {
					config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = githubToken
				} else {
					// Set a placeholder that will be validated later during connection
					config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "${GITHUB_TOKEN_REQUIRED}"
				}

				// Check for custom GitHub configuration
				if toolConfig, ok := toolValue.(map[string]any); ok {
					if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
						if allowedSlice, ok := allowed.([]any); ok {
							for _, item := range allowedSlice {
								if str, ok := item.(string); ok {
									config.Allowed = append(config.Allowed, str)
								}
							}
						}
					}

					// Check for custom Docker image version
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

				configs = append(configs, config)
			} else if toolName == "playwright" {
				// Handle Playwright MCP server - always use Docker by default
				config := MCPServerConfig{
					Name:    "playwright",
					Type:    "docker", // Playwright defaults to Docker (containerized)
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "--shm-size=2gb", "--cap-add=SYS_ADMIN",
						"-e", "PLAYWRIGHT_ALLOWED_DOMAINS",
						"mcr.microsoft.com/playwright:latest",
					},
					Env: make(map[string]string),
				}

				// Set default allowed domains to localhost only (matches implementation)
				allowedDomains := []string{"localhost", "127.0.0.1"}

				// Check for custom Playwright configuration
				if toolConfig, ok := toolValue.(map[string]any); ok {
					// Handle allowed_domains configuration with bundle resolution
					if domainsConfig, exists := toolConfig["allowed_domains"]; exists {
						// For now, we'll use a simple conversion. In a full implementation,
						// we'd need to use the same domain bundle resolution as the compiler
						var customDomains []string
						switch domains := domainsConfig.(type) {
						case []string:
							customDomains = domains
						case []any:
							customDomains = make([]string, len(domains))
							for i, domain := range domains {
								if domainStr, ok := domain.(string); ok {
									customDomains[i] = domainStr
								}
							}
						case string:
							customDomains = []string{domains}
						}

						// Ensure localhost domains are always included
						allowedDomains = ensureLocalhostDomains(customDomains)
					}

					// Check for custom Docker image version
					if version, exists := toolConfig["docker_image_version"]; exists {
						if versionStr, ok := version.(string); ok {
							dockerImage := "mcr.microsoft.com/playwright:" + versionStr
							// Update the Docker image in args
							for i, arg := range config.Args {
								if strings.HasPrefix(arg, "mcr.microsoft.com/playwright:") {
									config.Args[i] = dockerImage
									break
								}
							}
						}
					}
				}

				config.Env["PLAYWRIGHT_ALLOWED_DOMAINS"] = strings.Join(allowedDomains, ",")
				if len(allowedDomains) == 0 {
					config.Env["PLAYWRIGHT_BLOCK_ALL_DOMAINS"] = "true"
				}

				configs = append(configs, config)
			}
		} else {
			// Handle custom MCP tools (those with explicit MCP configuration)
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

			// Apply server filter if specified
			if serverFilter != "" && !strings.Contains(strings.ToLower(toolName), strings.ToLower(serverFilter)) {
				continue
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
