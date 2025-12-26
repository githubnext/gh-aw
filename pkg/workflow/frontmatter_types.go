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
	Strict         *bool  `json:"strict,omitempty"` // Pointer to distinguish unset from false

	// Configuration sections - using strongly-typed structs
	Tools       *ToolsConfig       `json:"tools,omitempty"`
	MCPServers  map[string]any     `json:"mcp-servers,omitempty"` // Legacy field, use Tools instead
	Runtimes    map[string]any     `json:"runtimes,omitempty"`
	Jobs        map[string]any     `json:"jobs,omitempty"`       // Custom workflow jobs
	SafeOutputs *SafeOutputsConfig `json:"safe-outputs,omitempty"`
	SafeJobs    map[string]any     `json:"safe-jobs,omitempty"` // Deprecated, use SafeOutputs.Jobs
	SafeInputs  *SafeInputsConfig  `json:"safe-inputs,omitempty"`

	// Event and trigger configuration
	On          map[string]any `json:"on,omitempty"`          // Complex trigger config with many variants
	Permissions map[string]any `json:"permissions,omitempty"` // Can be string or map
	Concurrency map[string]any `json:"concurrency,omitempty"`
	If          string         `json:"if,omitempty"`

	// Network and sandbox configuration
	Network *NetworkPermissions `json:"network,omitempty"`
	Sandbox *SandboxConfig      `json:"sandbox,omitempty"`

	// Feature flags and other settings
	Features map[string]any    `json:"features,omitempty"` // Dynamic feature flags
	Env      map[string]string `json:"env,omitempty"`
	Secrets  map[string]any    `json:"secrets,omitempty"`

	// Workflow execution settings
	RunsOn      string         `json:"runs-on,omitempty"`
	RunName     string         `json:"run-name,omitempty"`
	Steps       []any          `json:"steps,omitempty"`       // Custom workflow steps
	PostSteps   []any          `json:"post-steps,omitempty"`  // Post-workflow steps
	Environment map[string]any `json:"environment,omitempty"` // GitHub environment
	Container   map[string]any `json:"container,omitempty"`
	Services    map[string]any `json:"services,omitempty"`
	Cache       map[string]any `json:"cache,omitempty"`

	// Import and inclusion
	Imports any `json:"imports,omitempty"` // Can be string or array
	Include any `json:"include,omitempty"` // Can be string or array

	// Metadata
	Metadata        map[string]string `json:"metadata,omitempty"` // Custom metadata key-value pairs
	SecretMasking   *SecretMaskingConfig `json:"secret-masking,omitempty"`
	GithubToken     string            `json:"github-token,omitempty"`
	
	// Command/bot configuration
	Roles []string `json:"roles,omitempty"`
	Bots  []string `json:"bots,omitempty"`
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

// ToMap converts FrontmatterConfig back to map[string]any for backward compatibility
// This allows gradual migration from map[string]any to strongly-typed config
func (fc *FrontmatterConfig) ToMap() map[string]any {
	result := make(map[string]any)
	
	// Core fields
	if fc.Name != "" {
		result["name"] = fc.Name
	}
	if fc.Description != "" {
		result["description"] = fc.Description
	}
	if fc.Engine != "" {
		result["engine"] = fc.Engine
	}
	if fc.Source != "" {
		result["source"] = fc.Source
	}
	if fc.TrackerID != "" {
		result["tracker-id"] = fc.TrackerID
	}
	if fc.Version != "" {
		result["version"] = fc.Version
	}
	if fc.TimeoutMinutes != 0 {
		result["timeout-minutes"] = fc.TimeoutMinutes
	}
	if fc.Strict != nil {
		result["strict"] = *fc.Strict
	}
	
	// Configuration sections
	if fc.Tools != nil {
		result["tools"] = fc.Tools.ToMap()
	}
	if fc.MCPServers != nil {
		result["mcp-servers"] = fc.MCPServers
	}
	if fc.Runtimes != nil {
		result["runtimes"] = fc.Runtimes
	}
	if fc.Jobs != nil {
		result["jobs"] = fc.Jobs
	}
	if fc.SafeOutputs != nil {
		// Convert SafeOutputsConfig to map - would need a ToMap method
		result["safe-outputs"] = fc.SafeOutputs
	}
	if fc.SafeJobs != nil {
		result["safe-jobs"] = fc.SafeJobs
	}
	if fc.SafeInputs != nil {
		// Convert SafeInputsConfig to map - would need a ToMap method
		result["safe-inputs"] = fc.SafeInputs
	}
	
	// Event and trigger configuration
	if fc.On != nil {
		result["on"] = fc.On
	}
	if fc.Permissions != nil {
		result["permissions"] = fc.Permissions
	}
	if fc.Concurrency != nil {
		result["concurrency"] = fc.Concurrency
	}
	if fc.If != "" {
		result["if"] = fc.If
	}
	
	// Network and sandbox
	if fc.Network != nil {
		// Convert NetworkPermissions to map format
		networkMap := make(map[string]any)
		if fc.Network.Mode != "" {
			if fc.Network.Mode == "defaults" {
				result["network"] = "defaults"
			} else {
				networkMap["mode"] = fc.Network.Mode
			}
		}
		if len(fc.Network.Allowed) > 0 {
			networkMap["allowed"] = fc.Network.Allowed
		}
		if fc.Network.Firewall != nil {
			networkMap["firewall"] = fc.Network.Firewall
		}
		if len(networkMap) > 0 {
			result["network"] = networkMap
		}
	}
	if fc.Sandbox != nil {
		result["sandbox"] = fc.Sandbox
	}
	
	// Features and environment
	if fc.Features != nil {
		result["features"] = fc.Features
	}
	if fc.Env != nil {
		result["env"] = fc.Env
	}
	if fc.Secrets != nil {
		result["secrets"] = fc.Secrets
	}
	
	// Execution settings
	if fc.RunsOn != "" {
		result["runs-on"] = fc.RunsOn
	}
	if fc.RunName != "" {
		result["run-name"] = fc.RunName
	}
	if fc.Steps != nil {
		result["steps"] = fc.Steps
	}
	if fc.PostSteps != nil {
		result["post-steps"] = fc.PostSteps
	}
	if fc.Environment != nil {
		result["environment"] = fc.Environment
	}
	if fc.Container != nil {
		result["container"] = fc.Container
	}
	if fc.Services != nil {
		result["services"] = fc.Services
	}
	if fc.Cache != nil {
		result["cache"] = fc.Cache
	}
	
	// Import and inclusion
	if fc.Imports != nil {
		result["imports"] = fc.Imports
	}
	if fc.Include != nil {
		result["include"] = fc.Include
	}
	
	// Metadata
	if fc.Metadata != nil {
		result["metadata"] = fc.Metadata
	}
	if fc.SecretMasking != nil {
		result["secret-masking"] = fc.SecretMasking
	}
	if fc.GithubToken != "" {
		result["github-token"] = fc.GithubToken
	}
	if fc.Roles != nil {
		result["roles"] = fc.Roles
	}
	if fc.Bots != nil {
		result["bots"] = fc.Bots
	}
	
	return result
}
