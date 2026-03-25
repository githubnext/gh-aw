package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var removeLabelsLog = logger.New("workflow:remove_labels")

// RemoveLabelsConfig holds configuration for removing labels from issues/PRs from agent output
type RemoveLabelsConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed labels to remove. If omitted, any labels can be removed.
	Blocked                []string `yaml:"blocked,omitempty"` // Optional list of blocked label patterns (supports glob patterns like "~*", "*[bot]"). Labels matching these patterns will be rejected.
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
