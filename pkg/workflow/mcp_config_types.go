package workflow

// mcp_config_types.go contains type definitions and interfaces for MCP configuration.
// This file defines the core types used across MCP configuration rendering and parsing.

// ToolConfig represents a tool configuration interface for type safety
// It provides type-safe accessors for common configuration value types
type ToolConfig interface {
	// GetString retrieves a string value from the configuration
	GetString(key string) (string, bool)
	// GetStringArray retrieves a string array from the configuration
	GetStringArray(key string) ([]string, bool)
	// GetStringMap retrieves a string map from the configuration
	GetStringMap(key string) (map[string]string, bool)
	// GetAny retrieves any value from the configuration (fallback for complex types)
	GetAny(key string) (any, bool)
}

// MapToolConfig implements ToolConfig for map[string]any
// It provides type-safe access to tool configuration values stored in a map
type MapToolConfig map[string]any

// GetString retrieves a string value from the configuration map
// Returns the string value and true if the key exists and the value is a string
func (m MapToolConfig) GetString(key string) (string, bool) {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetStringArray retrieves a string array from the configuration map
// Handles both []any and []string formats
// Returns the string array and true if the key exists and the value can be converted
func (m MapToolConfig) GetStringArray(key string) ([]string, bool) {
	if value, exists := m[key]; exists {
		if arr, ok := value.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result, true
		}
		if arr, ok := value.([]string); ok {
			return arr, true
		}
	}
	return nil, false
}

// GetStringMap retrieves a string map from the configuration map
// Handles both map[string]any and map[string]string formats
// Returns the string map and true if the key exists and the value can be converted
func (m MapToolConfig) GetStringMap(key string) (map[string]string, bool) {
	if value, exists := m[key]; exists {
		if mapVal, ok := value.(map[string]any); ok {
			result := make(map[string]string)
			for k, v := range mapVal {
				if str, ok := v.(string); ok {
					result[k] = str
				}
			}
			return result, true
		}
		if mapVal, ok := value.(map[string]string); ok {
			return mapVal, true
		}
	}
	return nil, false
}

// GetAny retrieves any value from the configuration map
// This is a fallback for complex types that don't fit the typed accessors
// Returns the value and true if the key exists
func (m MapToolConfig) GetAny(key string) (any, bool) {
	value, exists := m[key]
	return value, exists
}

// MCPConfigRenderer contains configuration options for rendering MCP config
// It controls format-specific rendering options and features
type MCPConfigRenderer struct {
	// IndentLevel controls the indentation level for properties (e.g., "                " for JSON, "          " for TOML)
	IndentLevel string
	// Format specifies the output format ("json" for JSON-like, "toml" for TOML-like)
	Format string
	// RequiresCopilotFields indicates if the engine requires "type" and "tools" fields (true for copilot engine)
	RequiresCopilotFields bool
	// RewriteLocalhostToDocker indicates if localhost URLs should be rewritten to host.docker.internal
	// This is needed when the agent runs inside a firewall container and needs to access MCP servers on the host
	RewriteLocalhostToDocker bool
}
