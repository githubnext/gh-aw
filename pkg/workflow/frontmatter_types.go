package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var frontmatterTypesLog = logger.New("workflow:frontmatter_types")

// FrontmatterConfig represents the structured configuration from workflow frontmatter
// This provides compile-time type safety and clearer error messages compared to map[string]any
type FrontmatterConfig struct {
	// Core workflow fields
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	Engine         string `json:"engine,omitempty"`
	Source         string `json:"source,omitempty"`
	TrackerID      string `json:"tracker-id,omitempty"`
	Version        string `json:"version,omitempty"`
	TimeoutMinutes int    `json:"timeout-minutes,omitempty"`

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
	Network map[string]any `json:"network,omitempty"`
	Sandbox map[string]any `json:"sandbox,omitempty"`

	// Feature flags and other settings
	Features map[string]any `json:"features,omitempty"`
	Env      map[string]any `json:"env,omitempty"`
	Secrets  map[string]any `json:"secrets,omitempty"`

	// Workflow execution settings
	RunsOn  string `json:"runs-on,omitempty"`
	RunName string `json:"run-name,omitempty"`

	// Import and inclusion
	Imports any `json:"imports,omitempty"` // Can be string or array
	Include any `json:"include,omitempty"` // Can be string or array
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
//
//	var name string
//	err := unmarshalFromMap(frontmatter, "name", &name)
//
//	var tools map[string]any
//	err := unmarshalFromMap(frontmatter, "tools", &tools)
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
	frontmatterTypesLog.Printf("Parsing frontmatter config with %d fields", len(frontmatter))
	var config FrontmatterConfig

	// Use JSON marshaling for the entire frontmatter conversion
	// This automatically handles all field mappings
	jsonBytes, err := json.Marshal(frontmatter)
	if err != nil {
		frontmatterTypesLog.Printf("Failed to marshal frontmatter: %v", err)
		return nil, fmt.Errorf("failed to marshal frontmatter to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		frontmatterTypesLog.Printf("Failed to unmarshal frontmatter: %v", err)
		return nil, fmt.Errorf("failed to unmarshal frontmatter into config: %w", err)
	}

	frontmatterTypesLog.Printf("Successfully parsed frontmatter config: name=%s, engine=%s", config.Name, config.Engine)
	return &config, nil
}

// ExtractMapField is a convenience wrapper for extracting map[string]any fields
// from frontmatter. This maintains backward compatibility with existing extraction
// patterns while preserving original types (avoiding JSON conversion which would
// convert all numbers to float64).
//
// Returns an empty map if the key doesn't exist (for backward compatibility).
func ExtractMapField(frontmatter map[string]any, key string) map[string]any {
	// Check if key exists and value is not nil
	value, exists := frontmatter[key]
	if !exists || value == nil {
		frontmatterTypesLog.Printf("Field '%s' not found in frontmatter, returning empty map", key)
		return make(map[string]any)
	}

	// Direct type assertion to preserve original types (especially integers)
	// This avoids JSON marshaling which would convert integers to float64
	if valueMap, ok := value.(map[string]any); ok {
		frontmatterTypesLog.Printf("Extracted map field '%s' with %d entries", key, len(valueMap))
		return valueMap
	}

	// For backward compatibility, return empty map if not a map
	frontmatterTypesLog.Printf("Field '%s' is not a map type, returning empty map", key)
	return make(map[string]any)
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
