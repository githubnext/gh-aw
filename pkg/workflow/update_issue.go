package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateIssueLog = logger.New("workflow:update_issue")

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Status             *bool `yaml:"status,omitempty"` // Allow updating issue status (open/closed) - presence indicates field can be updated
	Title              *bool `yaml:"title,omitempty"`  // Allow updating issue title - presence indicates field can be updated
	Body               *bool `yaml:"body,omitempty"`   // Allow updating issue body - presence indicates field can be updated
}

// buildCreateOutputUpdateIssueJob creates the update_issue job
func (c *Compiler) buildCreateOutputUpdateIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.update-issue configuration is required")
	}

	cfg := data.SafeOutputs.UpdateIssues

	builder := UpdateEntityJobBuilder{
		EntityType:      UpdateEntityIssue,
		ConfigKey:       "update-issue",
		JobName:         "update_issue",
		StepName:        "Update Issue",
		ScriptGetter:    getUpdateIssueScript,
		PermissionsFunc: NewPermissionsContentsReadIssuesWrite,
		BuildCustomEnvVars: func(config *UpdateEntityConfig) []string {
			return []string{
				fmt.Sprintf("          GH_AW_UPDATE_STATUS: %t\n", cfg.Status != nil),
				fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", cfg.Title != nil),
				fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", cfg.Body != nil),
			}
		},
		BuildOutputs: func() map[string]string {
			return map[string]string{
				"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
				"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
			}
		},
		BuildEventCondition: func(target string) ConditionNode {
			return BuildPropertyAccess("github.event.issue.number")
		},
	}

	return c.buildUpdateEntityJobWithConfig(data, mainJobName, &cfg.UpdateEntityConfig, builder, updateIssueLog)
}

// parseUpdateIssuesConfig handles update-issue configuration
func (c *Compiler) parseUpdateIssuesConfig(outputMap map[string]any) *UpdateIssuesConfig {
	params := UpdateEntityJobParams{
		EntityType: UpdateEntityIssue,
		ConfigKey:  "update-issue",
	}

	parseSpecificFields := func(configMap map[string]any, baseConfig *UpdateEntityConfig) {
		// This will be called during parsing to handle issue-specific fields
		// The actual UpdateIssuesConfig fields are handled separately since they're not in baseConfig
	}

	baseConfig := c.parseUpdateEntityConfig(outputMap, params, updateIssueLog, parseSpecificFields)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateIssuesConfig and populate it
	updateIssuesConfig := &UpdateIssuesConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse issue-specific fields
	if configData, exists := outputMap["update-issue"]; exists {
		if configMap, ok := configData.(map[string]any); ok {
			// Parse status - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["status"]; exists {
				// If the key exists, it means we can update the status
				// We don't care about the value - just that the key is present
				updateIssuesConfig.Status = new(bool) // Allocate a new bool pointer (defaults to false)
			}

			// Parse title - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["title"]; exists {
				updateIssuesConfig.Title = new(bool)
			}

			// Parse body - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["body"]; exists {
				updateIssuesConfig.Body = new(bool)
			}
		}
	}

	return updateIssuesConfig
}
