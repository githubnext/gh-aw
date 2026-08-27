package workflow

import (
	"errors"

	"github.com/github/gh-aw/pkg/logger"
)

var removeLabelsLog = logger.New("workflow:remove_labels")

// RemoveLabelsConfig holds configuration for removing labels from issues/PRs from agent output
type RemoveLabelsConfig struct {
	BaseSafeOutputConfig       `yaml:",inline"`
	SafeOutputTargetConfig     `yaml:",inline"`
	SafeOutputFilterConfig     `yaml:",inline"`
	SafeOutputAllowBlockConfig `yaml:",inline"`
	Issues                     *bool `yaml:"issues,omitempty"`        // When false, excludes issues:write permission. Default (nil or true) includes issues:write.
	PullRequests               *bool `yaml:"pull-requests,omitempty"` // When false, excludes pull-requests:write permission. Default (nil or true) includes pull-requests:write.
}

// parseRemoveLabelsConfig handles remove-labels configuration
func (c *Compiler) parseRemoveLabelsConfig(outputMap map[string]any) *RemoveLabelsConfig {
	return parseConfigScaffoldWithPostProcess(outputMap, "remove-labels", removeLabelsLog,
		func(err error) *RemoveLabelsConfig {
			removeLabelsLog.Printf("Failed to unmarshal config: %v", err)
			removeLabelsLog.Print("Using fail-closed fallback configuration due to parse error")
			issuesDisabled := false
			pullRequestsDisabled := false
			return &RemoveLabelsConfig{
				Issues:       &issuesDisabled,
				PullRequests: &pullRequestsDisabled,
			}
		},
		func(config *RemoveLabelsConfig) {
			removeLabelsLog.Printf("Parsed configuration: allowed_count=%d, blocked_count=%d, target=%s", len(config.Allowed), len(config.Blocked), config.Target)
		})
}

// buildRemoveLabelsPermissions computes the permissions for remove_labels based on config.
func buildRemoveLabelsPermissions(config *RemoveLabelsConfig) *Permissions {
	permMap := map[PermissionScope]PermissionLevel{}
	if config == nil || config.Issues == nil || *config.Issues {
		permMap[PermissionIssues] = PermissionWrite
	}
	if config == nil || config.PullRequests == nil || *config.PullRequests {
		permMap[PermissionPullRequests] = PermissionWrite
	}
	return NewPermissionsFromMap(permMap)
}

func validateRemoveLabelsPermissions(config *SafeOutputsConfig) error {
	if config == nil || config.RemoveLabels == nil {
		return nil
	}
	c := config.RemoveLabels
	if c.Issues != nil && !*c.Issues && c.PullRequests != nil && !*c.PullRequests {
		return errors.New("safe-outputs.remove-labels: at least one of 'issues' or 'pull-requests' must be enabled")
	}
	return nil
}
