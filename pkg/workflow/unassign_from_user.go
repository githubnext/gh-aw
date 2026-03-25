package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var unassignFromUserLog = logger.New("workflow:unassign_from_user")

// UnassignFromUserConfig holds configuration for removing assignees from issues
type UnassignFromUserConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed usernames. If omitted, any users can be unassigned.
	Blocked                []string `yaml:"blocked,omitempty"` // Optional list of blocked usernames or patterns (e.g., "copilot", "*[bot]")
}

// parseUnassignFromUserConfig handles unassign-from-user configuration
func (c *Compiler) parseUnassignFromUserConfig(outputMap map[string]any) *UnassignFromUserConfig {
	config := parseConfigScaffold(outputMap, "unassign-from-user", unassignFromUserLog, func(err error) *UnassignFromUserConfig {
		unassignFromUserLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, use defaults
		unassignFromUserLog.Print("Using default configuration")
		return &UnassignFromUserConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{Max: defaultIntStr(1)},
		}
	})
	if config == nil {
		return nil
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(1)
	}

	unassignFromUserLog.Printf("Parsed configuration: allowed_count=%d, target=%s", len(config.Allowed), config.Target)

	return config
}
