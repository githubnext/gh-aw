package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var assignToUserLog = logger.New("workflow:assign_to_user")

// AssignToUserConfig holds configuration for assigning users to issues from agent output
type AssignToUserConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"`        // Optional list of allowed usernames. If omitted, any users are allowed.
	Blocked                []string `yaml:"blocked,omitempty"`        // Optional list of blocked usernames or patterns (e.g., "copilot", "*[bot]")
	UnassignFirst          *string  `yaml:"unassign-first,omitempty"` // If true, unassign all current assignees before assigning new ones
}

// parseAssignToUserConfig handles assign-to-user configuration
func (c *Compiler) parseAssignToUserConfig(outputMap map[string]any) *AssignToUserConfig {
	// Check if the key exists
	if _, exists := outputMap["assign-to-user"]; !exists {
		return nil
	}

	assignToUserLog.Print("Parsing assign-to-user configuration")

	// Get config data for pre-processing before YAML unmarshaling
	configData, _ := outputMap["assign-to-user"].(map[string]any)

	// Pre-process templatable bool fields
	if err := preprocessBoolFieldAsString(configData, "unassign-first", assignToUserLog); err != nil {
		assignToUserLog.Printf("Invalid unassign-first value: %v", err)
		return nil
	}

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", assignToUserLog); err != nil {
		assignToUserLog.Printf("Invalid max value: %v", err)
		return nil
	}

	// Unmarshal into typed config struct
	var config AssignToUserConfig
	if err := unmarshalConfig(outputMap, "assign-to-user", &config, assignToUserLog); err != nil {
		assignToUserLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, use defaults
		assignToUserLog.Print("Using default configuration")
		config = AssignToUserConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{Max: defaultIntStr(1)},
		}
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(1)
	}

	assignToUserLog.Printf("Parsed configuration: allowed_count=%d, target=%s", len(config.Allowed), config.Target)

	return &config
}
