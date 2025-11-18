package workflow

import (
	"fmt"
)

// AddMilestoneConfig holds configuration for adding issues to milestones from agent output
type AddMilestoneConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Allowed              []string `yaml:"allowed,omitempty"`     // Mandatory list of allowed milestone names or IDs
	Target               string   `yaml:"target,omitempty"`      // Target for milestones: "triggering" (default), "*" (any issue), or explicit issue number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository operations
}

// buildAddMilestoneJob creates the add_milestone job
func (c *Compiler) buildAddMilestoneJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddMilestone == nil {
		return nil, fmt.Errorf("safe-outputs.add-milestone configuration is required")
	}

	// Validate that allowed milestones are configured
	if len(data.SafeOutputs.AddMilestone.Allowed) == 0 {
		return nil, fmt.Errorf("safe-outputs.add-milestone.allowed must be configured with at least one milestone")
	}

	// Build custom environment variables specific to add-milestone
	var customEnvVars []string
	
	// Pass the allowed milestones list as comma-separated string
	allowedMilestonesStr := ""
	for i, milestone := range data.SafeOutputs.AddMilestone.Allowed {
		if i > 0 {
			allowedMilestonesStr += ","
		}
		allowedMilestonesStr += milestone
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONES_ALLOWED: %q\n", allowedMilestonesStr))

	// Pass the target configuration
	if data.SafeOutputs.AddMilestone.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MILESTONE_TARGET: %q\n", data.SafeOutputs.AddMilestone.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.AddMilestone.TargetRepoSlug)...)

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Add Milestone",
		StepID:        "add_milestone",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getAddMilestoneScript(),
		Token:         data.SafeOutputs.AddMilestone.GitHubToken,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"milestone_added": "${{ steps.add_milestone.outputs.milestone_added }}",
		"issue_number":    "${{ steps.add_milestone.outputs.issue_number }}",
	}

	// Build job condition
	var jobCondition = BuildSafeOutputType("add_milestone")
	if data.SafeOutputs.AddMilestone.Target == "" {
		// If target is not specified or is "triggering", require issue context
		eventCondition := BuildPropertyAccess("github.event.issue.number")
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "add_milestone",
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
