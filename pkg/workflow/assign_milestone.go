package workflow

import (
	"fmt"
)

// AssignMilestoneConfig holds configuration for assigning issues to milestones from agent output
type AssignMilestoneConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Allowed              []string `yaml:"allowed,omitempty"`     // Mandatory list of allowed milestone names or IDs
	Target               string   `yaml:"target,omitempty"`      // Target for milestones: "triggering" (default), "*" (any issue), or explicit issue number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository operations
}

// buildAssignMilestoneJob creates the assign_milestone job
func (c *Compiler) buildAssignMilestoneJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignMilestone == nil {
		return nil, fmt.Errorf("safe-outputs.assign-milestone configuration is required")
	}

	// Validate that allowed milestones are configured
	if len(data.SafeOutputs.AssignMilestone.Allowed) == 0 {
		return nil, fmt.Errorf("safe-outputs.assign-milestone.allowed must be configured with at least one milestone")
	}

	// Build custom environment variables specific to assign-milestone
	var customEnvVars []string

	// Pass the allowed milestones list as comma-separated string
	allowedMilestonesStr := ""
	for i, milestone := range data.SafeOutputs.AssignMilestone.Allowed {
		if i > 0 {
			allowedMilestonesStr += ","
		}
		allowedMilestonesStr += milestone
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONES_ALLOWED: %q\n", allowedMilestonesStr))

	// Pass the target configuration
	if data.SafeOutputs.AssignMilestone.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONE_TARGET: %q\n", data.SafeOutputs.AssignMilestone.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.AssignMilestone.TargetRepoSlug)...)

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Add Milestone",
		StepID:        "assign_milestone",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getAssignMilestoneScript(),
		Token:         data.SafeOutputs.AssignMilestone.GitHubToken,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"milestone_added": "${{ steps.assign_milestone.outputs.milestone_added }}",
		"issue_number":    "${{ steps.assign_milestone.outputs.issue_number }}",
	}

	// Build job condition
	var jobCondition = BuildSafeOutputType("assign_milestone")
	if data.SafeOutputs.AssignMilestone.Target == "" {
		// If target is not specified or is "triggering", require issue context
		eventCondition := BuildPropertyAccess("github.event.issue.number")
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "assign_milestone",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadIssuesWrite().RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
