package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var replaceLabelLog = logger.New("workflow:replace_label")

// LabelTransition represents an allowed label state transition.
// When allowed-transitions is configured, only listed (from → to) pairs are permitted.
type LabelTransition struct {
	From string `yaml:"from"` // Label that must be present and will be removed
	To   string `yaml:"to"`   // Label that will be added
}

// ReplaceLabelConfig holds configuration for replacing one label with another on issues/PRs from agent output.
// It combines the capabilities of add-labels and remove-labels into a single GraphQL request,
// enabling clear state transitions (e.g. "in-progress" → "done").
type ReplaceLabelConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
	AllowedAdd             []string          `yaml:"allowed-add,omitempty"`         // Optional list of allowed label patterns that can be added (supports glob patterns like "state-*"). If omitted, any labels are allowed.
	AllowedRemove          []string          `yaml:"allowed-remove,omitempty"`      // Optional list of allowed label patterns that can be removed (supports glob patterns like "state-*"). If omitted, any labels can be removed.
	Blocked                []string          `yaml:"blocked,omitempty"`             // Optional list of blocked label patterns (supports glob patterns like "~*", "*[bot]"). Applied to both add and remove labels.
	AllowedTransitions     []LabelTransition `yaml:"allowed-transitions,omitempty"` // Optional list of allowed (from → to) label transitions. When specified, only these exact pairs are permitted regardless of allowed-add/allowed-remove.
}

// parseReplaceLabelConfig handles replace-label configuration
func (c *Compiler) parseReplaceLabelConfig(outputMap map[string]any) *ReplaceLabelConfig {
	config := parseConfigScaffold(outputMap, "replace-label", replaceLabelLog, func(err error) *ReplaceLabelConfig {
		replaceLabelLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config (allows any labels)
		replaceLabelLog.Print("Using empty configuration (allows any labels)")
		return &ReplaceLabelConfig{}
	})
	if config != nil {
		replaceLabelLog.Printf("Parsed configuration: allowed_add_count=%d, allowed_remove_count=%d, blocked_count=%d, allowed_transitions_count=%d, target=%s",
			len(config.AllowedAdd), len(config.AllowedRemove), len(config.Blocked), len(config.AllowedTransitions), config.Target)
	}
	return config
}
