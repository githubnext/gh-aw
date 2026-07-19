package workflow

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var mcpScriptsLog = logger.New("workflow:mcp_scripts")

// parseTimeoutString converts a string timeout value to seconds.
// It accepts plain integers ("120"), Go duration strings ("6m", "1h", "30s", "2h30m"),
// and trims surrounding whitespace. Returns (seconds, true) on success,
// or (0, false) if the value cannot be parsed.
func parseTimeoutString(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// Plain integer – e.g. "120"
	if n, err := strconv.Atoi(s); err == nil {
		return n, true
	}
	// Go duration – e.g. "6m", "1h", "30s", "2h30m"
	if d, err := time.ParseDuration(s); err == nil {
		return int(d.Seconds()), true
	}
	return 0, false
}

// MCPScriptsConfig holds the configuration for mcp-scripts custom tools
type MCPScriptsConfig struct {
	Mode  string // Transport mode: "http" (default) or "stdio"
	Tools map[string]*MCPScriptToolConfig
}

// MCPScriptToolConfig holds the configuration for a single mcp-script tool
type MCPScriptToolConfig struct {
	Name         string                     // Tool name (key from the config)
	Description  string                     // Required: tool description
	Inputs       map[string]*MCPScriptParam // Optional: input parameters
	Script       string                     // JavaScript implementation (mutually exclusive with Run, Py, and Go)
	Run          string                     // Shell script implementation (mutually exclusive with Script, Py, and Go)
	Py           string                     // Python script implementation (mutually exclusive with Script, Run, and Go)
	Go           string                     // Go script implementation (mutually exclusive with Script, Run, and Py)
	Dependencies []string                   // Optional runtime dependencies
	Env          map[string]string          // Environment variables (typically for secrets)
	Timeout      int                        // Timeout in seconds for tool execution (default: 60)
}

// MCPScriptParam holds the configuration for a tool input parameter
type MCPScriptParam struct {
	Type        string // JSON schema type (string, number, boolean, array, object)
	Description string // Description of the parameter
	Required    bool   // Whether the parameter is required
	Default     any    // Default value
}

// MCPScriptsMode constants define the available transport modes
const (
	MCPScriptsModeHTTP = "http"
)

// MCPScriptsDirectory is the directory where mcp-scripts files are generated
const MCPScriptsDirectory = GhAwMCPScriptsDir

// HasMCPScripts checks if mcp-scripts are configured
func HasMCPScripts(mcpScripts *MCPScriptsConfig) bool {
	return mcpScripts != nil && len(mcpScripts.Tools) > 0
}

// IsMCPScriptsEnabled checks if mcp-scripts are configured.
// MCP Scripts are enabled by default when configured in the workflow.
func IsMCPScriptsEnabled(mcpScripts *MCPScriptsConfig) bool {
	return HasMCPScripts(mcpScripts)
}

// parseMCPScriptToolConfig parses a single MCP script tool configuration from a map.
// It initialises all fields to their defaults and populates them from toolMap.
func parseMCPScriptToolConfig(toolName string, toolMap map[string]any) *MCPScriptToolConfig {
	toolConfig := &MCPScriptToolConfig{
		Name:    toolName,
		Inputs:  make(map[string]*MCPScriptParam),
		Env:     make(map[string]string),
		Timeout: 60, // Default timeout: 60 seconds
	}

	// Parse description (required)
	if desc, exists := toolMap["description"]; exists {
		if descStr, ok := desc.(string); ok {
			toolConfig.Description = descStr
		}
	}

	parseMCPScriptInputs(toolMap, toolConfig)

	// Parse script (JavaScript implementation)
	if script, exists := toolMap["script"]; exists {
		if scriptStr, ok := script.(string); ok {
			toolConfig.Script = scriptStr
		}
	}

	// Parse run (shell script implementation)
	if run, exists := toolMap["run"]; exists {
		if runStr, ok := run.(string); ok {
			toolConfig.Run = runStr
		}
	}

	// Parse py (Python script implementation)
	if py, exists := toolMap["py"]; exists {
		if pyStr, ok := py.(string); ok {
			toolConfig.Py = pyStr
		}
	}

	// Parse go (Go script implementation)
	if goScript, exists := toolMap["go"]; exists {
		if goStr, ok := goScript.(string); ok {
			toolConfig.Go = goStr
		}
	}

	parseMCPScriptEnv(toolMap, toolConfig)
	parseMCPScriptDependencies(toolMap, toolConfig)
	parseMCPScriptTimeout(toolName, toolMap, toolConfig)

	return toolConfig
}

func parseMCPScriptInputs(toolMap map[string]any, toolConfig *MCPScriptToolConfig) {
	inputsMap, ok := toolMap["inputs"].(map[string]any)
	if !ok {
		return
	}
	for paramName, paramValue := range inputsMap {
		if paramMap, ok := paramValue.(map[string]any); ok {
			toolConfig.Inputs[paramName] = parseMCPScriptParam(paramMap)
		}
	}
}

func parseMCPScriptParam(paramMap map[string]any) *MCPScriptParam {
	param := &MCPScriptParam{Type: "string"}
	if t, exists := paramMap["type"]; exists {
		if tStr, ok := t.(string); ok {
			param.Type = tStr
		}
	}
	if desc, exists := paramMap["description"]; exists {
		if descStr, ok := desc.(string); ok {
			param.Description = descStr
		}
	}
	if req, exists := paramMap["required"]; exists {
		if reqBool, ok := req.(bool); ok {
			param.Required = reqBool
		}
	}
	if def, exists := paramMap["default"]; exists {
		param.Default = def
	}
	return param
}

func parseMCPScriptEnv(toolMap map[string]any, toolConfig *MCPScriptToolConfig) {
	envMap, ok := toolMap["env"].(map[string]any)
	if !ok {
		return
	}
	for envName, envValue := range envMap {
		if envStr, ok := envValue.(string); ok {
			toolConfig.Env[envName] = envStr
		}
	}
}

func parseMCPScriptDependencies(toolMap map[string]any, toolConfig *MCPScriptToolConfig) {
	depsList, ok := toolMap["dependencies"].([]any)
	if !ok {
		return
	}
	for _, dep := range depsList {
		if depStr, ok := dep.(string); ok {
			toolConfig.Dependencies = append(toolConfig.Dependencies, depStr)
		}
	}
}

func parseMCPScriptTimeout(toolName string, toolMap map[string]any, toolConfig *MCPScriptToolConfig) {
	timeout, exists := toolMap["timeout"]
	if !exists {
		return
	}
	switch t := timeout.(type) {
	case int:
		toolConfig.Timeout = t
	case uint64:
		toolConfig.Timeout = typeutil.SafeUint64ToInt(t)
	case float64:
		parseMCPScriptFloatTimeout(toolName, t, toolConfig)
	case string:
		parseMCPScriptStringTimeout(toolName, t, toolConfig)
	}
}

func parseMCPScriptFloatTimeout(toolName string, timeout float64, toolConfig *MCPScriptToolConfig) {
	maxInt := int(^uint(0) >> 1)
	if timeout != timeout || timeout < 0 || timeout > float64(maxInt) {
		mcpScriptsLog.Printf("Warning: invalid timeout value %v for tool %q, keeping default timeout (60s)", timeout, toolName)
		return
	}
	toolConfig.Timeout = int(timeout)
}

func parseMCPScriptStringTimeout(toolName, timeout string, toolConfig *MCPScriptToolConfig) {
	if n, ok := parseTimeoutString(timeout); ok {
		toolConfig.Timeout = n
		return
	}
	mcpScriptsLog.Printf("Warning: invalid timeout value %q for tool %q, keeping default timeout (60s)", timeout, toolName)
}

// parseMCPScriptsMap parses mcp-scripts configuration from a map.
// It is used by extractMCPScriptsConfig to convert frontmatter into an MCPScriptsConfig.
// Returns the config and a boolean indicating whether any tools were found.
func parseMCPScriptsMap(mcpScriptsMap map[string]any) (*MCPScriptsConfig, bool) {
	config := &MCPScriptsConfig{
		Mode:  "http", // Only HTTP mode is supported
		Tools: make(map[string]*MCPScriptToolConfig),
	}

	// Mode field is ignored - only HTTP mode is supported
	// All mcp-scripts configurations use HTTP transport

	for toolName, toolValue := range mcpScriptsMap {
		// Skip the "mode" field as it's not a tool definition
		if toolName == "mode" {
			continue
		}

		toolMap, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		config.Tools[toolName] = parseMCPScriptToolConfig(toolName, toolMap)
	}

	return config, len(config.Tools) > 0
}

// extractMCPScriptsConfig extracts mcp-scripts configuration from frontmatter
func (c *Compiler) extractMCPScriptsConfig(frontmatter map[string]any) *MCPScriptsConfig {
	mcpScriptsLog.Print("Extracting mcp-scripts configuration from frontmatter")

	mcpScripts, exists := frontmatter["mcp-scripts"]
	if !exists {
		return nil
	}

	mcpScriptsMap, ok := mcpScripts.(map[string]any)
	if !ok {
		return nil
	}

	config, hasTools := parseMCPScriptsMap(mcpScriptsMap)
	if !hasTools {
		return nil
	}

	mcpScriptsLog.Printf("Extracted %d mcp-script tools", len(config.Tools))
	return config
}

// mergeMCPScripts merges mcp-scripts configuration from imports into the main configuration
func (c *Compiler) mergeMCPScripts(main *MCPScriptsConfig, importedConfigs []string) *MCPScriptsConfig {
	if main == nil {
		main = &MCPScriptsConfig{
			Mode:  "http", // Default to HTTP mode
			Tools: make(map[string]*MCPScriptToolConfig),
		}
	}

	for _, configJSON := range importedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Merge the imported JSON config
		var importedMap map[string]any
		if err := json.Unmarshal([]byte(configJSON), &importedMap); err != nil {
			mcpScriptsLog.Printf("Warning: failed to parse imported mcp-scripts config: %v", err)
			continue
		}

		// Mode field is ignored - only HTTP mode is supported
		// All mcp-scripts configurations use HTTP transport

		// Merge each tool from the imported config
		for toolName, toolValue := range importedMap {
			// Skip mode field as it's already handled
			if toolName == "mode" {
				continue
			}

			// Skip if tool already exists in main config (main takes precedence)
			if _, exists := main.Tools[toolName]; exists {
				mcpScriptsLog.Printf("Skipping imported tool '%s' - already defined in main config", toolName)
				continue
			}

			toolMap, ok := toolValue.(map[string]any)
			if !ok {
				continue
			}

			main.Tools[toolName] = parseMCPScriptToolConfig(toolName, toolMap)
			mcpScriptsLog.Printf("Merged imported mcp-script tool: %s", toolName)
		}
	}

	return main
}
