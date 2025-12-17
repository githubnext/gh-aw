package workflow

import (
	"fmt"

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

// buildAssignMilestoneJob creates the assign_milestone job
func (c *Compiler) buildAssignMilestoneJob(data *WorkflowData, mainJobName string) (*Job, error) {
	assignMilestoneLog.Printf("Building assign-milestone job: mainJobName=%s", mainJobName)

	if data.SafeOutputs == nil || data.SafeOutputs.AssignMilestone == nil {
		assignMilestoneLog.Print("No assign-milestone configuration in safe-outputs")
		return nil, fmt.Errorf("safe-outputs.assign-milestone configuration is required")
	}

	cfg := data.SafeOutputs.AssignMilestone

	// Build list job config
	listJobConfig := ListJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		Allowed:                cfg.Allowed,
	}
	assignMilestoneLog.Printf("Built list job config: allowed_count=%d", len(listJobConfig.Allowed))

	// Use shared builder for list-based safe-output jobs
	return c.BuildListSafeOutputJob(data, mainJobName, listJobConfig, cfg.BaseSafeOutputConfig, ListJobBuilderConfig{
		JobName:     "assign_milestone",
		StepName:    "Assign Milestone",
		StepID:      "assign_milestone",
		EnvPrefix:   "GH_AW_MILESTONE",
		OutputName:  "assigned_milestones",
		Script:      getAssignMilestoneScript(),
		Permissions: NewPermissionsContentsReadIssuesWrite(),
		DefaultMax:  1,
	})
}
