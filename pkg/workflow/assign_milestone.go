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

// buildAssignMilestoneJob creates the assign_milestone job
func (c *Compiler) buildAssignMilestoneJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignMilestone == nil {
		return nil, fmt.Errorf("safe-outputs.assign-milestone configuration is required")
	}

	cfg := data.SafeOutputs.AssignMilestone

	// Handle max count with default of 1
	maxCount := 1
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables using shared helpers
	listJobConfig := ListJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		Allowed:                cfg.Allowed,
	}
	customEnvVars := BuildListJobEnvVars("GH_AW_MILESTONE", listJobConfig, maxCount)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"assigned_milestones": "${{ steps.assign_milestone.outputs.assigned_milestones }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "assign_milestone",
		StepName:       "Assign Milestone",
		StepID:         "assign_milestone",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAssignMilestoneScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Token:          cfg.GitHubToken,
		Condition:      BuildSafeOutputType("assign_milestone"),
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
