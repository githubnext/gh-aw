package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
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
	assignMilestoneLog.Print("Parsing assign-milestone configuration")

	if milestone, exists := outputMap["assign-milestone"]; exists {
		if milestoneMap, ok := milestone.(map[string]any); ok {
			milestoneConfig := &AssignMilestoneConfig{}

			// Parse list job config (target, target-repo, allowed)
			listJobConfig, _ := ParseListJobConfig(milestoneMap, "allowed")
			milestoneConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
			milestoneConfig.Allowed = listJobConfig.Allowed
			assignMilestoneLog.Printf("Parsed milestone config: target=%s, allowed_count=%d",
				milestoneConfig.Target, len(milestoneConfig.Allowed))

			// Parse common base fields (github-token, max)
			c.parseBaseSafeOutputConfig(milestoneMap, &milestoneConfig.BaseSafeOutputConfig, 0)

			return milestoneConfig
		} else if milestone == nil {
			// Handle null case: create empty config (allows any milestones)
			assignMilestoneLog.Print("Null milestone config, allowing any milestones")
			return &AssignMilestoneConfig{}
		}
	}

	assignMilestoneLog.Print("No assign-milestone configuration found")
	return nil
}
