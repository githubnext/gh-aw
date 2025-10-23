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
	Reviewers            []string `yaml:"reviewers,omitempty"`     // List of users/bots to assign as reviewers to the pull request
	Draft                *bool    `yaml:"draft,omitempty"`         // Pointer to distinguish between unset (nil) and explicitly false
	IfNoChanges          string   `yaml:"if-no-changes,omitempty"` // Behavior when no changes to push: "warn" (default), "error", or "ignore"
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository pull requests
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
	steps = append(steps, "          path: /tmp/gh-aw/\n")

	// Step 2: Checkout repository
	steps = buildCheckoutRepository(steps, c)

	// Step 3: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Build custom environment variables specific to create-pull-request
	var customEnvVars []string
	// Pass the workflow ID for branch naming
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", mainJobName))
	// Pass the workflow name for footer generation
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	// Pass the base branch from GitHub context
	customEnvVars = append(customEnvVars, "          GH_AW_BASE_BRANCH: ${{ github.ref_name }}\n")
	if data.SafeOutputs.CreatePullRequests.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_TITLE_PREFIX: %q\n", data.SafeOutputs.CreatePullRequests.TitlePrefix))
	}
	if len(data.SafeOutputs.CreatePullRequests.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreatePullRequests.Labels, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_LABELS: %q\n", labelsStr))
	}
	// Pass draft setting - default to true for backwards compatibility
	draftValue := true // Default value
	if data.SafeOutputs.CreatePullRequests.Draft != nil {
		draftValue = *data.SafeOutputs.CreatePullRequests.Draft
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_DRAFT: %q\n", fmt.Sprintf("%t", draftValue)))

	// Pass the if-no-changes configuration
	ifNoChanges := data.SafeOutputs.CreatePullRequests.IfNoChanges
	if ifNoChanges == "" {
		ifNoChanges = "warn" // Default value
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_IF_NO_CHANGES: %q\n", ifNoChanges))

	// Pass the maximum patch size configuration
	maxPatchSize := 1024 // Default value
	if data.SafeOutputs != nil && data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CreatePullRequests.TargetRepoSlug,
	)...)

	// Step 4: Create pull request using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Create Pull Request",
		StepID:        "create_pull_request",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        createPullRequestScript,
		Token:         data.SafeOutputs.CreatePullRequests.GitHubToken,
	})
	steps = append(steps, scriptSteps...)

	// Add reviewer steps if reviewers are configured
	if len(data.SafeOutputs.CreatePullRequests.Reviewers) > 0 {
		// Add checkout step for gh CLI to work
		steps = append(steps, "      - name: Checkout repository for gh CLI\n")
		steps = append(steps, "        if: steps.create_pull_request.outputs.pull_request_url != ''\n")
		steps = append(steps, "        uses: actions/checkout@v5\n")

		// Get the effective GitHub token to use for gh CLI
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}

		// Check if any reviewer is "copilot" to determine token preference
		hasCopilotReviewer := false
		for _, reviewer := range data.SafeOutputs.CreatePullRequests.Reviewers {
			if reviewer == "copilot" {
				hasCopilotReviewer = true
				break
			}
		}

		// Use Copilot token preference if adding copilot as reviewer, otherwise use regular token
		var effectiveToken string
		if hasCopilotReviewer {
			effectiveToken = getEffectiveCopilotGitHubToken(data.SafeOutputs.CreatePullRequests.GitHubToken, getEffectiveCopilotGitHubToken(safeOutputsToken, data.GitHubToken))
		} else {
			effectiveToken = getEffectiveGitHubToken(data.SafeOutputs.CreatePullRequests.GitHubToken, getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken))
		}

		for i, reviewer := range data.SafeOutputs.CreatePullRequests.Reviewers {
			// Special handling: "copilot" uses the GitHub API with "copilot-pull-request-reviewer[bot]"
			// because gh pr edit --add-reviewer does not support @copilot
			if reviewer == "copilot" {
				steps = append(steps, fmt.Sprintf("      - name: Add %s as reviewer\n", reviewer))
				steps = append(steps, "        if: steps.create_pull_request.outputs.pull_request_number != ''\n")
				steps = append(steps, "        env:\n")
				steps = append(steps, fmt.Sprintf("          GH_TOKEN: %s\n", effectiveToken))
				steps = append(steps, "          PR_NUMBER: ${{ steps.create_pull_request.outputs.pull_request_number }}\n")
				steps = append(steps, "        run: |\n")
				steps = append(steps, "          gh api --method POST /repos/${{ github.repository }}/pulls/$PR_NUMBER/requested_reviewers \\\n")
				steps = append(steps, "            -f 'reviewers[]=copilot-pull-request-reviewer[bot]'\n")
			} else {
				steps = append(steps, fmt.Sprintf("      - name: Add %s as reviewer\n", reviewer))
				steps = append(steps, "        if: steps.create_pull_request.outputs.pull_request_url != ''\n")
				steps = append(steps, "        env:\n")
				steps = append(steps, fmt.Sprintf("          GH_TOKEN: %s\n", effectiveToken))
				steps = append(steps, fmt.Sprintf("          REVIEWER: %q\n", reviewer))
				steps = append(steps, "          PR_URL: ${{ steps.create_pull_request.outputs.pull_request_url }}\n")
				steps = append(steps, "        run: |\n")
				steps = append(steps, "          gh pr edit \"$PR_URL\" --add-reviewer \"$REVIEWER\"\n")
			}

			// Add a comment after each reviewer step except the last
			if i < len(data.SafeOutputs.CreatePullRequests.Reviewers)-1 {
				steps = append(steps, "\n")
			}
		}
	}

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.create_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.create_pull_request.outputs.pull_request_url }}",
		"issue_number":        "${{ steps.create_pull_request.outputs.issue_number }}",
		"issue_url":           "${{ steps.create_pull_request.outputs.issue_url }}",
		"branch_name":         "${{ steps.create_pull_request.outputs.branch_name }}",
		"fallback_used":       "${{ steps.create_pull_request.outputs.fallback_used }}",
	}

	jobCondition := BuildSafeOutputType("create_pull_request", data.SafeOutputs.CreatePullRequests.Min)

	job := &Job{
		Name:           "create_pull_request",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsWriteIssuesWritePRWrite().RenderToYAML(),
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
		// Parse title-prefix using shared helper
		pullRequestsConfig.TitlePrefix = parseTitlePrefixFromConfig(configMap)

		// Parse labels using shared helper
		pullRequestsConfig.Labels = parseLabelsFromConfig(configMap)

		// Parse reviewers (supports both string and array)
		if reviewers, exists := configMap["reviewers"]; exists {
			if reviewerStr, ok := reviewers.(string); ok {
				// Single string format
				pullRequestsConfig.Reviewers = []string{reviewerStr}
			} else if reviewersArray, ok := reviewers.([]any); ok {
				// Array format
				var reviewerStrings []string
				for _, reviewer := range reviewersArray {
					if reviewerStr, ok := reviewer.(string); ok {
						reviewerStrings = append(reviewerStrings, reviewerStr)
					}
				}
				pullRequestsConfig.Reviewers = reviewerStrings
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

		// Parse target-repo using shared helper
		targetRepoSlug := parseTargetRepoFromConfig(configMap)
		// Validate that target-repo is not "*" - only definite strings are allowed
		if targetRepoSlug == "*" {
			return nil // Invalid configuration, return nil to cause validation error
		}
		pullRequestsConfig.TargetRepoSlug = targetRepoSlug

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
