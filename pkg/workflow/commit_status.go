package workflow

import (
	"fmt"
)

// CommitStatusConfig holds configuration for commit status updates from agent output
type CommitStatusConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Context              string `yaml:"context,omitempty"`      // Status context/label (defaults to "agentic-workflow")
	TargetRepoSlug       string `yaml:"target-repo,omitempty"`  // Target repository for cross-repo status updates
}

// buildCommitStatusJob creates the commit_status job
func (c *Compiler) buildCommitStatusJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CommitStatus == nil {
		return nil, fmt.Errorf("safe-outputs.commit-status configuration is required")
	}

	// Build custom environment variables specific to commit-status
	var customEnvVars []string
	
	// Pass the commit SHA from activation job output or github context
	customEnvVars = append(customEnvVars, "          GH_AW_COMMIT_SHA: ${{ github.sha }}\n")

	// Pass the context configuration if provided
	if data.SafeOutputs.CommitStatus.Context != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMIT_STATUS_CONTEXT: %q\n", data.SafeOutputs.CommitStatus.Context))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CommitStatus.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.CommitStatus != nil {
		token = data.SafeOutputs.CommitStatus.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"status_id":  "${{ steps.commit_status.outputs.status_id }}",
		"status_url": "${{ steps.commit_status.outputs.status_url }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "commit_status",
		StepName:       "Update Commit Status",
		StepID:         "commit_status",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCommitStatusScript(),
		Permissions:    NewPermissionsContentsReadRepositoryWrite(),
		Outputs:        outputs,
		Token:          token,
		TargetRepoSlug: data.SafeOutputs.CommitStatus.TargetRepoSlug,
	})
}

// parseCommitStatusConfig handles commit-status configuration
func (c *Compiler) parseCommitStatusConfig(outputMap map[string]any) *CommitStatusConfig {
	if configData, exists := outputMap["commit-status"]; exists {
		commitStatusConfig := &CommitStatusConfig{}
		commitStatusConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &commitStatusConfig.BaseSafeOutputConfig)

			// Parse context
			if context, exists := configMap["context"]; exists {
				if contextStr, ok := context.(string); ok {
					commitStatusConfig.Context = contextStr
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			commitStatusConfig.TargetRepoSlug = targetRepoSlug
		}

		return commitStatusConfig
	}

	return nil
}
