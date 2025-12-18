package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var assignToUserLog = logger.New("workflow:assign_to_user")

// AssignToUserConfig holds configuration for assigning users to issues from agent output
type AssignToUserConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed usernames. If omitted, any users are allowed.
}

// parseAssignToUserConfig handles assign-to-user configuration
func (c *Compiler) parseAssignToUserConfig(outputMap map[string]any) *AssignToUserConfig {
	if configData, exists := outputMap["assign-to-user"]; exists {
		assignToUserLog.Print("Parsing assign-to-user configuration")
		assignToUserConfig := &AssignToUserConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse list job config (target, target-repo, allowed)
			listJobConfig, _ := ParseListJobConfig(configMap, "allowed")
			assignToUserConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
			assignToUserConfig.Allowed = listJobConfig.Allowed

			// Parse common base fields (github-token, max) with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &assignToUserConfig.BaseSafeOutputConfig, 1)
			assignToUserLog.Printf("Parsed configuration: allowed_count=%d, target=%s", len(assignToUserConfig.Allowed), assignToUserConfig.Target)
		} else {
			// If configData is nil or not a map (e.g., "assign-to-user:" with no value),
			// use defaults
			assignToUserLog.Print("Using default configuration")
			assignToUserConfig.Max = 1
		}

		return assignToUserConfig
	}

	return nil
}
