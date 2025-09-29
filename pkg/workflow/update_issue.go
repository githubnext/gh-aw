package workflow

import (
	"fmt"
)

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Status               *bool  `yaml:"status,omitempty"` // Allow updating issue status (open/closed) - presence indicates field can be updated
	Target               string `yaml:"target,omitempty"` // Target for updates: "triggering" (default), "*" (any issue), or explicit issue number
	Title                *bool  `yaml:"title,omitempty"`  // Allow updating issue title - presence indicates field can be updated
	Body                 *bool  `yaml:"body,omitempty"`   // Allow updating issue body - presence indicates field can be updated
}

// buildCreateOutputUpdateIssueJob creates the update_issue job
func (c *Compiler) buildCreateOutputUpdateIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.update-issue configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Update Issue\n")
	steps = append(steps, "        id: update_issue\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	// Pass the configuration flags
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_STATUS: %t\n", data.SafeOutputs.UpdateIssues.Status != nil))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_TITLE: %t\n", data.SafeOutputs.UpdateIssues.Title != nil))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_BODY: %t\n", data.SafeOutputs.UpdateIssues.Body != nil))

	// Pass the target configuration
	if data.SafeOutputs.UpdateIssues.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdateIssues.Target))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.UpdateIssues != nil {
		token = data.SafeOutputs.UpdateIssues.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(updateIssueScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
	}

	// Determine the job condition based on target configuration
	var baseCondition string
	if data.SafeOutputs.UpdateIssues.Target == "*" {
		// Allow updates to any issue - no specific context required
		baseCondition = "always()"
	} else if data.SafeOutputs.UpdateIssues.Target != "" {
		// Explicit issue number specified - no specific context required
		baseCondition = "always()"
	} else {
		// Default behavior: only update triggering issue
		baseCondition = "github.event.issue.number"
	}

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		if baseCondition == "always()" {
			// If base condition is always(), just use the command condition
			jobCondition = commandConditionStr
		} else {
			// Combine both conditions with AND
			jobCondition = fmt.Sprintf("(%s) && (%s)", commandConditionStr, baseCondition)
		}
	} else {
		// No command trigger, just use the base condition
		jobCondition = baseCondition
	}

	job := &Job{
		Name:           "update_issue",
		If:             jobCondition,
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      issues: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseUpdateIssuesConfig handles update-issue configuration
func (c *Compiler) parseUpdateIssuesConfig(outputMap map[string]any) *UpdateIssuesConfig {
	if configData, exists := outputMap["update-issue"]; exists {
		updateIssuesConfig := &UpdateIssuesConfig{}
		updateIssuesConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &updateIssuesConfig.BaseSafeOutputConfig)

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					updateIssuesConfig.Target = targetStr
				}
			}

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

		return updateIssuesConfig
	}

	return nil
}
