package workflow

import (
	"fmt"
	"strings"
)

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	TitlePrefix string   `yaml:"title-prefix,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
	Max         int      `yaml:"max,omitempty"`          // Maximum number of issues to create
	GitHubToken string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	if configData, exists := outputMap["create-issue"]; exists {
		issuesConfig := &CreateIssuesConfig{Max: 1} // Default max is 1

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

			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := parseIntValue(max); ok {
					issuesConfig.Max = maxInt
				}
			}

			// Parse github-token
			if githubToken, exists := configMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					issuesConfig.GitHubToken = githubTokenStr
				}
			}
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

	steps = append(steps, "      - name: Create Output Issue\n")
	steps = append(steps, "        id: create_issue\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	if data.SafeOutputs.CreateIssues.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_ISSUE_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateIssues.TitlePrefix))
	}
	if len(data.SafeOutputs.CreateIssues.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreateIssues.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_ISSUE_LABELS: %q\n", labelsStr))
	}

	// Pass the staged flag if it's set to true
	if data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.CreateIssues != nil {
		token = data.SafeOutputs.CreateIssues.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createIssueScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
	}

	// Determine the job condition for command workflows
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = commandConditionStr
	} else {
		jobCondition = "" // No conditional execution
	}

	// Set base permissions
	permissions := "permissions:\n      contents: read\n      issues: write"

	job := &Job{
		Name:           "create_issue",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    permissions,
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
