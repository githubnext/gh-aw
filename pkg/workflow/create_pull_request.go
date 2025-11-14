package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var createPRLog = logger.New("workflow:create_pull_request")

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

	if createPRLog.Enabled() {
		draftValue := true // Default
		if data.SafeOutputs.CreatePullRequests.Draft != nil {
			draftValue = *data.SafeOutputs.CreatePullRequests.Draft
		}
		createPRLog.Printf("Building create-pull-request job: workflow=%s, main_job=%s, draft=%v, reviewers=%d",
			data.Name, mainJobName, draftValue, len(data.SafeOutputs.CreatePullRequests.Reviewers))
	}

	// Build pre-steps for patch download, checkout, and git config
	var preSteps []string

	// Step 1: Download patch artifact
	preSteps = append(preSteps, "      - name: Download patch artifact\n")
	preSteps = append(preSteps, "        continue-on-error: true\n")
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, "          name: aw.patch\n")
	preSteps = append(preSteps, "          path: /tmp/gh-aw/\n")

	// Step 2: Checkout repository
	preSteps = buildCheckoutRepository(preSteps, c)

	// Step 3: Configure Git credentials
	preSteps = append(preSteps, c.generateGitConfigurationSteps()...)

	// Build custom environment variables specific to create-pull-request
	var customEnvVars []string
	// Pass the workflow ID for branch naming
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", mainJobName))
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

	// Pass activation comment information if available (for updating the comment with PR link)
	// These outputs are only available when reaction is configured in the workflow
	if data.AIReaction != "" && data.AIReaction != "none" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CreatePullRequests.TargetRepoSlug)...)

	// Build post-steps for reviewers if configured
	var postSteps []string
	if len(data.SafeOutputs.CreatePullRequests.Reviewers) > 0 {
		// Get the effective GitHub token to use for gh CLI
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}

		postSteps = buildCopilotParticipantSteps(CopilotParticipantConfig{
			Participants:       data.SafeOutputs.CreatePullRequests.Reviewers,
			ParticipantType:    "reviewer",
			CustomToken:        data.SafeOutputs.CreatePullRequests.GitHubToken,
			SafeOutputsToken:   safeOutputsToken,
			WorkflowToken:      data.GitHubToken,
			ConditionStepID:    "create_pull_request",
			ConditionOutputKey: "pull_request_url",
		})
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

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "create_pull_request",
		StepName:       "Create Pull Request",
		StepID:         "create_pull_request",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCreatePullRequestScript(),
		Permissions:    NewPermissionsContentsWriteIssuesWritePRWrite(),
		Outputs:        outputs,
		PreSteps:       preSteps,
		PostSteps:      postSteps,
		Token:          data.SafeOutputs.CreatePullRequests.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CreatePullRequests.TargetRepoSlug,
	})
}

// parsePullRequestsConfig handles only create-pull-request (singular) configuration
func (c *Compiler) parsePullRequestsConfig(outputMap map[string]any) *CreatePullRequestsConfig {
	// Check for singular form only
	if _, exists := outputMap["create-pull-request"]; !exists {
		createPRLog.Print("No create-pull-request configuration found")
		return nil
	}

	createPRLog.Print("Parsing create-pull-request configuration")
	configData := outputMap["create-pull-request"]
	pullRequestsConfig := &CreatePullRequestsConfig{}

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

		// Parse target-repo using shared helper with validation
		targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
		if isInvalid {
			return nil // Invalid configuration, return nil to cause validation error
		}
		pullRequestsConfig.TargetRepoSlug = targetRepoSlug

		// Parse common base fields (github-token, max if specified by user)
		c.parseBaseSafeOutputConfig(configMap, &pullRequestsConfig.BaseSafeOutputConfig, -1)

		// Note: max parameter is not supported for pull requests (always limited to 1)
		// Override any user-specified max value to enforce the limit
		pullRequestsConfig.Max = 1
	} else {
		// If configData is nil or not a map (e.g., "create-pull-request:" with no value),
		// still set the default max (always 1 for pull requests)
		pullRequestsConfig.Max = 1
	}

	return pullRequestsConfig
}
