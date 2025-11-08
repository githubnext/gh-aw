package workflow

import (
	"fmt"
)

// CreateCommitStatusConfig holds configuration for creating commit statuses from agent output
type CreateCommitStatusConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Context              string `yaml:"context,omitempty"` // Context string to differentiate status (default: workflow name)
}

// parseCommitStatusConfig handles create-commit-status configuration
func (c *Compiler) parseCommitStatusConfig(outputMap map[string]any) *CreateCommitStatusConfig {
	if _, exists := outputMap["create-commit-status"]; !exists {
		return nil
	}

	configData := outputMap["create-commit-status"]
	commitStatusConfig := &CreateCommitStatusConfig{}
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
	}

	return commitStatusConfig
}

// buildCreateCommitStatusJob creates the create_commit_status job
func (c *Compiler) buildCreateCommitStatusJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateCommitStatus == nil {
		return nil, fmt.Errorf("safe-outputs.create-commit-status configuration is required")
	}

	// Build custom environment variables specific to create-commit-status
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	
	// Pass the context configuration, defaulting to workflow name
	contextName := data.SafeOutputs.CreateCommitStatus.Context
	if contextName == "" {
		if data.FrontmatterName != "" {
			contextName = data.FrontmatterName
		} else {
			contextName = data.Name // fallback to H1 header name
		}
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMIT_STATUS_CONTEXT: %q\n", contextName))

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		"", // commit status doesn't support target-repo
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.CreateCommitStatus != nil {
		token = data.SafeOutputs.CreateCommitStatus.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"status_created": "${{ steps.create_commit_status.outputs.status_created }}",
		"status_url":     "${{ steps.create_commit_status.outputs.status_url }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "create_commit_status",
		StepName:      "Create Commit Status",
		StepID:        "create_commit_status",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getCreateCommitStatusScript(),
		Permissions:   NewPermissionsContentsReadStatusesWrite(),
		Outputs:       outputs,
		Token:         token,
	})
}
