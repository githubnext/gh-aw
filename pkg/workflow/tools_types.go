package workflow

// Tools represents the parsed tools configuration from workflow frontmatter
type Tools struct {
	// Built-in tools - using pointers to distinguish between "not set" and "set to nil/empty"
	GitHub           *GitHubToolConfig           `yaml:"github,omitempty"`
	Bash             *BashToolConfig             `yaml:"bash,omitempty"`
	WebFetch         *WebFetchToolConfig         `yaml:"web-fetch,omitempty"`
	WebSearch        *WebSearchToolConfig        `yaml:"web-search,omitempty"`
	Edit             *EditToolConfig             `yaml:"edit,omitempty"`
	Playwright       *PlaywrightToolConfig       `yaml:"playwright,omitempty"`
	AgenticWorkflows *AgenticWorkflowsToolConfig `yaml:"agentic-workflows,omitempty"`
	CacheMemory      *CacheMemoryToolConfig      `yaml:"cache-memory,omitempty"`
	SafetyPrompt     *bool                       `yaml:"safety-prompt,omitempty"`
	Timeout          *int                        `yaml:"timeout,omitempty"`
	StartupTimeout   *int                        `yaml:"startup-timeout,omitempty"`

	// Custom MCP tools (anything not in the above list)
	Custom map[string]any `yaml:",inline"`

	// Raw map for backwards compatibility
	raw map[string]any
}

// GitHubToolConfig represents the configuration for the GitHub tool
// Can be nil (enabled with defaults), string, or an object with specific settings
type GitHubToolConfig struct {
	Allowed     []string `yaml:"allowed,omitempty"`
	Mode        string   `yaml:"mode,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Args        []string `yaml:"args,omitempty"`
	ReadOnly    bool     `yaml:"read-only,omitempty"`
	GitHubToken string   `yaml:"github-token,omitempty"`
	Toolset     []string `yaml:"toolset,omitempty"`
}

// PlaywrightToolConfig represents the configuration for the Playwright tool
type PlaywrightToolConfig struct {
	Version        string   `yaml:"version,omitempty"`
	AllowedDomains []string `yaml:"allowed_domains,omitempty"`
	Args           []string `yaml:"args,omitempty"`
}

// BashToolConfig represents the configuration for the Bash tool
// Can be nil (all commands allowed) or an array of allowed commands
type BashToolConfig struct {
	AllowedCommands []string `yaml:"-"` // List of allowed bash commands
}

// WebFetchToolConfig represents the configuration for the web-fetch tool
type WebFetchToolConfig struct {
	// Currently an empty object or nil
}

// WebSearchToolConfig represents the configuration for the web-search tool
type WebSearchToolConfig struct {
	// Currently an empty object or nil
}

// EditToolConfig represents the configuration for the edit tool
type EditToolConfig struct {
	// Currently an empty object or nil
}

// AgenticWorkflowsToolConfig represents the configuration for the agentic-workflows tool
type AgenticWorkflowsToolConfig struct {
	// Can be boolean or nil
	Enabled bool `yaml:"-"`
}

// CacheMemoryToolConfig represents the configuration for cache-memory
// This is handled separately by the existing CacheMemoryConfig in cache.go
type CacheMemoryToolConfig struct {
	// Can be boolean, object, or array - handled by cache.go
	Raw any `yaml:"-"`
}

// NewTools creates a new Tools instance from a map
func NewTools(toolsMap map[string]any) *Tools {
	if toolsMap == nil {
		return &Tools{
			Custom: make(map[string]any),
			raw:    make(map[string]any),
		}
	}

	tools := &Tools{
		Custom: make(map[string]any),
		raw:    make(map[string]any),
	}

	// Copy raw map
	for k, v := range toolsMap {
		tools.raw[k] = v
	}

	// Extract and parse known tools
	if val, exists := toolsMap["github"]; exists {
		tools.GitHub = parseGitHubTool(val)
	}
	if val, exists := toolsMap["bash"]; exists {
		tools.Bash = parseBashTool(val)
	}
	if val, exists := toolsMap["web-fetch"]; exists {
		tools.WebFetch = parseWebFetchTool(val)
	}
	if val, exists := toolsMap["web-search"]; exists {
		tools.WebSearch = parseWebSearchTool(val)
	}
	if val, exists := toolsMap["edit"]; exists {
		tools.Edit = parseEditTool(val)
	}
	if val, exists := toolsMap["playwright"]; exists {
		tools.Playwright = parsePlaywrightTool(val)
	}
	if val, exists := toolsMap["agentic-workflows"]; exists {
		tools.AgenticWorkflows = parseAgenticWorkflowsTool(val)
	}
	if val, exists := toolsMap["cache-memory"]; exists {
		tools.CacheMemory = parseCacheMemoryTool(val)
	}
	if val, exists := toolsMap["safety-prompt"]; exists {
		tools.SafetyPrompt = parseSafetyPromptTool(val)
	}
	if val, exists := toolsMap["timeout"]; exists {
		tools.Timeout = parseTimeoutTool(val)
	}
	if val, exists := toolsMap["startup-timeout"]; exists {
		tools.StartupTimeout = parseStartupTimeoutTool(val)
	}

	// Extract custom MCP tools (anything not in the known list)
	knownTools := map[string]bool{
		"github":            true,
		"bash":              true,
		"web-fetch":         true,
		"web-search":        true,
		"edit":              true,
		"playwright":        true,
		"agentic-workflows": true,
		"cache-memory":      true,
		"safety-prompt":     true,
		"timeout":           true,
		"startup-timeout":   true,
	}

	for name, config := range toolsMap {
		if !knownTools[name] {
			tools.Custom[name] = config
		}
	}

	return tools
}

// parseGitHubTool converts raw github tool configuration to GitHubToolConfig
func parseGitHubTool(val any) *GitHubToolConfig {
	if val == nil {
		return &GitHubToolConfig{}
	}

	// Handle string type (simple enable)
	if _, ok := val.(string); ok {
		return &GitHubToolConfig{}
	}

	// Handle map type (detailed configuration)
	if configMap, ok := val.(map[string]any); ok {
		config := &GitHubToolConfig{}

		if allowed, ok := configMap["allowed"].([]any); ok {
			config.Allowed = make([]string, 0, len(allowed))
			for _, item := range allowed {
				if str, ok := item.(string); ok {
					config.Allowed = append(config.Allowed, str)
				}
			}
		}

		if mode, ok := configMap["mode"].(string); ok {
			config.Mode = mode
		}

		if version, ok := configMap["version"].(string); ok {
			config.Version = version
		}

		if args, ok := configMap["args"].([]any); ok {
			config.Args = make([]string, 0, len(args))
			for _, item := range args {
				if str, ok := item.(string); ok {
					config.Args = append(config.Args, str)
				}
			}
		}

		if readOnly, ok := configMap["read-only"].(bool); ok {
			config.ReadOnly = readOnly
		}

		if token, ok := configMap["github-token"].(string); ok {
			config.GitHubToken = token
		}

		if toolset, ok := configMap["toolset"].([]any); ok {
			config.Toolset = make([]string, 0, len(toolset))
			for _, item := range toolset {
				if str, ok := item.(string); ok {
					config.Toolset = append(config.Toolset, str)
				}
			}
		}

		return config
	}

	return &GitHubToolConfig{}
}

// parseBashTool converts raw bash tool configuration to BashToolConfig
func parseBashTool(val any) *BashToolConfig {
	if val == nil {
		// nil means all commands allowed
		return &BashToolConfig{}
	}

	// Handle array of allowed commands
	if cmdArray, ok := val.([]any); ok {
		config := &BashToolConfig{
			AllowedCommands: make([]string, 0, len(cmdArray)),
		}
		for _, item := range cmdArray {
			if str, ok := item.(string); ok {
				config.AllowedCommands = append(config.AllowedCommands, str)
			}
		}
		return config
	}

	return &BashToolConfig{}
}

// parsePlaywrightTool converts raw playwright tool configuration to PlaywrightToolConfig
func parsePlaywrightTool(val any) *PlaywrightToolConfig {
	if val == nil {
		return &PlaywrightToolConfig{}
	}

	if configMap, ok := val.(map[string]any); ok {
		config := &PlaywrightToolConfig{}

		if version, ok := configMap["version"].(string); ok {
			config.Version = version
		}

		// Handle allowed_domains - can be string or array
		if allowedDomains, ok := configMap["allowed_domains"]; ok {
			if str, ok := allowedDomains.(string); ok {
				config.AllowedDomains = []string{str}
			} else if arr, ok := allowedDomains.([]any); ok {
				config.AllowedDomains = make([]string, 0, len(arr))
				for _, item := range arr {
					if str, ok := item.(string); ok {
						config.AllowedDomains = append(config.AllowedDomains, str)
					}
				}
			}
		}

		if args, ok := configMap["args"].([]any); ok {
			config.Args = make([]string, 0, len(args))
			for _, item := range args {
				if str, ok := item.(string); ok {
					config.Args = append(config.Args, str)
				}
			}
		}

		return config
	}

	return &PlaywrightToolConfig{}
}

// parseWebFetchTool converts raw web-fetch tool configuration
func parseWebFetchTool(val any) *WebFetchToolConfig {
	// web-fetch is either nil or an empty object
	return &WebFetchToolConfig{}
}

// parseWebSearchTool converts raw web-search tool configuration
func parseWebSearchTool(val any) *WebSearchToolConfig {
	// web-search is either nil or an empty object
	return &WebSearchToolConfig{}
}

// parseEditTool converts raw edit tool configuration
func parseEditTool(val any) *EditToolConfig {
	// edit is either nil or an empty object
	return &EditToolConfig{}
}

// parseAgenticWorkflowsTool converts raw agentic-workflows tool configuration
func parseAgenticWorkflowsTool(val any) *AgenticWorkflowsToolConfig {
	config := &AgenticWorkflowsToolConfig{}

	if boolVal, ok := val.(bool); ok {
		config.Enabled = boolVal
	} else if val == nil {
		config.Enabled = true // nil means enabled
	}

	return config
}

// parseCacheMemoryTool converts raw cache-memory tool configuration
func parseCacheMemoryTool(val any) *CacheMemoryToolConfig {
	// cache-memory can be boolean, object, or array - store raw value
	return &CacheMemoryToolConfig{Raw: val}
}

// parseSafetyPromptTool converts raw safety-prompt tool configuration
func parseSafetyPromptTool(val any) *bool {
	if boolVal, ok := val.(bool); ok {
		return &boolVal
	}
	// Default to true if not specified or invalid type
	defaultVal := true
	return &defaultVal
}

// parseTimeoutTool converts raw timeout tool configuration
func parseTimeoutTool(val any) *int {
	if intVal, ok := val.(int); ok {
		return &intVal
	}
	if floatVal, ok := val.(float64); ok {
		intVal := int(floatVal)
		return &intVal
	}
	return nil
}

// parseStartupTimeoutTool converts raw startup-timeout tool configuration
func parseStartupTimeoutTool(val any) *int {
	if intVal, ok := val.(int); ok {
		return &intVal
	}
	if floatVal, ok := val.(float64); ok {
		intVal := int(floatVal)
		return &intVal
	}
	return nil
}

// HasTool checks if a tool is present in the configuration
func (t *Tools) HasTool(name string) bool {
	if t == nil {
		return false
	}

	switch name {
	case "github":
		return t.GitHub != nil
	case "bash":
		return t.Bash != nil
	case "web-fetch":
		return t.WebFetch != nil
	case "web-search":
		return t.WebSearch != nil
	case "edit":
		return t.Edit != nil
	case "playwright":
		return t.Playwright != nil
	case "agentic-workflows":
		return t.AgenticWorkflows != nil
	case "cache-memory":
		return t.CacheMemory != nil
	case "safety-prompt":
		return t.SafetyPrompt != nil
	case "timeout":
		return t.Timeout != nil
	case "startup-timeout":
		return t.StartupTimeout != nil
	default:
		_, exists := t.Custom[name]
		return exists
	}
}

// GetToolNames returns a list of all tool names configured
func (t *Tools) GetToolNames() []string {
	if t == nil {
		return []string{}
	}

	names := []string{}

	if t.GitHub != nil {
		names = append(names, "github")
	}
	if t.Bash != nil {
		names = append(names, "bash")
	}
	if t.WebFetch != nil {
		names = append(names, "web-fetch")
	}
	if t.WebSearch != nil {
		names = append(names, "web-search")
	}
	if t.Edit != nil {
		names = append(names, "edit")
	}
	if t.Playwright != nil {
		names = append(names, "playwright")
	}
	if t.AgenticWorkflows != nil {
		names = append(names, "agentic-workflows")
	}
	if t.CacheMemory != nil {
		names = append(names, "cache-memory")
	}
	if t.SafetyPrompt != nil {
		names = append(names, "safety-prompt")
	}
	if t.Timeout != nil {
		names = append(names, "timeout")
	}
	if t.StartupTimeout != nil {
		names = append(names, "startup-timeout")
	}

	// Add custom tools
	for name := range t.Custom {
		names = append(names, name)
	}

	return names
}
