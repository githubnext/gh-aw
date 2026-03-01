package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var setIssueTypeLog = logger.New("workflow:set_issue_type")

// SetIssueTypeConfig holds configuration for setting the type of an issue from agent output
type SetIssueTypeConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed issue type names. If omitted, any type is allowed (including clearing with "").
}

// parseSetIssueTypeConfig handles set-issue-type configuration
func (c *Compiler) parseSetIssueTypeConfig(outputMap map[string]any) *SetIssueTypeConfig {
	// Check if the key exists
	if _, exists := outputMap["set-issue-type"]; !exists {
		return nil
	}

	setIssueTypeLog.Print("Parsing set-issue-type configuration")

	// Unmarshal into typed config struct
	var config SetIssueTypeConfig
	if err := unmarshalConfig(outputMap, "set-issue-type", &config, setIssueTypeLog); err != nil {
		setIssueTypeLog.Printf("Failed to unmarshal set-issue-type config, disabling handler: %v", err)
		return nil
	}

	setIssueTypeLog.Printf("Parsed configuration: allowed_count=%d, target=%s", len(config.Allowed), config.Target)

	return &config
}
