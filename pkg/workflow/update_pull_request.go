package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var updatePullRequestLog = logger.New("workflow:update_pull_request")

// UpdatePullRequestsConfig holds configuration for updating GitHub pull requests from agent output
type UpdatePullRequestsConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Title              *bool `yaml:"title,omitempty"` // Allow updating PR title - defaults to true, set to false to disable
	Body               *bool `yaml:"body,omitempty"`  // Allow updating PR body - defaults to true, set to false to disable
}

// buildCreateOutputUpdatePullRequestJob creates the update_pull_request job
func (c *Compiler) buildCreateOutputUpdatePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	updatePullRequestLog.Printf("Building update pull request job: workflow=%s, mainJob=%s", data.Name, mainJobName)
	if data.SafeOutputs == nil || data.SafeOutputs.UpdatePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.update-pull-request configuration is required")
	}

	cfg := data.SafeOutputs.UpdatePullRequests

	return c.buildStandardUpdateEntityJob(
		data,
		mainJobName,
		&cfg.UpdateEntityConfig,
		UpdateEntityPullRequest,
		"update-pull-request",
		"update_pull_request",
		"Update Pull Request",
		getUpdatePullRequestScript,
		NewPermissionsContentsReadPRWrite,
		func(config *UpdateEntityConfig) []string {
			// Default to true for both title and body unless explicitly set to false
			canUpdateTitle := cfg.Title == nil || *cfg.Title
			canUpdateBody := cfg.Body == nil || *cfg.Body
			return []string{
				fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", canUpdateTitle),
				fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", canUpdateBody),
			}
		},
		func() map[string]string {
			return map[string]string{
				"pull_request_number": "${{ steps.update_pull_request.outputs.pull_request_number }}",
				"pull_request_url":    "${{ steps.update_pull_request.outputs.pull_request_url }}",
			}
		},
		func(target string) ConditionNode {
			return BuildPropertyAccess("github.event.pull_request.number")
		},
		updatePullRequestLog,
	)
}

// parseUpdatePullRequestsConfig handles update-pull-request configuration
func (c *Compiler) parseUpdatePullRequestsConfig(outputMap map[string]any) *UpdatePullRequestsConfig {
	updatePullRequestLog.Print("Parsing update pull request configuration")
	params := UpdateEntityJobParams{
		EntityType: UpdateEntityPullRequest,
		ConfigKey:  "update-pull-request",
	}

	parseSpecificFields := func(configMap map[string]any, baseConfig *UpdateEntityConfig) {
		// This will be called during parsing to handle PR-specific fields
		// The actual UpdatePullRequestsConfig fields are handled separately since they're not in baseConfig
	}

	baseConfig := c.parseUpdateEntityConfig(outputMap, params, updatePullRequestLog, parseSpecificFields)
	if baseConfig == nil {
		return nil
	}

	// Create UpdatePullRequestsConfig and populate it
	updatePullRequestsConfig := &UpdatePullRequestsConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse PR-specific fields
	if configData, exists := outputMap["update-pull-request"]; exists {
		if configMap, ok := configData.(map[string]any); ok {
			// Parse title - boolean to enable/disable (defaults to true if nil or not set)
			if titleVal, exists := configMap["title"]; exists {
				if titleBool, ok := titleVal.(bool); ok {
					updatePullRequestsConfig.Title = &titleBool
				}
				// If present but not a bool (e.g., null), leave as nil (defaults to enabled)
			}

			// Parse body - boolean to enable/disable (defaults to true if nil or not set)
			if bodyVal, exists := configMap["body"]; exists {
				if bodyBool, ok := bodyVal.(bool); ok {
					updatePullRequestsConfig.Body = &bodyBool
				}
				// If present but not a bool (e.g., null), leave as nil (defaults to enabled)
			}
		}
	}

	return updatePullRequestsConfig
}
