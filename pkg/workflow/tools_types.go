package workflow

// Tools represents the parsed tools configuration from workflow frontmatter
type Tools struct {
	// Built-in tools - using pointers to distinguish between "not set" and "set to nil/empty"
	GitHub           *GitHubToolConfig           `yaml:"github,omitempty"`
	Claude           *ClaudeToolConfig           `yaml:"claude,omitempty"`
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

// ClaudeToolConfig represents the configuration for Claude tools
// The Allowed field can be an array of strings or an object mapping tool names to arrays
type ClaudeToolConfig struct {
	// Allowed can be:
	// - []string: list of allowed tools
	// - map[string][]string: tool categories with specific allowed functions
	AllowedTools []string            `yaml:"-"` // When Allowed is an array
	AllowedMap   map[string][]string `yaml:"-"` // When Allowed is an object
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
		tools.hasGitHub = true
	}
	if val, exists := toolsMap["claude"]; exists {
		tools.Claude = parseClaudeTool(val)
		tools.hasClaude = true
	}
	if val, exists := toolsMap["bash"]; exists {
		tools.Bash = parseBashTool(val)
		tools.hasBash = true
	}
	if val, exists := toolsMap["web-fetch"]; exists {
		tools.WebFetch = parseWebFetchTool(val)
		tools.hasWebFetch = true
	}
	if val, exists := toolsMap["web-search"]; exists {
		tools.WebSearch = parseWebSearchTool(val)
		tools.hasWebSearch = true
	}
	if val, exists := toolsMap["edit"]; exists {
		tools.Edit = parseEditTool(val)
		tools.hasEdit = true
	}
	if val, exists := toolsMap["playwright"]; exists {
		tools.Playwright = parsePlaywrightTool(val)
		tools.hasPlaywright = true
	}
	if val, exists := toolsMap["agentic-workflows"]; exists {
		tools.AgenticWorkflows = parseAgenticWorkflowsTool(val)
		tools.hasAgenticWorkflows = true
	}
	if val, exists := toolsMap["cache-memory"]; exists {
		tools.CacheMemory = parseCacheMemoryTool(val)
		tools.hasCacheMemory = true
	}
	if val, exists := toolsMap["safety-prompt"]; exists {
		tools.SafetyPrompt = parseSafetyPromptTool(val)
		tools.hasSafetyPrompt = true
	}
	if val, exists := toolsMap["timeout"]; exists {
		tools.Timeout = parseTimeoutTool(val)
		tools.hasTimeout = true
	}
	if val, exists := toolsMap["startup-timeout"]; exists {
		tools.StartupTimeout = parseStartupTimeoutTool(val)
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

// parseClaudeTool converts raw claude tool configuration to ClaudeToolConfig
func parseClaudeTool(val any) *ClaudeToolConfig {
	if val == nil {
		return &ClaudeToolConfig{}
	}

	// Handle string type
	if _, ok := val.(string); ok {
		return &ClaudeToolConfig{}
	}

	// Handle map type
	if configMap, ok := val.(map[string]any); ok {
		config := &ClaudeToolConfig{}

		if allowed, ok := configMap["allowed"]; ok {
			// Check if it's an array
			if allowedArray, ok := allowed.([]any); ok {
				config.AllowedTools = make([]string, 0, len(allowedArray))
				for _, item := range allowedArray {
					if str, ok := item.(string); ok {
						config.AllowedTools = append(config.AllowedTools, str)
					}
				}
			} else if allowedMap, ok := allowed.(map[string]any); ok {
				// It's an object mapping tool names to arrays
				config.AllowedMap = make(map[string][]string)
				for key, val := range allowedMap {
					if valArray, ok := val.([]any); ok {
						strArray := make([]string, 0, len(valArray))
						for _, item := range valArray {
							if str, ok := item.(string); ok {
								strArray = append(strArray, str)
							}
						}
						config.AllowedMap[key] = strArray
					}
				}
			}
		}

		return config
	}

	return &ClaudeToolConfig{}
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

// ToMap converts the Tools struct back to a map[string]any for backwards compatibility
func (t *Tools) ToMap() map[string]any {
	if t == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)

	// Add known tools if they exist (use has flags to check)
	if t.hasGitHub {
		result["github"] = convertGitHubConfigToMap(t.GitHub)
	}
	if t.hasClaude {
		result["claude"] = convertClaudeConfigToMap(t.Claude)
	}
	if t.hasBash {
		result["bash"] = convertBashConfigToMap(t.Bash)
	}
	if t.hasWebFetch {
		result["web-fetch"] = convertWebFetchConfigToMap(t.WebFetch)
	}
	if t.hasWebSearch {
		result["web-search"] = convertWebSearchConfigToMap(t.WebSearch)
	}
	if t.hasEdit {
		result["edit"] = convertEditConfigToMap(t.Edit)
	}
	if t.hasPlaywright {
		result["playwright"] = convertPlaywrightConfigToMap(t.Playwright)
	}
	if t.hasAgenticWorkflows {
		result["agentic-workflows"] = convertAgenticWorkflowsConfigToMap(t.AgenticWorkflows)
	}
	if t.hasCacheMemory {
		result["cache-memory"] = convertCacheMemoryConfigToMap(t.CacheMemory)
	}
	if t.hasSafetyPrompt {
		if t.SafetyPrompt != nil {
			result["safety-prompt"] = *t.SafetyPrompt
		} else {
			result["safety-prompt"] = nil
		}
	}
	if t.hasTimeout {
		if t.Timeout != nil {
			result["timeout"] = *t.Timeout
		} else {
			result["timeout"] = nil
		}
	}
	if t.hasStartupTimeout {
		if t.StartupTimeout != nil {
			result["startup-timeout"] = *t.StartupTimeout
		} else {
			result["startup-timeout"] = nil
		}
	}

	// Add custom MCP tools
	for name, config := range t.Custom {
		result[name] = config
	}

	return result
}

// Helper functions to convert typed configs back to map representations

func convertGitHubConfigToMap(config *GitHubToolConfig) any {
	if config == nil {
		return nil
	}

	// If all fields are empty, return nil (simple enable)
	if len(config.Allowed) == 0 && config.Mode == "" && config.Version == "" &&
		len(config.Args) == 0 && !config.ReadOnly && config.GitHubToken == "" && len(config.Toolset) == 0 {
		return nil
	}

	m := make(map[string]any)
	if len(config.Allowed) > 0 {
		allowed := make([]any, len(config.Allowed))
		for i, v := range config.Allowed {
			allowed[i] = v
		}
		m["allowed"] = allowed
	}
	if config.Mode != "" {
		m["mode"] = config.Mode
	}
	if config.Version != "" {
		m["version"] = config.Version
	}
	if len(config.Args) > 0 {
		args := make([]any, len(config.Args))
		for i, v := range config.Args {
			args[i] = v
		}
		m["args"] = args
	}
	if config.ReadOnly {
		m["read-only"] = config.ReadOnly
	}
	if config.GitHubToken != "" {
		m["github-token"] = config.GitHubToken
	}
	if len(config.Toolset) > 0 {
		toolset := make([]any, len(config.Toolset))
		for i, v := range config.Toolset {
			toolset[i] = v
		}
		m["toolset"] = toolset
	}

	if len(m) == 0 {
		return nil
	}
	return m
}

func convertClaudeConfigToMap(config *ClaudeToolConfig) any {
	if config == nil {
		return nil
	}

	// If both are empty, return nil
	if len(config.AllowedTools) == 0 && len(config.AllowedMap) == 0 {
		return nil
	}

	m := make(map[string]any)

	if len(config.AllowedTools) > 0 {
		allowed := make([]any, len(config.AllowedTools))
		for i, v := range config.AllowedTools {
			allowed[i] = v
		}
		m["allowed"] = allowed
	} else if len(config.AllowedMap) > 0 {
		allowedMap := make(map[string]any)
		for key, vals := range config.AllowedMap {
			arr := make([]any, len(vals))
			for i, v := range vals {
				arr[i] = v
			}
			allowedMap[key] = arr
		}
		m["allowed"] = allowedMap
	}

	if len(m) == 0 {
		return nil
	}
	return m
}

func convertBashConfigToMap(config *BashToolConfig) any {
	if config == nil {
		return nil
	}

	if len(config.AllowedCommands) == 0 {
		return nil // nil means all commands allowed
	}

	commands := make([]any, len(config.AllowedCommands))
	for i, v := range config.AllowedCommands {
		commands[i] = v
	}
	return commands
}

func convertPlaywrightConfigToMap(config *PlaywrightToolConfig) any {
	if config == nil {
		return nil
	}

	// If all fields are empty, return nil
	if config.Version == "" && len(config.AllowedDomains) == 0 && len(config.Args) == 0 {
		return nil
	}

	m := make(map[string]any)
	if config.Version != "" {
		m["version"] = config.Version
	}
	if len(config.AllowedDomains) > 0 {
		// Convert back to array
		domains := make([]any, len(config.AllowedDomains))
		for i, v := range config.AllowedDomains {
			domains[i] = v
		}
		m["allowed_domains"] = domains
	}
	if len(config.Args) > 0 {
		args := make([]any, len(config.Args))
		for i, v := range config.Args {
			args[i] = v
		}
		m["args"] = args
	}

	if len(m) == 0 {
		return nil
	}
	return m
}

func convertWebFetchConfigToMap(config *WebFetchToolConfig) any {
	if config == nil {
		return nil
	}
	return nil // or empty object
}

func convertWebSearchConfigToMap(config *WebSearchToolConfig) any {
	if config == nil {
		return nil
	}
	return nil // or empty object
}

func convertEditConfigToMap(config *EditToolConfig) any {
	if config == nil {
		return nil
	}
	return nil // or empty object
}

func convertAgenticWorkflowsConfigToMap(config *AgenticWorkflowsToolConfig) any {
	if config == nil {
		return nil
	}
	return config.Enabled
}

func convertCacheMemoryConfigToMap(config *CacheMemoryToolConfig) any {
	if config == nil {
		return nil
	}
	return config.Raw
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
// Returns the typed config struct (or pointer to primitive type) cast to any for backwards compatibility
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

// GetGitHubConfig returns the GitHub tool configuration
// Since GitHub field is now properly typed, this just returns it directly
func (t *Tools) GetGitHubConfig() *GitHubToolConfig {
	if t == nil {
		return nil
	}
	return t.GitHub
}

// GetPlaywrightConfig returns the Playwright tool configuration
// Since Playwright field is now properly typed, this just returns it directly
func (t *Tools) GetPlaywrightConfig() *PlaywrightToolConfig {
	if t == nil {
		return nil
	}
	return t.Playwright
}

// GetClaudeConfig returns the Claude tool configuration
// Since Claude field is now properly typed, this just returns it directly
func (t *Tools) GetClaudeConfig() *ClaudeToolConfig {
	if t == nil {
		return nil
	}
	return t.Claude
}
