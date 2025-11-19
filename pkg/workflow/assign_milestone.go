package workflow

import (
	"fmt"
)

// AssignMilestoneConfig holds configuration for assigning milestones to issues from agent output
type AssignMilestoneConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Allowed              []string `yaml:"allowed,omitempty"`      // Optional list of allowed milestone titles or IDs
	Target               string   `yaml:"target,omitempty"`       // Target for milestone assignment: "triggering" (default) or explicit issue number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`  // Target repository in format "owner/repo" for cross-repository assignments
}

// buildAssignMilestoneJob creates the assign_milestone job
func (c *Compiler) buildAssignMilestoneJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignMilestone == nil {
		return nil, fmt.Errorf("safe-outputs.assign-milestone configuration is required")
	}

	// Handle case where AssignMilestone is not nil
	var allowedMilestones []string
	maxCount := 1

	allowedMilestones = data.SafeOutputs.AssignMilestone.Allowed
	if data.SafeOutputs.AssignMilestone.Max > 0 {
		maxCount = data.SafeOutputs.AssignMilestone.Max
	}

	// Build custom environment variables specific to assign-milestone
	var customEnvVars []string
	
	// Pass the allowed milestones list (empty string if no restrictions)
	if len(allowedMilestones) > 0 {
		allowedMilestonesStr := ""
		for i, milestone := range allowedMilestones {
			if i > 0 {
				allowedMilestonesStr += ","
			}
			allowedMilestonesStr += milestone
		}
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONE_ALLOWED: %q\n", allowedMilestonesStr))
	}
	
	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONE_MAX_COUNT: %d\n", maxCount))

	// Pass the target configuration
	if data.SafeOutputs.AssignMilestone.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONE_TARGET: %q\n", data.SafeOutputs.AssignMilestone.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.AssignMilestone.TargetRepoSlug)...)

	// Get token from config
	var token string
	if data.SafeOutputs.AssignMilestone != nil {
		token = data.SafeOutputs.AssignMilestone.GitHubToken
	}

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
		Token:          token,
		Condition:      BuildSafeOutputType("assign_milestone"),
		TargetRepoSlug: data.SafeOutputs.AssignMilestone.TargetRepoSlug,
	})
}
