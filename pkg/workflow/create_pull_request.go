package workflow

import (
	"fmt"
	"strings"
)

// CreatePullRequestsConfig holds configuration for creating GitHub pull requests from agent output
type CreatePullRequestsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`
	Labels               []string `yaml:"labels,omitempty"`
	Draft                *bool    `yaml:"draft,omitempty"`         // Pointer to distinguish between unset (nil) and explicitly false
	IfNoChanges          string   `yaml:"if-no-changes,omitempty"` // Behavior when no changes to push: "warn" (default), "error", or "ignore"
}

// buildCreateOutputPullRequestJob creates the create_pull_request job
func (c *Compiler) buildCreateOutputPullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.create-pull-request configuration is required")
	}

	var steps []string

	// Step 1: Download patch artifact
	steps = append(steps, "      - name: Download patch artifact\n")
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: aw.patch\n")
	steps = append(steps, "          path: /tmp/\n")

	// Step 2: Checkout repository
	steps = buildCheckoutRepository(steps, c)

	// Step 3: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Step 4: Create pull request
	steps = append(steps, "      - name: Create Pull Request\n")
	steps = append(steps, "        id: create_pull_request\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the workflow ID for branch naming
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_ID: %q\n", mainJobName))
	// Pass the workflow name for footer generation
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))
	// Pass the base branch from GitHub context
	steps = append(steps, "          GITHUB_AW_BASE_BRANCH: ${{ github.ref_name }}\n")
	if data.SafeOutputs.CreatePullRequests.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_TITLE_PREFIX: %q\n", data.SafeOutputs.CreatePullRequests.TitlePrefix))
	}
	if len(data.SafeOutputs.CreatePullRequests.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreatePullRequests.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_LABELS: %q\n", labelsStr))
	}
	// Pass draft setting - default to true for backwards compatibility
	draftValue := true // Default value
	if data.SafeOutputs.CreatePullRequests.Draft != nil {
		draftValue = *data.SafeOutputs.CreatePullRequests.Draft
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_DRAFT: %q\n", fmt.Sprintf("%t", draftValue)))

	// Pass the if-no-changes configuration
	ifNoChanges := data.SafeOutputs.CreatePullRequests.IfNoChanges
	if ifNoChanges == "" {
		ifNoChanges = "warn" // Default value
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_IF_NO_CHANGES: %q\n", ifNoChanges))

	// Pass the maximum patch size configuration
	maxPatchSize := 1024 // Default value
	if data.SafeOutputs != nil && data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))

	// Pass the staged flag if it's set to true
	if c.trialMode || data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}
	if c.trialMode && c.trialTargetRepoSlug != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", c.trialTargetRepoSlug))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	c.addSafeOutputGitHubTokenForConfig(&steps, data, data.SafeOutputs.CreatePullRequests.GitHubToken)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createPullRequestScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.create_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.create_pull_request.outputs.pull_request_url }}",
		"issue_number":        "${{ steps.create_pull_request.outputs.issue_number }}",
		"issue_url":           "${{ steps.create_pull_request.outputs.issue_url }}",
		"branch_name":         "${{ steps.create_pull_request.outputs.branch_name }}",
		"fallback_used":       "${{ steps.create_pull_request.outputs.fallback_used }}",
	}

	jobCondition := BuildSafeOutputType("create-pull-request", data.SafeOutputs.CreatePullRequests.Min)

	job := &Job{
		Name:           "create_pull_request",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: write\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parsePullRequestsConfig handles only create-pull-request (singular) configuration
func (c *Compiler) parsePullRequestsConfig(outputMap map[string]any) *CreatePullRequestsConfig {
	// Check for singular form only
	if _, exists := outputMap["create-pull-request"]; !exists {
		return nil
	}

	configData := outputMap["create-pull-request"]
	pullRequestsConfig := &CreatePullRequestsConfig{}
	pullRequestsConfig.Max = 1 // Always max 1 for pull requests

	if configMap, ok := configData.(map[string]any); ok {
		// Parse title-prefix
		if titlePrefix, exists := configMap["title-prefix"]; exists {
			if titlePrefixStr, ok := titlePrefix.(string); ok {
				pullRequestsConfig.TitlePrefix = titlePrefixStr
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
				pullRequestsConfig.Labels = labelStrings
			}
		}

		// Parse draft
		if draft, exists := configMap["draft"]; exists {
			if draftBool, ok := draft.(bool); ok {
				pullRequestsConfig.Draft = &draftBool
			}
		}

		// Parse if-no-changes
		if ifNoChanges, exists := configMap["if-no-changes"]; exists {
			if ifNoChangesStr, ok := ifNoChanges.(string); ok {
				pullRequestsConfig.IfNoChanges = ifNoChangesStr
			}
		}

		// Parse min and github-token (max is always 1 for pull requests)
		if min, exists := configMap["min"]; exists {
			if minInt, ok := parseIntValue(min); ok {
				pullRequestsConfig.Min = minInt
			}
		}
		if githubToken, exists := configMap["github-token"]; exists {
			if githubTokenStr, ok := githubToken.(string); ok {
				pullRequestsConfig.GitHubToken = githubTokenStr
			}
		}

		// Note: max parameter is not supported for pull requests (always limited to 1)
		// If max is specified, it will be ignored as pull requests are singular only
	}

	return pullRequestsConfig
}
