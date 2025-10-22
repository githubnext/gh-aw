package parser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EnsureLocalhostDomains ensures that localhost and 127.0.0.1 are always included
// in the allowed domains list for Playwright, even when custom domains are specified
// Includes port variations to allow all ports on localhost and 127.0.0.1
func EnsureLocalhostDomains(domains []string) []string {
	hasLocalhost := false
	hasLocalhostPorts := false
	hasLoopback := false
	hasLoopbackPorts := false

	for _, domain := range domains {
		switch domain {
		case "localhost":
			hasLocalhost = true
		case "localhost:*":
			hasLocalhostPorts = true
		case "127.0.0.1":
			hasLoopback = true
		case "127.0.0.1:*":
			hasLoopbackPorts = true
		}
	}

	// CWE-190: Allocation Size Overflow Prevention
	// Instead of pre-calculating capacity (len(domains)+4), which could overflow
	// if domains is extremely large, we let Go's append handle capacity growth
	// automatically. This is safe and efficient for domain arrays which are
	// typically small in practice.
	var result []string

	// Always add localhost domains first (with and without port specifications)
	if !hasLocalhost {
		result = append(result, "localhost")
	}
	if !hasLocalhostPorts {
		result = append(result, "localhost:*")
	}
	if !hasLoopback {
		result = append(result, "127.0.0.1")
	}
	if !hasLoopbackPorts {
		result = append(result, "127.0.0.1:*")
	}

	// Add the rest of the domains
	result = append(result, domains...)

	return result
}

// MCPServerConfig represents a parsed MCP server configuration
type MCPServerConfig struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`           // stdio, http, docker
	Registry       string            `json:"registry"`       // URI to installation location from registry
	Command        string            `json:"command"`        // for stdio
	Args           []string          `json:"args"`           // for stdio
	Container      string            `json:"container"`      // for docker
	Version        string            `json:"version"`        // optional version/tag for container
	EntrypointArgs []string          `json:"entrypointArgs"` // arguments to add after container image
	URL            string            `json:"url"`            // for http
	Headers        map[string]string `json:"headers"`        // for http
	Env            map[string]string `json:"env"`            // environment variables
	ProxyArgs      []string          `json:"proxy-args"`     // custom proxy arguments for container-based tools
	Allowed        []string          `json:"allowed"`        // allowed tools
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
					case "add-comment":
						config.Allowed = append(config.Allowed, "add-comment")
					case "create-pull-request":
						config.Allowed = append(config.Allowed, "create-pull-request")
					case "create-pull-request-review-comment":
						config.Allowed = append(config.Allowed, "create-pull-request-review-comment")
					case "create-code-scanning-alert":
						config.Allowed = append(config.Allowed, "create-code-scanning-alert")
					case "add-labels":
						config.Allowed = append(config.Allowed, "add-labels")
					case "update-issue":
						config.Allowed = append(config.Allowed, "update-issue")
					case "push-to-pull-request-branch":
						config.Allowed = append(config.Allowed, "push-to-pull-request-branch")
					case "missing-tool":
						config.Allowed = append(config.Allowed, "missing-tool")

					}
				}
			}

			configs = append(configs, config)
		}
	}

	// Check for top-level safe-jobs configuration
	if safeJobsSection, hasSafeJobs := frontmatter["safe-jobs"]; hasSafeJobs {
		// Apply server filter if specified
		if serverFilter == "" || strings.Contains("safe-outputs", strings.ToLower(serverFilter)) {
			// Find existing safe-outputs config or create new one
			var config *MCPServerConfig
			for i := range configs {
				if configs[i].Name == "safe-outputs" {
					config = &configs[i]
					break
				}
			}

			if config == nil {
				newConfig := MCPServerConfig{
					Name:    "safe-outputs",
					Type:    "stdio",
					Command: "node",
					Env:     make(map[string]string),
				}
				configs = append(configs, newConfig)
				config = &configs[len(configs)-1]
			}

			// Add each safe-job as a tool
			if safeJobsMap, ok := safeJobsSection.(map[string]any); ok {
				for jobName := range safeJobsMap {
					config.Allowed = append(config.Allowed, jobName)
				}
			}
		}
	}

	// Get mcp-servers section from frontmatter
	mcpServersSection, hasMCPServers := frontmatter["mcp-servers"]
	if !hasMCPServers {
		// Also check tools section for built-in MCP tools (github, playwright)
		toolsSection, hasTools := frontmatter["tools"]
		if hasTools {
			if tools, ok := toolsSection.(map[string]any); ok {
				for toolName, toolValue := range tools {
					// Only handle built-in MCP tools (github and playwright)
					if toolName == "github" || toolName == "playwright" {
						config, err := processBuiltinMCPTool(toolName, toolValue, serverFilter)
						if err != nil {
							return nil, err
						}
						if config != nil {
							configs = append(configs, *config)
						}
					}
				}
			}
		}
		return configs, nil // No mcp-servers configured, but we might have safe-outputs and built-in tools
	}

	mcpServers, ok := mcpServersSection.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mcp-servers section is not a valid map")
	}

	// Process built-in MCP tools from tools section
	toolsSection, hasTools := frontmatter["tools"]
	if hasTools {
		if tools, ok := toolsSection.(map[string]any); ok {
			for toolName, toolValue := range tools {
				// Only handle built-in MCP tools (github and playwright)
				if toolName == "github" || toolName == "playwright" {
					config, err := processBuiltinMCPTool(toolName, toolValue, serverFilter)
					if err != nil {
						return nil, err
					}
					if config != nil {
						configs = append(configs, *config)
					}
				}
			}
		}
	}

	// Process custom MCP servers from mcp-servers section
	for serverName, serverValue := range mcpServers {
		// Apply server filter if specified
		if serverFilter != "" && !strings.Contains(strings.ToLower(serverName), strings.ToLower(serverFilter)) {
			continue
		}

		// Handle custom MCP tools (those with explicit MCP configuration)
		toolConfig, ok := serverValue.(map[string]any)
		if !ok {
			continue
		}

		config, err := ParseMCPConfig(serverName, toolConfig, toolConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP config for %s: %w", serverName, err)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// processBuiltinMCPTool handles built-in MCP tools (github and playwright)
func processBuiltinMCPTool(toolName string, toolValue any, serverFilter string) (*MCPServerConfig, error) {
	// Apply server filter if specified
	if serverFilter != "" && !strings.Contains(strings.ToLower(toolName), strings.ToLower(serverFilter)) {
		return nil, nil
	}

	if toolName == "github" {
		// Check for custom GitHub configuration to determine mode (local vs remote)
		var useRemote bool
		var customGitHubToken string
		var readOnly bool

		if toolConfig, ok := toolValue.(map[string]any); ok {
			// Check if mode is specified (remote or local)
			if modeField, hasMode := toolConfig["mode"]; hasMode {
				if modeStr, ok := modeField.(string); ok && modeStr == "remote" {
					useRemote = true
				}
			}

			// Check for custom github-token
			if token, hasToken := toolConfig["github-token"]; hasToken {
				if tokenStr, ok := token.(string); ok {
					customGitHubToken = tokenStr
				}
			}

			// Check for read-only mode
			if readOnlyField, hasReadOnly := toolConfig["read-only"]; hasReadOnly {
				if readOnlyBool, ok := readOnlyField.(bool); ok {
					readOnly = readOnlyBool
				}
			}
		}

		var config MCPServerConfig

		if useRemote {
			// Handle GitHub MCP server in remote mode (hosted)
			config = MCPServerConfig{
				Name:    "github",
				Type:    "http",
				URL:     "https://api.githubcopilot.com/mcp/",
				Headers: make(map[string]string),
				Env:     make(map[string]string),
			}

			// Store custom token for later use in workflow generation
			if customGitHubToken != "" {
				config.Env["GITHUB_TOKEN"] = customGitHubToken
			}

			// Add X-MCP-Readonly header if read-only mode is enabled
			if readOnly {
				config.Headers["X-MCP-Readonly"] = "true"
			}
		} else {
			// Handle GitHub MCP server - use local/Docker by default
			config = MCPServerConfig{
				Name:    "github",
				Type:    "docker", // GitHub defaults to Docker (local containerized)
				Command: "docker",
				Args: []string{
					"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
					"ghcr.io/github/github-mcp-server:" + constants.DefaultGitHubMCPServerVersion,
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
		}

		// Check for custom GitHub configuration
		if toolConfig, ok := toolValue.(map[string]any); ok {
			// Check for read-only mode (only applicable in local/Docker mode)
			if !useRemote && readOnly {
				// When read-only is true, inline GITHUB_READ_ONLY=1 in docker args
				config.Args = append(config.Args[:5], append([]string{"-e", "GITHUB_READ_ONLY=1"}, config.Args[5:]...)...)
			}

			if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
				if allowedSlice, ok := allowed.([]any); ok {
					for _, item := range allowedSlice {
						if str, ok := item.(string); ok {
							config.Allowed = append(config.Allowed, str)
						}
					}
				}
			}

			// Check for custom Docker image version (only applicable in local/Docker mode)
			if !useRemote {
				if version, exists := toolConfig["version"]; exists {
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

				// Check for custom args (only applicable in local/Docker mode)
				if argsValue, exists := toolConfig["args"]; exists {
					// Handle []any format
					if argsSlice, ok := argsValue.([]any); ok {
						for _, arg := range argsSlice {
							if argStr, ok := arg.(string); ok {
								config.Args = append(config.Args, argStr)
							}
						}
					}
					// Handle []string format
					if argsSlice, ok := argsValue.([]string); ok {
						config.Args = append(config.Args, argsSlice...)
					}
				}
			}
		}

		return &config, nil
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

		// Set default allowed domains to localhost with all port variations (matches implementation)
		allowedDomains := []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*"}

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
				allowedDomains = EnsureLocalhostDomains(customDomains)
			}

			// Check for custom Docker image version
			if version, exists := toolConfig["version"]; exists {
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

			// Check for custom args
			if argsValue, exists := toolConfig["args"]; exists {
				// Handle []any format
				if argsSlice, ok := argsValue.([]any); ok {
					for _, arg := range argsSlice {
						if argStr, ok := arg.(string); ok {
							config.Args = append(config.Args, argStr)
						}
					}
				}
				// Handle []string format
				if argsSlice, ok := argsValue.([]string); ok {
					config.Args = append(config.Args, argsSlice...)
				}
			}
		}

		config.Env["PLAYWRIGHT_ALLOWED_DOMAINS"] = strings.Join(allowedDomains, ",")
		if len(allowedDomains) == 0 {
			config.Env["PLAYWRIGHT_BLOCK_ALL_DOMAINS"] = "true"
		}

		return &config, nil
	}

	return nil, nil
}

// ParseMCPConfig parses MCP configuration from various formats (map or JSON string)
func ParseMCPConfig(toolName string, mcpSection any, toolConfig map[string]any) (MCPServerConfig, error) {
	config := MCPServerConfig{
		Name:    toolName,
		Env:     make(map[string]string),
		Headers: make(map[string]string),
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

	// Extract type (explicit or inferred)
	if typeVal, hasType := mcpConfig["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			// Normalize "local" to "stdio"
			if typeStr == "local" {
				config.Type = "stdio"
			} else {
				config.Type = typeStr
			}
		} else {
			return config, fmt.Errorf("type must be a string")
		}
	} else {
		// Infer type from presence of fields
		if _, hasURL := mcpConfig["url"]; hasURL {
			config.Type = "http"
		} else if _, hasCommand := mcpConfig["command"]; hasCommand {
			config.Type = "stdio"
		} else if _, hasContainer := mcpConfig["container"]; hasContainer {
			config.Type = "stdio"
		} else {
			return config, fmt.Errorf("unable to determine MCP type for tool '%s': missing type, url, command, or container", toolName)
		}
	}

	// Extract registry field (available for both stdio and http)
	if registry, hasRegistry := mcpConfig["registry"]; hasRegistry {
		if registryStr, ok := registry.(string); ok {
			config.Registry = registryStr
		} else {
			return config, fmt.Errorf("registry must be a string")
		}
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
						// Sort environment variable keys to ensure deterministic arg order
						var envKeys []string
						for key := range envMap {
							envKeys = append(envKeys, key)
						}
						sort.Strings(envKeys)

						for _, key := range envKeys {
							if valueStr, ok := envMap[key].(string); ok {
								config.Args = append(config.Args, "-e", key)
								config.Env[key] = valueStr
							}
						}
					}
				}

				config.Args = append(config.Args, containerStr)

				// Add entrypoint args after the container image
				if entrypointArgs, hasEntrypointArgs := mcpConfig["entrypointArgs"]; hasEntrypointArgs {
					if entrypointArgsSlice, ok := entrypointArgs.([]any); ok {
						for _, arg := range entrypointArgsSlice {
							if argStr, ok := arg.(string); ok {
								config.Args = append(config.Args, argStr)
							}
						}
					}
				}
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

		// Extract network configuration for stdio (container-based tools)
		if network, hasNetwork := mcpConfig["network"]; hasNetwork {
			if networkMap, ok := network.(map[string]any); ok {
				// Extract proxy arguments from network config
				if proxyArgs, hasProxyArgs := networkMap["proxy-args"]; hasProxyArgs {
					if proxyArgsSlice, ok := proxyArgs.([]any); ok {
						for _, arg := range proxyArgsSlice {
							if argStr, ok := arg.(string); ok {
								config.ProxyArgs = append(config.ProxyArgs, argStr)
							}
						}
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
