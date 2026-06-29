package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var removeLabelsLog = logger.New("workflow:remove_labels")

// RemoveLabelsConfig holds configuration for removing labels from issues/PRs from agent output
type RemoveLabelsConfig struct {
	BaseSafeOutputConfig       `yaml:",inline"`
	SafeOutputTargetConfig     `yaml:",inline"`
	SafeOutputFilterConfig     `yaml:",inline"`
	SafeOutputAllowBlockConfig `yaml:",inline"`
}

// parseRemoveLabelsConfig handles remove-labels configuration
func (c *Compiler) parseRemoveLabelsConfig(outputMap map[string]any) *RemoveLabelsConfig {
	config := parseConfigScaffold(outputMap, "remove-labels", removeLabelsLog, func(err error) *RemoveLabelsConfig {
		removeLabelsLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config (allows any labels)
		removeLabelsLog.Print("Using empty configuration (allows any labels)")
		return &RemoveLabelsConfig{}
	})
	if config != nil {
		removeLabelsLog.Printf("Parsed configuration: allowed_count=%d, blocked_count=%d, target=%s", len(config.Allowed), len(config.Blocked), config.Target)
	}
	return config
}
