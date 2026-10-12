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

type MCPParamType string

const (
	MCPParamTypeString  MCPParamType = "string"
	MCPParamTypeNumber  MCPParamType = "number"
	MCPParamTypeBoolean MCPParamType = "boolean"
	MCPParamTypeArray   MCPParamType = "array"
	MCPParamTypeObject  MCPParamType = "object"
)

// MCPScriptParam holds the configuration for a tool input parameter
type MCPScriptParam struct {
	Type        MCPParamType         // JSON schema type (string, number, boolean, array, object)
	Items       *MCPScriptParamItems // Optional element type for array parameters
	Description string               // Description of the parameter
	Required    bool                 // Whether the parameter is required
	Default     any                  // Default value
}

// MCPScriptParamItems constrains array elements to a JSON schema type.
type MCPScriptParamItems struct {
	Type MCPParamType `json:"type"`
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
		Inputs:  parseMCPScriptInputs(toolMap["inputs"]),
		Env:     make(map[string]string),
		Timeout: parseMCPScriptTimeout(toolName, toolMap["timeout"]),
	}

	// Parse description (required)
	if desc, ok := toolMap["description"].(string); ok {
		toolConfig.Description = desc
	}

	// Parse script (JavaScript implementation)
	if script, ok := toolMap["script"].(string); ok {
		toolConfig.Script = script
	}

	// Parse run (shell script implementation)
	if run, ok := toolMap["run"].(string); ok {
		toolConfig.Run = run
	}

	// Parse py (Python script implementation)
	if py, ok := toolMap["py"].(string); ok {
		toolConfig.Py = py
	}

	// Parse go (Go script implementation)
	if goScript, ok := toolMap["go"].(string); ok {
		toolConfig.Go = goScript
	}

	// Parse env (environment variables)
	if env, exists := toolMap["env"]; exists {
		if envMap, ok := env.(map[string]any); ok {
			for envName, envValue := range envMap {
				if envStr, ok := envValue.(string); ok {
					toolConfig.Env[envName] = envStr
				}
			}
		}
	}

	// Parse dependencies (optional list of strings)
	if dependencies, exists := toolMap["dependencies"]; exists {
		if depsList, ok := dependencies.([]any); ok {
			for _, dep := range depsList {
				if depStr, ok := dep.(string); ok {
					toolConfig.Dependencies = append(toolConfig.Dependencies, depStr)
				}
			}
		}
	}

	return toolConfig
}

func parseMCPScriptInputs(inputs any) map[string]*MCPScriptParam {
	params := make(map[string]*MCPScriptParam)
	inputsMap, ok := inputs.(map[string]any)
	if !ok {
		return params
	}
	for name, value := range inputsMap {
		paramMap, ok := value.(map[string]any)
		if !ok {
			continue
		}
		param := &MCPScriptParam{Type: MCPParamTypeString}
		if paramType, ok := paramMap["type"].(string); ok {
			param.Type = MCPParamType(paramType)
		}
		if items, ok := paramMap["items"].(map[string]any); ok {
			if itemType, ok := items["type"].(string); ok {
				param.Items = &MCPScriptParamItems{Type: MCPParamType(itemType)}
			}
		}
		if description, ok := paramMap["description"].(string); ok {
			param.Description = description
		}
		if required, ok := paramMap["required"].(bool); ok {
			param.Required = required
		}
		param.Default = paramMap["default"]
		params[name] = param
	}
	return params
}

func parseMCPScriptTimeout(toolName string, timeout any) int {
	switch t := timeout.(type) {
	case int:
		return t
	case uint64:
		return typeutil.SafeUint64ToInt(t)
	case float64:
		maxInt := int(^uint(0) >> 1)
		if t != t || t < 0 || t > float64(maxInt) {
			mcpScriptsLog.Printf("Warning: invalid timeout value %v for tool %q, keeping default timeout (60s)", t, toolName)
		} else {
			return int(t)
		}
	case string:
		if n, ok := parseTimeoutString(t); ok {
			return n
		}
		mcpScriptsLog.Printf("Warning: invalid timeout value %q for tool %q, keeping default timeout (60s)", t, toolName)
	}
	return 60
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
