package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var assignMilestoneLog = logger.New("workflow:assign_milestone")

// AssignMilestoneConfig holds configuration for assigning milestones to issues from agent output
type AssignMilestoneConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed milestone titles or IDs
}

// parseAssignMilestoneConfig handles assign-milestone configuration
func (c *Compiler) parseAssignMilestoneConfig(outputMap map[string]any) *AssignMilestoneConfig {
	config := parseConfigScaffold(outputMap, "assign-milestone", assignMilestoneLog, func(err error) *AssignMilestoneConfig {
		assignMilestoneLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config (allows any milestones)
		assignMilestoneLog.Print("Null milestone config, allowing any milestones")
		return &AssignMilestoneConfig{}
	})
	if config != nil {
		assignMilestoneLog.Printf("Parsed milestone config: target=%s, allowed_count=%d",
			config.Target, len(config.Allowed))
	}
	return config
}
