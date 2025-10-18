package workflow

// Tools represents the parsed tools configuration from workflow frontmatter
type Tools struct {
	// Built-in tools
	GitHub           any `yaml:"github,omitempty"`
	Claude           any `yaml:"claude,omitempty"`
	Bash             any `yaml:"bash,omitempty"`
	WebFetch         any `yaml:"web-fetch,omitempty"`
	WebSearch        any `yaml:"web-search,omitempty"`
	Edit             any `yaml:"edit,omitempty"`
	Playwright       any `yaml:"playwright,omitempty"`
	AgenticWorkflows any `yaml:"agentic-workflows,omitempty"`
	CacheMemory      any `yaml:"cache-memory,omitempty"`
	SafetyPrompt     any `yaml:"safety-prompt,omitempty"`
	Timeout          any `yaml:"timeout,omitempty"`
	StartupTimeout   any `yaml:"startup-timeout,omitempty"`

	// Custom MCP tools (anything not in the above list)
	Custom map[string]any `yaml:",inline"`

	// Raw map for backwards compatibility
	raw map[string]any

	// Track which known tools are explicitly set (even if nil)
	hasGitHub           bool
	hasClaude           bool
	hasBash             bool
	hasWebFetch         bool
	hasWebSearch        bool
	hasEdit             bool
	hasPlaywright       bool
	hasAgenticWorkflows bool
	hasCacheMemory      bool
	hasSafetyPrompt     bool
	hasTimeout          bool
	hasStartupTimeout   bool
}

// GitHubToolConfig represents the configuration for the GitHub tool
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
	AllowedDomains any      `yaml:"allowed_domains,omitempty"`
	Args           []string `yaml:"args,omitempty"`
}

// ClaudeToolConfig represents the configuration for Claude tools
type ClaudeToolConfig struct {
	Allowed any `yaml:"allowed,omitempty"`
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

	// Extract known tools (check for existence in map, not just non-nil values)
	if _, exists := toolsMap["github"]; exists {
		tools.GitHub = toolsMap["github"]
		tools.hasGitHub = true
	}
	if _, exists := toolsMap["claude"]; exists {
		tools.Claude = toolsMap["claude"]
		tools.hasClaude = true
	}
	if _, exists := toolsMap["bash"]; exists {
		tools.Bash = toolsMap["bash"]
		tools.hasBash = true
	}
	if _, exists := toolsMap["web-fetch"]; exists {
		tools.WebFetch = toolsMap["web-fetch"]
		tools.hasWebFetch = true
	}
	if _, exists := toolsMap["web-search"]; exists {
		tools.WebSearch = toolsMap["web-search"]
		tools.hasWebSearch = true
	}
	if _, exists := toolsMap["edit"]; exists {
		tools.Edit = toolsMap["edit"]
		tools.hasEdit = true
	}
	if _, exists := toolsMap["playwright"]; exists {
		tools.Playwright = toolsMap["playwright"]
		tools.hasPlaywright = true
	}
	if _, exists := toolsMap["agentic-workflows"]; exists {
		tools.AgenticWorkflows = toolsMap["agentic-workflows"]
		tools.hasAgenticWorkflows = true
	}
	if _, exists := toolsMap["cache-memory"]; exists {
		tools.CacheMemory = toolsMap["cache-memory"]
		tools.hasCacheMemory = true
	}
	if _, exists := toolsMap["safety-prompt"]; exists {
		tools.SafetyPrompt = toolsMap["safety-prompt"]
		tools.hasSafetyPrompt = true
	}
	if _, exists := toolsMap["timeout"]; exists {
		tools.Timeout = toolsMap["timeout"]
		tools.hasTimeout = true
	}
	if _, exists := toolsMap["startup-timeout"]; exists {
		tools.StartupTimeout = toolsMap["startup-timeout"]
		tools.hasStartupTimeout = true
	}

	// Extract custom MCP tools (anything not in the known list)
	knownTools := map[string]bool{
		"github":            true,
		"claude":            true,
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

// ToMap converts the Tools struct back to a map[string]any for backwards compatibility
func (t *Tools) ToMap() map[string]any {
	if t == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)

	// Add known tools if they exist (use has flags to check)
	if t.hasGitHub {
		result["github"] = t.GitHub
	}
	if t.hasClaude {
		result["claude"] = t.Claude
	}
	if t.hasBash {
		result["bash"] = t.Bash
	}
	if t.hasWebFetch {
		result["web-fetch"] = t.WebFetch
	}
	if t.hasWebSearch {
		result["web-search"] = t.WebSearch
	}
	if t.hasEdit {
		result["edit"] = t.Edit
	}
	if t.hasPlaywright {
		result["playwright"] = t.Playwright
	}
	if t.hasAgenticWorkflows {
		result["agentic-workflows"] = t.AgenticWorkflows
	}
	if t.hasCacheMemory {
		result["cache-memory"] = t.CacheMemory
	}
	if t.hasSafetyPrompt {
		result["safety-prompt"] = t.SafetyPrompt
	}
	if t.hasTimeout {
		result["timeout"] = t.Timeout
	}
	if t.hasStartupTimeout {
		result["startup-timeout"] = t.StartupTimeout
	}

	// Add custom MCP tools
	for name, config := range t.Custom {
		result[name] = config
	}

	return result
}

// HasTool checks if a tool is present in the configuration
func (t *Tools) HasTool(name string) bool {
	if t == nil {
		return false
	}

	switch name {
	case "github":
		return t.hasGitHub
	case "claude":
		return t.hasClaude
	case "bash":
		return t.hasBash
	case "web-fetch":
		return t.hasWebFetch
	case "web-search":
		return t.hasWebSearch
	case "edit":
		return t.hasEdit
	case "playwright":
		return t.hasPlaywright
	case "agentic-workflows":
		return t.hasAgenticWorkflows
	case "cache-memory":
		return t.hasCacheMemory
	case "safety-prompt":
		return t.hasSafetyPrompt
	case "timeout":
		return t.hasTimeout
	case "startup-timeout":
		return t.hasStartupTimeout
	default:
		_, exists := t.Custom[name]
		return exists
	}
}

// GetTool returns the configuration for a specific tool
func (t *Tools) GetTool(name string) any {
	if t == nil {
		return nil
	}

	switch name {
	case "github":
		return t.GitHub
	case "claude":
		return t.Claude
	case "bash":
		return t.Bash
	case "web-fetch":
		return t.WebFetch
	case "web-search":
		return t.WebSearch
	case "edit":
		return t.Edit
	case "playwright":
		return t.Playwright
	case "agentic-workflows":
		return t.AgenticWorkflows
	case "cache-memory":
		return t.CacheMemory
	case "safety-prompt":
		return t.SafetyPrompt
	case "timeout":
		return t.Timeout
	case "startup-timeout":
		return t.StartupTimeout
	default:
		return t.Custom[name]
	}
}

// GetToolNames returns a list of all tool names configured
func (t *Tools) GetToolNames() []string {
	if t == nil {
		return []string{}
	}

	names := []string{}

	if t.hasGitHub {
		names = append(names, "github")
	}
	if t.hasClaude {
		names = append(names, "claude")
	}
	if t.hasBash {
		names = append(names, "bash")
	}
	if t.hasWebFetch {
		names = append(names, "web-fetch")
	}
	if t.hasWebSearch {
		names = append(names, "web-search")
	}
	if t.hasEdit {
		names = append(names, "edit")
	}
	if t.hasPlaywright {
		names = append(names, "playwright")
	}
	if t.hasAgenticWorkflows {
		names = append(names, "agentic-workflows")
	}
	if t.hasCacheMemory {
		names = append(names, "cache-memory")
	}
	if t.hasSafetyPrompt {
		names = append(names, "safety-prompt")
	}
	if t.hasTimeout {
		names = append(names, "timeout")
	}
	if t.hasStartupTimeout {
		names = append(names, "startup-timeout")
	}

	// Add custom tools
	for name := range t.Custom {
		names = append(names, name)
	}

	return names
}

// GetGitHubConfig returns the GitHub tool configuration as a typed struct
func (t *Tools) GetGitHubConfig() *GitHubToolConfig {
	if t == nil || t.GitHub == nil {
		return nil
	}

	// If it's a map, convert it
	if configMap, ok := t.GitHub.(map[string]any); ok {
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

	return nil
}

// GetPlaywrightConfig returns the Playwright tool configuration as a typed struct
func (t *Tools) GetPlaywrightConfig() *PlaywrightToolConfig {
	if t == nil || t.Playwright == nil {
		return nil
	}

	// If it's a map, convert it
	if configMap, ok := t.Playwright.(map[string]any); ok {
		config := &PlaywrightToolConfig{}

		if version, ok := configMap["version"].(string); ok {
			config.Version = version
		}

		if allowedDomains, ok := configMap["allowed_domains"]; ok {
			config.AllowedDomains = allowedDomains
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

	return nil
}

// GetClaudeConfig returns the Claude tool configuration as a typed struct
func (t *Tools) GetClaudeConfig() *ClaudeToolConfig {
	if t == nil || t.Claude == nil {
		return nil
	}

	// If it's a map, convert it
	if configMap, ok := t.Claude.(map[string]any); ok {
		config := &ClaudeToolConfig{}

		if allowed, ok := configMap["allowed"]; ok {
			config.Allowed = allowed
		}

		return config
	}

	return nil
}
