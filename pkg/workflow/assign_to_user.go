package workflow

import (
	"fmt"
)

// AssignToUserConfig holds configuration for assigning users to issues from agent output
type AssignToUserConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed usernames. If omitted, any users are allowed.
}

// buildAssignToUserJob creates the assign_to_user job
func (c *Compiler) buildAssignToUserJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignToUser == nil {
		return nil, fmt.Errorf("safe-outputs.assign-to-user configuration is required")
	}

	cfg := data.SafeOutputs.AssignToUser

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
	customEnvVars := BuildListJobEnvVars("GH_AW_ASSIGNEES", listJobConfig, maxCount)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"assigned_users": "${{ steps.assign_to_user.outputs.assigned_users }}",
	}

	var jobCondition = BuildSafeOutputType("assign_to_user")
	if cfg.Target == "" {
		// Only run if in issue context when target is not specified
		issueCondition := BuildPropertyAccess("github.event.issue.number")
		jobCondition = buildAnd(jobCondition, issueCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "assign_to_user",
		StepName:       "Assign to User",
		StepID:         "assign_to_user",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAssignToUserScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// parseAssignToUserConfig handles assign-to-user configuration
func (c *Compiler) parseAssignToUserConfig(outputMap map[string]any) *AssignToUserConfig {
	if configData, exists := outputMap["assign-to-user"]; exists {
		assignToUserConfig := &AssignToUserConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse list job config (target, target-repo, allowed)
			listJobConfig, _ := ParseListJobConfig(configMap, "allowed")
			assignToUserConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
			assignToUserConfig.Allowed = listJobConfig.Allowed

			// Parse common base fields (github-token, max) with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &assignToUserConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "assign-to-user:" with no value),
			// use defaults
			assignToUserConfig.Max = 1
		}

		return assignToUserConfig
	}

	return nil
}
