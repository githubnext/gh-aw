package workflow

import (
	"fmt"
)

// buildUpdateProjectJob creates the update_project job
func (c *Compiler) buildUpdateProjectJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateProjects == nil {
		return nil, fmt.Errorf("safe-outputs.update-project configuration is required")
	}

	var steps []string

	// Build custom environment variables specific to update-project
	var customEnvVars []string

	// Add common safe output job environment variables (staged/target repo)
	// Note: Project operations always work on the current repo, so targetRepoSlug is ""
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		"", // targetRepoSlug - projects always work on current repo
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.UpdateProjects != nil {
		token = data.SafeOutputs.UpdateProjects.GitHubToken
	}

	// Build the GitHub Script step using the common helper and append to existing steps
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Update Project",
		StepID:        "update_project",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getUpdateProjectScript(),
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	jobCondition := BuildSafeOutputType("update_project")

	job := &Job{
		Name:           "update_project",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadProjectsWrite().RenderToYAML(),
		TimeoutMinutes: 10,
		Steps:          steps,
		Needs:          []string{mainJobName},
	}

	return job, nil
}
