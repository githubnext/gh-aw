package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var addLabelsLog = logger.New("workflow:add_labels")

// AddLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddLabelsConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed labels. Labels will be created if they don't already exist in the repository. If omitted, any labels are allowed (including creating new ones).
	Blocked                []string `yaml:"blocked,omitempty"` // Optional list of blocked label patterns (supports glob patterns like "~*", "*[bot]"). Labels matching these patterns will be rejected.
}

// parseAddLabelsConfig handles add-labels configuration
func (c *Compiler) parseAddLabelsConfig(outputMap map[string]any) *AddLabelsConfig {
	// Check if the key exists
	if _, exists := outputMap["add-labels"]; !exists {
		return nil
	}

	addLabelsLog.Print("Parsing add-labels configuration")

	// Unmarshal into typed config struct
	var config AddLabelsConfig
	if err := unmarshalConfig(outputMap, "add-labels", &config, addLabelsLog); err != nil {
		addLabelsLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config (allows any labels)
		addLabelsLog.Print("Using empty configuration (allows any labels)")
		return &AddLabelsConfig{}
	}

	addLabelsLog.Printf("Parsed configuration: allowed_count=%d, blocked_count=%d, target=%s", len(config.Allowed), len(config.Blocked), config.Target)

	return &config
}
