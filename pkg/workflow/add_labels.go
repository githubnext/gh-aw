package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var addLabelsLog = logger.New("workflow:add_labels")

// AddLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddLabelsConfig struct {
	BaseSafeOutputConfig       `yaml:",inline"`
	SafeOutputTargetConfig     `yaml:",inline"`
	SafeOutputFilterConfig     `yaml:",inline"`
	SafeOutputAllowBlockConfig `yaml:",inline"`
	Issues                     *bool `yaml:"issues,omitempty"`        // When false, excludes issues:write permission. Default (nil or true) includes issues:write.
	PullRequests               *bool `yaml:"pull-requests,omitempty"` // When false, excludes pull-requests:write permission. Default (nil or true) includes pull-requests:write.
}

// parseAddLabelsConfig handles add-labels configuration
func (c *Compiler) parseAddLabelsConfig(outputMap map[string]any) *AddLabelsConfig {
	config := parseConfigScaffold(outputMap, "add-labels", addLabelsLog, func(err error) *AddLabelsConfig {
		addLabelsLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config (allows any labels)
		addLabelsLog.Print("Using empty configuration (allows any labels)")
		return &AddLabelsConfig{}
	})
	if config != nil {
		addLabelsLog.Printf("Parsed configuration: allowed_count=%d, blocked_count=%d, target=%s", len(config.Allowed), len(config.Blocked), config.Target)
	}
	return config
}

// buildAddLabelsPermissions computes the permissions for add_labels based on config.
// Issues: nil or true → issues:write (default: true)
// PullRequests: nil or true → pull-requests:write (default: true)
func buildAddLabelsPermissions(config *AddLabelsConfig) *Permissions {
	permMap := map[PermissionScope]PermissionLevel{}
	if config == nil || config.Issues == nil || *config.Issues {
		permMap[PermissionIssues] = PermissionWrite
	}
	if config == nil || config.PullRequests == nil || *config.PullRequests {
		permMap[PermissionPullRequests] = PermissionWrite
	}
	return NewPermissionsFromMap(permMap)
}
