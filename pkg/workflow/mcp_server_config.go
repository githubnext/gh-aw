package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// MCPServerConfiguration represents a computed MCP server configuration
type MCPServerConfiguration struct {
	// Basic server information
	Name string `json:"name"`
	Type string `json:"type"` // stdio, http

	// Stdio configuration
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Container string            `json:"container,omitempty"`
	Env       map[string]string `json:"env,omitempty"`

	// HTTP configuration
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	// Tool access control
	Allowed []string `json:"allowed,omitempty"`

	// Internal flags
	UsesProxy bool `json:"uses_proxy,omitempty"`

	// Engine-specific configuration
	EngineConfig map[string]any `json:"engine_config,omitempty"`
}

// MCPServerConfigProvider provides computed MCP server configurations
type MCPServerConfigProvider struct {
	configurations []MCPServerConfiguration
	mcpTools       []string
}

// NewMCPServerConfigProvider creates a new MCP server configuration provider
func NewMCPServerConfigProvider() *MCPServerConfigProvider {
	return &MCPServerConfigProvider{
		configurations: make([]MCPServerConfiguration, 0),
		mcpTools:       make([]string, 0),
	}
}

// NewMCPServerConfigurationsFromFrontmatter creates and computes MCP server configurations from workflow frontmatter
func NewMCPServerConfigurationsFromFrontmatter(frontmatter map[string]any, networkPermissions *NetworkPermissions, workflowData *WorkflowData) ([]MCPServerConfiguration, error) {
	provider := NewMCPServerConfigProvider()
	err := provider.ComputeMCPServerConfigurations(frontmatter, networkPermissions, workflowData)
	if err != nil {
		return nil, err
	}
	return provider.GetConfigurations(), nil
}

// ComputeMCPServerConfigurations extracts and computes MCP server configurations from workflow frontmatter
func (p *MCPServerConfigProvider) ComputeMCPServerConfigurations(frontmatter map[string]any, networkPermissions *NetworkPermissions, workflowData *WorkflowData) error {
	// Clear previous configurations
	p.configurations = make([]MCPServerConfiguration, 0)
	p.mcpTools = make([]string, 0)

	// Get tools section from frontmatter
	toolsSection, hasTools := frontmatter["tools"]
	if !hasTools {
		return nil // No tools configured
	}

	tools, ok := toolsSection.(map[string]any)
	if !ok {
		return fmt.Errorf("tools section is not a valid map")
	}

	// Validate MCP configurations first
	if err := ValidateMCPConfigs(tools); err != nil {
		return fmt.Errorf("MCP configuration validation failed: %w", err)
	}

	// Process each tool and extract MCP configurations
	for toolName, toolValue := range tools {
		config, err := p.computeToolMCPConfig(toolName, toolValue, tools, networkPermissions, workflowData)
		if err != nil {
			return fmt.Errorf("failed to compute MCP config for tool '%s': %w", toolName, err)
		}
		if config != nil {
			p.configurations = append(p.configurations, *config)
			p.mcpTools = append(p.mcpTools, toolName)
		}
	}

	// Sort configurations by name for consistent ordering
	sort.Slice(p.configurations, func(i, j int) bool {
		return p.configurations[i].Name < p.configurations[j].Name
	})

	// Sort mcpTools to match
	sort.Strings(p.mcpTools)

	return nil
}

// computeToolMCPConfig computes MCP configuration for a single tool
func (p *MCPServerConfigProvider) computeToolMCPConfig(toolName string, toolValue any, allTools map[string]any, networkPermissions *NetworkPermissions, workflowData *WorkflowData) (*MCPServerConfiguration, error) {
	switch toolName {
	case "github":
		return p.computeGitHubMCPConfig(toolValue, workflowData)
	case "playwright":
		return p.computePlaywrightMCPConfig(toolValue, networkPermissions)
	default:
		// Handle custom MCP tools (those with explicit MCP configuration)
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			return nil, nil // Not a valid tool configuration
		}

		// Check if it has MCP configuration
		mcpSection, hasMcp := toolConfig["mcp"]
		if !hasMcp {
			return nil, nil // Not an MCP tool
		}

		return p.computeCustomMCPConfig(toolName, mcpSection, toolConfig)
	}
}

// computeGitHubMCPConfig computes MCP configuration for the GitHub tool
func (p *MCPServerConfigProvider) computeGitHubMCPConfig(toolValue any, workflowData *WorkflowData) (*MCPServerConfiguration, error) {
	config := &MCPServerConfiguration{
		Name:    "github",
		Type:    "stdio",
		Command: "docker",
		Args: []string{
			"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
			"ghcr.io/github/github-mcp-server:sha-09deac4",
		},
		Env: map[string]string{
			"GITHUB_PERSONAL_ACCESS_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
		EngineConfig: make(map[string]any),
	}

	// Compute user_agent field for Codex engine
	userAgent := "github-agentic-workflow"
	if workflowData != nil {
		// Check if user_agent is configured in engine config first
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.UserAgent != "" {
			userAgent = workflowData.EngineConfig.UserAgent
		} else if workflowData.Name != "" {
			// Fall back to converting workflow name to identifier
			userAgent = convertToIdentifier(workflowData.Name)
		}
	}

	// Store user_agent in EngineConfig for engine-specific rendering
	config.EngineConfig["user_agent"] = userAgent

	// Extract configuration from tool settings
	if toolConfig, ok := toolValue.(map[string]any); ok {
		// Handle allowed tools
		if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
			if allowedSlice, ok := allowed.([]any); ok {
				for _, item := range allowedSlice {
					if str, ok := item.(string); ok {
						config.Allowed = append(config.Allowed, str)
					}
				}
			}
		}

		// Handle custom Docker image version
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

		// Store original config for engine-specific rendering
		config.EngineConfig["github"] = toolConfig
	}

	return config, nil
}

// computePlaywrightMCPConfig computes MCP configuration for the Playwright tool
func (p *MCPServerConfigProvider) computePlaywrightMCPConfig(toolValue any, networkPermissions *NetworkPermissions) (*MCPServerConfiguration, error) {
	config := &MCPServerConfiguration{
		Name:    "playwright",
		Type:    "stdio",
		Command: "docker",
		Args: []string{
			"run", "-i", "--rm", "--shm-size=2gb", "--cap-add=SYS_ADMIN",
			"-e", "PLAYWRIGHT_ALLOWED_DOMAINS",
			"mcr.microsoft.com/playwright:latest",
		},
		Env:          map[string]string{},
		EngineConfig: make(map[string]any),
	}

	// Set default allowed domains to localhost only
	allowedDomains := []string{"localhost", "127.0.0.1"}

	// Extract configuration from tool settings
	if toolConfig, ok := toolValue.(map[string]any); ok {
		// Handle allowed_domains configuration
		if domainsConfig, exists := toolConfig["allowed_domains"]; exists {
			switch domains := domainsConfig.(type) {
			case []string:
				allowedDomains = domains
			case []any:
				allowedDomains = make([]string, len(domains))
				for i, domain := range domains {
					if domainStr, ok := domain.(string); ok {
						allowedDomains[i] = domainStr
					}
				}
			case string:
				allowedDomains = []string{domains}
			}
		}

		// Handle custom Docker image version
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

		// Store original config for engine-specific rendering
		config.EngineConfig["playwright"] = toolConfig
	}

	// Set environment variables for domain control
	config.Env["PLAYWRIGHT_ALLOWED_DOMAINS"] = strings.Join(allowedDomains, ",")
	if len(allowedDomains) == 0 {
		config.Env["PLAYWRIGHT_BLOCK_ALL_DOMAINS"] = "true"
	}

	return config, nil
}

// computeCustomMCPConfig computes MCP configuration for custom MCP tools
func (p *MCPServerConfigProvider) computeCustomMCPConfig(toolName string, mcpSection any, toolConfig map[string]any) (*MCPServerConfiguration, error) {
	config := &MCPServerConfiguration{
		Name:         toolName,
		Env:          make(map[string]string),
		EngineConfig: make(map[string]any),
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

	// Parse the MCP section using existing logic from mcp-config.go
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	// Extract type (required)
	if typeVal, hasType := mcpConfig["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			config.Type = typeStr
		} else {
			return nil, fmt.Errorf("type must be a string")
		}
	} else {
		return nil, fmt.Errorf("missing required 'type' field")
	}

	// Extract configuration based on type
	switch config.Type {
	case "stdio":
		if err := p.extractStdioConfig(config, mcpConfig); err != nil {
			return nil, err
		}
	case "http":
		if err := p.extractHTTPConfig(config, mcpConfig); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported MCP type: %s", config.Type)
	}

	// Check for proxy usage
	if _, hasProxyFlag := mcpConfig["__uses_proxy"]; hasProxyFlag {
		config.UsesProxy = true
	}

	// Store original config for engine-specific rendering
	config.EngineConfig[toolName] = toolConfig

	return config, nil
}

// extractStdioConfig extracts stdio-specific configuration
func (p *MCPServerConfigProvider) extractStdioConfig(config *MCPServerConfiguration, mcpConfig map[string]any) error {
	// Handle container field (Docker mode)
	if container, hasContainer := mcpConfig["container"]; hasContainer {
		if containerStr, ok := container.(string); ok {
			config.Container = containerStr
			config.Command = "docker"
			config.Args = []string{"run", "--rm", "-i"}

			// Add environment variables for Docker
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
		} else {
			return fmt.Errorf("container must be a string")
		}
	} else {
		// Handle command and args
		if command, hasCommand := mcpConfig["command"]; hasCommand {
			if commandStr, ok := command.(string); ok {
				config.Command = commandStr
			} else {
				return fmt.Errorf("command must be a string")
			}
		} else {
			return fmt.Errorf("stdio type requires 'command' or 'container' field")
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

	// Extract environment variables
	if env, hasEnv := mcpConfig["env"]; hasEnv {
		if envMap, ok := env.(map[string]any); ok {
			for key, value := range envMap {
				if valueStr, ok := value.(string); ok {
					config.Env[key] = valueStr
				}
			}
		}
	}

	return nil
}

// extractHTTPConfig extracts HTTP-specific configuration
func (p *MCPServerConfigProvider) extractHTTPConfig(config *MCPServerConfiguration, mcpConfig map[string]any) error {
	if url, hasURL := mcpConfig["url"]; hasURL {
		if urlStr, ok := url.(string); ok {
			config.URL = urlStr
		} else {
			return fmt.Errorf("url must be a string")
		}
	} else {
		return fmt.Errorf("http type requires 'url' field")
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

	return nil
}

// GetConfigurations returns the computed MCP server configurations
func (p *MCPServerConfigProvider) GetConfigurations() []MCPServerConfiguration {
	return p.configurations
}

// GetMCPTools returns the list of MCP tool names
func (p *MCPServerConfigProvider) GetMCPTools() []string {
	return p.mcpTools
}

// GetConfigurationByName returns the configuration for a specific tool name
func (p *MCPServerConfigProvider) GetConfigurationByName(name string) (*MCPServerConfiguration, bool) {
	for _, config := range p.configurations {
		if config.Name == name {
			return &config, true
		}
	}
	return nil, false
}

// HasMCPTools returns true if any MCP tools are configured
func (p *MCPServerConfigProvider) HasMCPTools() bool {
	return len(p.mcpTools) > 0
}

// ToParserMCPServerConfigs converts MCPServerConfiguration to parser.MCPServerConfig for compatibility
func (p *MCPServerConfigProvider) ToParserMCPServerConfigs() []parser.MCPServerConfig {
	var result []parser.MCPServerConfig

	for _, config := range p.configurations {
		parserConfig := parser.MCPServerConfig{
			Name:      config.Name,
			Type:      config.Type,
			Command:   config.Command,
			Args:      config.Args,
			Container: config.Container,
			URL:       config.URL,
			Headers:   config.Headers,
			Env:       config.Env,
			Allowed:   config.Allowed,
		}
		result = append(result, parserConfig)
	}

	return result
}

// RenderConfigForEngine renders MCP configuration for a specific engine format
func (config *MCPServerConfiguration) RenderConfigForEngine(engineType string, renderer MCPConfigRenderer) (string, error) {
	var result strings.Builder

	switch engineType {
	case "claude":
		return config.renderForClaude()
	case "codex":
		return config.renderForCodex()
	default:
		// Use the shared rendering logic for other engines
		err := renderSharedMCPConfig(&result, config.Name, config.toMap(), renderer)
		return result.String(), err
	}
}

// renderForClaude renders configuration in Claude's JSON format
func (config *MCPServerConfiguration) renderForClaude() (string, error) {
	var result strings.Builder

	switch config.Name {
	case "github":
		// Use the existing GitHub Claude rendering logic
		fmt.Fprintf(&result, "              \"github\": {\n")
		fmt.Fprintf(&result, "                \"command\": \"docker\",\n")
		fmt.Fprintf(&result, "                \"args\": [\n")
		for i, arg := range config.Args {
			comma := ","
			if i == len(config.Args)-1 {
				comma = ""
			}
			fmt.Fprintf(&result, "                  \"%s\"%s\n", arg, comma)
		}
		fmt.Fprintf(&result, "                ],\n")
		fmt.Fprintf(&result, "                \"env\": {\n")
		fmt.Fprintf(&result, "                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"%s\"\n", config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"])
		fmt.Fprintf(&result, "                }\n")
		fmt.Fprintf(&result, "              }")

	case "playwright":
		// Use the existing Playwright Claude rendering logic
		fmt.Fprintf(&result, "              \"playwright\": {\n")
		fmt.Fprintf(&result, "                \"command\": \"docker\",\n")
		fmt.Fprintf(&result, "                \"args\": [\n")
		for i, arg := range config.Args {
			comma := ","
			if i == len(config.Args)-1 {
				comma = ""
			}
			fmt.Fprintf(&result, "                  \"%s\"%s\n", arg, comma)
		}
		fmt.Fprintf(&result, "                ],\n")
		fmt.Fprintf(&result, "                \"env\": {\n")
		envKeys := make([]string, 0, len(config.Env))
		for key := range config.Env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)
		for i, key := range envKeys {
			comma := ","
			if i == len(envKeys)-1 {
				comma = ""
			}
			fmt.Fprintf(&result, "                  \"%s\": \"%s\"%s\n", key, config.Env[key], comma)
		}
		fmt.Fprintf(&result, "                }\n")
		fmt.Fprintf(&result, "              }")

	default:
		// Use shared rendering for custom tools
		fmt.Fprintf(&result, "              \"%s\": {\n", config.Name)
		renderer := MCPConfigRenderer{
			IndentLevel: "                ",
			Format:      "json",
		}
		var configBuilder strings.Builder

		configMap := config.toMap()

		err := renderSharedMCPConfig(&configBuilder, config.Name, configMap, renderer)
		if err != nil {
			return "", err
		}
		result.WriteString(configBuilder.String())
		fmt.Fprintf(&result, "              }")
	}

	return result.String(), nil
}

// renderForCodex renders configuration in Codex's TOML format
func (config *MCPServerConfiguration) renderForCodex() (string, error) {
	var result strings.Builder

	fmt.Fprintf(&result, "          [mcp_servers.%s]\n", config.Name)

	// Handle GitHub tool specially to include user_agent
	if config.Name == "github" {
		// Add user_agent field first if available
		if userAgent, exists := config.EngineConfig["user_agent"]; exists {
			if userAgentStr, ok := userAgent.(string); ok {
				fmt.Fprintf(&result, "          user_agent = \"%s\"\n", userAgentStr)
			}
		}
	}

	renderer := MCPConfigRenderer{
		IndentLevel: "          ",
		Format:      "toml",
	}

	err := renderSharedMCPConfig(&result, config.Name, config.toMap(), renderer)
	return result.String(), err
}

// toMap converts MCPServerConfiguration to a map for rendering
func (config *MCPServerConfiguration) toMap() map[string]any {
	result := make(map[string]any)

	// Wrap the configuration in an "mcp" section for compatibility with getMCPConfig
	mcpSection := make(map[string]any)
	mcpSection["type"] = config.Type

	switch config.Type {
	case "stdio":
		if config.Container != "" {
			mcpSection["container"] = config.Container
		} else {
			mcpSection["command"] = config.Command
		}
		if len(config.Args) > 0 {
			args := make([]any, len(config.Args))
			for i, arg := range config.Args {
				args[i] = arg
			}
			mcpSection["args"] = args
		}
		if len(config.Env) > 0 {
			env := make(map[string]any)
			for k, v := range config.Env {
				env[k] = v
			}
			mcpSection["env"] = env
		}
	case "http":
		mcpSection["url"] = config.URL
		if len(config.Headers) > 0 {
			headers := make(map[string]any)
			for k, v := range config.Headers {
				headers[k] = v
			}
			mcpSection["headers"] = headers
		}
	}

	// Wrap in mcp section for compatibility with getMCPConfig
	result["mcp"] = mcpSection

	return result
}
