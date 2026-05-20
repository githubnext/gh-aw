package workflow

import "github.com/github/gh-aw/pkg/logger"

var setIssueFieldLog = logger.New("workflow:set_issue_field")

// SetIssueFieldConfig holds configuration for setting a single issue field from agent output.
type SetIssueFieldConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
	AllowedFields          []string `yaml:"allowed-fields,omitempty"` // Optional list of allowed issue field names. If omitted or empty, any field is allowed. Use ["*"] to explicitly allow all.
}

// parseSetIssueFieldConfig handles set-issue-field configuration.
func (c *Compiler) parseSetIssueFieldConfig(outputMap map[string]any) *SetIssueFieldConfig {
	config := parseConfigScaffold(outputMap, "set-issue-field", setIssueFieldLog, func(err error) *SetIssueFieldConfig {
		setIssueFieldLog.Printf("Failed to unmarshal set-issue-field config, disabling handler: %v", err)
		return nil
	})
	if config != nil {
		setIssueFieldLog.Printf("Parsed configuration: target=%s", config.Target)
	}
	return config
}
