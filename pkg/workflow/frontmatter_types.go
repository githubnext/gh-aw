package workflow

import (
	"encoding/json"
	"fmt"
)

// FrontmatterConfig represents the structured configuration from workflow frontmatter
// This provides compile-time type safety and clearer error messages compared to map[string]any
type FrontmatterConfig struct {
	// Core workflow fields
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Engine      string         `json:"engine,omitempty"`
	Source      string         `json:"source,omitempty"`
	TrackerID   string         `json:"tracker-id,omitempty"`
	Version     string         `json:"version,omitempty"`
	TimeoutMinutes int         `json:"timeout-minutes,omitempty"`
	
	// Configuration sections
	Tools       map[string]any `json:"tools,omitempty"`
	MCPServers  map[string]any `json:"mcp-servers,omitempty"`
	Runtimes    map[string]any `json:"runtimes,omitempty"`
	Jobs        map[string]any `json:"jobs,omitempty"`
	SafeOutputs map[string]any `json:"safe-outputs,omitempty"`
	SafeJobs    map[string]any `json:"safe-jobs,omitempty"`
	SafeInputs  map[string]any `json:"safe-inputs,omitempty"`
	
	// Event and trigger configuration
	On          map[string]any `json:"on,omitempty"`
	Permissions map[string]any `json:"permissions,omitempty"`
	Concurrency map[string]any `json:"concurrency,omitempty"`
	If          string         `json:"if,omitempty"`
	
	// Network and sandbox configuration
	Network     map[string]any `json:"network,omitempty"`
	Sandbox     map[string]any `json:"sandbox,omitempty"`
	
	// Feature flags and other settings
	Features    map[string]any `json:"features,omitempty"`
	Env         map[string]any `json:"env,omitempty"`
	Secrets     map[string]any `json:"secrets,omitempty"`
	
	// Workflow execution settings
	RunsOn      string         `json:"runs-on,omitempty"`
	RunName     string         `json:"run-name,omitempty"`
	
	// Import and inclusion
	Imports     any            `json:"imports,omitempty"` // Can be string or array
	Include     any            `json:"include,omitempty"` // Can be string or array
}

// unmarshalFromMap converts a value from a map[string]any to a destination variable
// using JSON marshaling/unmarshaling for type conversion.
// This provides cleaner error messages than manual type assertions.
//
// Parameters:
//   - data: The map containing the configuration data
//   - key: The key to extract from the map
//   - dest: Pointer to the destination variable to unmarshal into (can be any type)
//
// Returns an error if:
//   - The key doesn't exist in the map
//   - The value cannot be marshaled to JSON
//   - The JSON cannot be unmarshaled into the destination type
//
// Example:
//   var name string
//   err := unmarshalFromMap(frontmatter, "name", &name)
//
//   var tools map[string]any
//   err := unmarshalFromMap(frontmatter, "tools", &tools)
func unmarshalFromMap(data map[string]any, key string, dest any) error {
	value, exists := data[key]
	if !exists {
		return fmt.Errorf("key '%s' not found in frontmatter", key)
	}
	
	// Use JSON as intermediate format for type conversion
	// This handles nested maps, arrays, and complex structures cleanly
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal '%s' to JSON: %w", key, err)
	}
	
	if err := json.Unmarshal(jsonBytes, dest); err != nil {
		return fmt.Errorf("failed to unmarshal '%s' into destination type: %w", key, err)
	}
	
	return nil
}

// ParseFrontmatterConfig creates a FrontmatterConfig from a raw frontmatter map
// This provides a single entry point for converting untyped frontmatter into
// a structured configuration with better error handling.
func ParseFrontmatterConfig(frontmatter map[string]any) (*FrontmatterConfig, error) {
	var config FrontmatterConfig
	
	// Use JSON marshaling for the entire frontmatter conversion
	// This automatically handles all field mappings
	jsonBytes, err := json.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter to JSON: %w", err)
	}
	
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter into config: %w", err)
	}
	
	return &config, nil
}

// ExtractMapField is a convenience wrapper around unmarshalFromMap for extracting
// map[string]any fields from frontmatter. This maintains backward compatibility
// with existing extraction patterns while providing better error messages.
//
// Returns an empty map if the key doesn't exist (for backward compatibility).
func ExtractMapField(frontmatter map[string]any, key string) map[string]any {
	// Check if key exists and value is not nil
	value, exists := frontmatter[key]
	if !exists || value == nil {
		return make(map[string]any)
	}
	
	var result map[string]any
	err := unmarshalFromMap(frontmatter, key, &result)
	if err != nil {
		// For backward compatibility, return empty map instead of error
		return make(map[string]any)
	}
	
	// Handle case where unmarshal succeeded but result is still nil
	if result == nil {
		return make(map[string]any)
	}
	
	return result
}

// ExtractStringField is a convenience wrapper for extracting string fields.
// Returns empty string if the key doesn't exist or cannot be converted.
func ExtractStringField(frontmatter map[string]any, key string) string {
	var result string
	err := unmarshalFromMap(frontmatter, key, &result)
	if err != nil {
		return ""
	}
	return result
}

// ExtractIntField is a convenience wrapper for extracting integer fields.
// Returns 0 if the key doesn't exist or cannot be converted.
func ExtractIntField(frontmatter map[string]any, key string) int {
	var result int
	err := unmarshalFromMap(frontmatter, key, &result)
	if err != nil {
		return 0
	}
	return result
}
