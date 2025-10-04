package workflow

import (
	"fmt"
	"strings"
)

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`
	Labels               []string `yaml:"labels,omitempty"`
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	if configData, exists := outputMap["create-issue"]; exists {
		issuesConfig := &CreateIssuesConfig{}
		issuesConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					issuesConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse labels
			if labels, exists := configMap["labels"]; exists {
				if labelsArray, ok := labels.([]any); ok {
					var labelStrings []string
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							labelStrings = append(labelStrings, labelStr)
						}
					}
					issuesConfig.Labels = labelStrings
				}
			}

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &issuesConfig.BaseSafeOutputConfig)
		}

		return issuesConfig
	}

	return nil
}

// buildCreateOutputIssueJob creates the create_issue job
func (c *Compiler) buildCreateOutputIssueJob(data *WorkflowData, mainJobName string, taskJobCreated bool, frontmatter map[string]any) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.create-issue configuration is required")
	}

	var steps []string

	// Add permission checks if no task job was created but permission checks are needed
	if !taskJobCreated && c.needsRoleCheck(data, frontmatter) {
		// Add team member check step
		steps = append(steps, "      - name: Check team membership for workflow\n")
		steps = append(steps, "        id: check-team-member\n")
		steps = append(steps, "        uses: actions/github-script@v8\n")

		// Add environment variables for permission check
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_REQUIRED_ROLES: %s\n", strings.Join(data.Roles, ",")))

		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Generate the JavaScript code for the permission check
		scriptContent := c.generateRoleCheckScript(data.Roles)
		scriptLines := strings.Split(scriptContent, "\n")
		for _, line := range scriptLines {
			if strings.TrimSpace(line) != "" {
				steps = append(steps, fmt.Sprintf("            %s\n", line))
			}
		}
	}

	// Build environment variables
	env := make(map[string]string)
	c.getCustomSafeOutputEnvVars(env, data, mainJobName, nil)

	if data.SafeOutputs.CreateIssues.TitlePrefix != "" {
		env["GITHUB_AW_ISSUE_TITLE_PREFIX"] = fmt.Sprintf("%q", data.SafeOutputs.CreateIssues.TitlePrefix)
	}
	if len(data.SafeOutputs.CreateIssues.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreateIssues.Labels, ",")
		env["GITHUB_AW_ISSUE_LABELS"] = fmt.Sprintf("%q", labelsStr)
	}

	// Build with parameters
	withParams := make(map[string]string)
	token := ""
	if data.SafeOutputs.CreateIssues != nil {
		token = data.SafeOutputs.CreateIssues.GitHubToken
	}
	c.populateGitHubTokenForSafeOutput(withParams, data, token)

	// Build github-script step
	stepLines := BuildGitHubScriptStepLines("Create Output Issue", "create_issue", createIssueScript, env, withParams)
	steps = append(steps, stepLines...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
	}

	jobCondition := BuildSafeOutputType("create-issue", data.SafeOutputs.CreateIssues.Min)

	// Set base permissions
	permissions := "permissions:\n      contents: read\n      issues: write"

	job := &Job{
		Name:           "create_issue",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions,
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
