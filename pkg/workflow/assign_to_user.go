package workflow

import (
	"fmt"
)

// AssignToUserConfig holds configuration for assigning users to issues from agent output
type AssignToUserConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string   `yaml:"target,omitempty"`      // Target for user assignment: "triggering" (default) or "*" for any issue
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository assignments
	Allowed              []string `yaml:"allowed,omitempty"`     // List of allowed usernames that can be assigned
}

// buildAssignToUserJob creates the assign_to_user job
func (c *Compiler) buildAssignToUserJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignToUser == nil {
		return nil, fmt.Errorf("safe-outputs.assign-to-user configuration is required")
	}

	maxCount := 1
	if data.SafeOutputs.AssignToUser.Max > 0 {
		maxCount = data.SafeOutputs.AssignToUser.Max
	}

	// Build custom environment variables specific to assign-to-user
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_USER_MAX_COUNT: %d\n", maxCount))

	// Pass allowed users if configured
	if len(data.SafeOutputs.AssignToUser.Allowed) > 0 {
		allowedJSON, err := toJSON(data.SafeOutputs.AssignToUser.Allowed)
		if err == nil {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ALLOWED_USERS: %s\n", singleQuote(allowedJSON)))
		}
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.AssignToUser.TargetRepoSlug)...)

	// Get token from config for step-level github-token
	var token string
	if data.SafeOutputs.AssignToUser != nil {
		token = data.SafeOutputs.AssignToUser.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"assigned_users": "${{ steps.assign_to_user.outputs.assigned_users }}",
	}

	// Use the shared builder function to create the job
	// User assignment only requires issues:write permission
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "assign_to_user",
		StepName:       "Assign to User",
		StepID:         "assign_to_user",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAssignToUserScript(),
		Permissions:    NewPermissionsIssuesWrite(),
		Outputs:        outputs,
		Token:          token,
		UseAgentToken:  false, // Regular user assignment doesn't need agent token
		Condition:      BuildSafeOutputType("assign_to_user"),
		TargetRepoSlug: data.SafeOutputs.AssignToUser.TargetRepoSlug,
	})
}
