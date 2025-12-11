package workflow

import (
	"fmt"
)

// buildCreateProjectJob creates the create_project job
func (c *Compiler) buildCreateProjectJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateProjects == nil {
		return nil, fmt.Errorf("safe-outputs.create-project configuration is required")
	}

	var steps []string

	// Build custom environment variables specific to create-project
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
	if data.SafeOutputs.CreateProjects != nil {
		token = data.SafeOutputs.CreateProjects.GitHubToken
	}

	// Build the GitHub Script step using the common helper and append to existing steps
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Create Project",
		StepID:        "create_project",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getCreateProjectScript(),
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	jobCondition := BuildSafeOutputType("create_project")

	job := &Job{
		Name:           "create_project",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadProjectsWrite().RenderToYAML(),
		TimeoutMinutes: 10,
		Steps:          steps,
		Needs:          []string{mainJobName},
	}

	return job, nil
}
