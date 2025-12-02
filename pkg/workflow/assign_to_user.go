package workflow

import (
	"fmt"
)

// AssignToUserConfig holds configuration for assigning users to issues from agent output
type AssignToUserConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // List of allowed usernames that can be assigned
}

// buildAssignToUserJob creates the assign_to_user job
func (c *Compiler) buildAssignToUserJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignToUser == nil {
		return nil, fmt.Errorf("safe-outputs.assign-to-user configuration is required")
	}

	cfg := data.SafeOutputs.AssignToUser

	maxCount := 1
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables specific to assign-to-user
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_USER_MAX_COUNT", maxCount)...)

	// Pass allowed users if configured
	customEnvVars = append(customEnvVars, BuildAllowedListEnvVar("GH_AW_ALLOWED_USERS", cfg.Allowed)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"assigned_users": "${{ steps.assign_to_user.outputs.assigned_users }}",
	}

	// Use the shared builder function to create the job
	// User assignment requires contents:read and issues:write permissions
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "assign_to_user",
		StepName:       "Assign to User",
		StepID:         "assign_to_user",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAssignToUserScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Token:          cfg.GitHubToken,
		UseAgentToken:  false, // Regular user assignment doesn't need agent token
		Condition:      BuildSafeOutputType("assign_to_user"),
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
