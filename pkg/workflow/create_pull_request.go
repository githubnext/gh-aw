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
	// Start building the job with the fluent builder
	builder := c.NewSafeOutputJobBuilder(data, "create_pull_request").
		WithConfig(data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequests == nil).
		WithStepMetadata("Create Pull Request", "create_pull_request").
		WithMainJobName(mainJobName).
		WithScript(createPullRequestScript).
		WithPermissions(NewPermissionsContentsWriteIssuesWritePRWrite()).
		WithOutputs(map[string]string{
			"pull_request_number": "${{ steps.create_pull_request.outputs.pull_request_number }}",
			"pull_request_url":    "${{ steps.create_pull_request.outputs.pull_request_url }}",
			"issue_number":        "${{ steps.create_pull_request.outputs.issue_number }}",
			"issue_url":           "${{ steps.create_pull_request.outputs.issue_url }}",
			"branch_name":         "${{ steps.create_pull_request.outputs.branch_name }}",
			"fallback_used":       "${{ steps.create_pull_request.outputs.fallback_used }}",
		})

	// Build pre-steps for patch download, checkout, and git config
	var preSteps []string
	preSteps = append(preSteps, "      - name: Download patch artifact\n")
	preSteps = append(preSteps, "        continue-on-error: true\n")
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, "          name: aw.patch\n")
	preSteps = append(preSteps, "          path: /tmp/gh-aw/\n")
	preSteps = buildCheckoutRepository(preSteps, c)
	preSteps = append(preSteps, c.generateGitConfigurationSteps()...)
	builder.WithPreSteps(preSteps)

	// Add job-specific environment variables
	builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", mainJobName))
	builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	builder.AddEnvVar("          GH_AW_BASE_BRANCH: ${{ github.ref_name }}\n")

	if data.SafeOutputs != nil && data.SafeOutputs.CreatePullRequests != nil {
		if data.SafeOutputs.CreatePullRequests.TitlePrefix != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_PR_TITLE_PREFIX: %q\n", data.SafeOutputs.CreatePullRequests.TitlePrefix))
		}
		if len(data.SafeOutputs.CreatePullRequests.Labels) > 0 {
			labelsStr := strings.Join(data.SafeOutputs.CreatePullRequests.Labels, ",")
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_PR_LABELS: %q\n", labelsStr))
		}

		// Pass draft setting - default to true for backwards compatibility
		draftValue := true
		if data.SafeOutputs.CreatePullRequests.Draft != nil {
			draftValue = *data.SafeOutputs.CreatePullRequests.Draft
		}
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_PR_DRAFT: %q\n", fmt.Sprintf("%t", draftValue)))

		// Pass the if-no-changes configuration
		ifNoChanges := data.SafeOutputs.CreatePullRequests.IfNoChanges
		if ifNoChanges == "" {
			ifNoChanges = "warn"
		}
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_PR_IF_NO_CHANGES: %q\n", ifNoChanges))

		// Pass the maximum patch size configuration
		maxPatchSize := 1024
		if data.SafeOutputs != nil && data.SafeOutputs.MaximumPatchSize > 0 {
			maxPatchSize = data.SafeOutputs.MaximumPatchSize
		}
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))

		// Set token and target repo
		builder.WithToken(data.SafeOutputs.CreatePullRequests.GitHubToken).
			WithTargetRepoSlug(data.SafeOutputs.CreatePullRequests.TargetRepoSlug)

		// Build post-steps for reviewers if configured
		if len(data.SafeOutputs.CreatePullRequests.Reviewers) > 0 {
			var safeOutputsToken string
			if data.SafeOutputs != nil {
				safeOutputsToken = data.SafeOutputs.GitHubToken
			}

			postSteps := buildCopilotParticipantSteps(CopilotParticipantConfig{
				Participants:       data.SafeOutputs.CreatePullRequests.Reviewers,
				ParticipantType:    "reviewer",
				CustomToken:        data.SafeOutputs.CreatePullRequests.GitHubToken,
				SafeOutputsToken:   safeOutputsToken,
				WorkflowToken:      data.GitHubToken,
				ConditionStepID:    "create_pull_request",
				ConditionOutputKey: "pull_request_url",
			})
			builder.WithPostSteps(postSteps)
		}
	}

	return builder.Build()
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

		// Parse github-token (max is always 1 for pull requests)
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
