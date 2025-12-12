package workflow

import (
	"fmt"
)

// AssignMilestoneConfig holds configuration for assigning milestones to issues from agent output
type AssignMilestoneConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed milestone titles or IDs
}

// parseAssignMilestoneConfig handles assign-milestone configuration
func (c *Compiler) parseAssignMilestoneConfig(outputMap map[string]any) *AssignMilestoneConfig {
	if milestone, exists := outputMap["assign-milestone"]; exists {
		if milestoneMap, ok := milestone.(map[string]any); ok {
			milestoneConfig := &AssignMilestoneConfig{}

			// Parse list job config (target, target-repo, allowed)
			listJobConfig, _ := ParseListJobConfig(milestoneMap, "allowed")
			milestoneConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
			milestoneConfig.Allowed = listJobConfig.Allowed

			// Parse common base fields (github-token, max)
			c.parseBaseSafeOutputConfig(milestoneMap, &milestoneConfig.BaseSafeOutputConfig, 0)

			return milestoneConfig
		} else if milestone == nil {
			// Handle null case: create empty config (allows any milestones)
			return &AssignMilestoneConfig{}
		}
	}

	return nil
}

// buildAssignMilestoneJob creates the assign_milestone job
func (c *Compiler) buildAssignMilestoneJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignMilestone == nil {
		return nil, fmt.Errorf("safe-outputs.assign-milestone configuration is required")
	}

	cfg := data.SafeOutputs.AssignMilestone

	// Build list job config
	listJobConfig := ListJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		Allowed:                cfg.Allowed,
	}

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
