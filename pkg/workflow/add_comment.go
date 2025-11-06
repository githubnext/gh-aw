package workflow

import (
	"fmt"
)

// AddCommentConfig holds configuration for creating GitHub issue/PR comments from agent output (deprecated, use AddCommentsConfig)
type AddCommentConfig struct {
	// Empty struct for now, as per requirements, but structured for future expansion
}

// AddCommentsConfig holds configuration for creating GitHub issue/PR comments from agent output
type AddCommentsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string `yaml:"target,omitempty"`      // Target for comments: "triggering" (default), "*" (any issue), or explicit issue number
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository comments
	Discussion           *bool  `yaml:"discussion,omitempty"`  // Target discussion comments instead of issue/PR comments. Must be true if present.
}

// buildCreateOutputAddCommentJob creates the add_comment job
func (c *Compiler) buildCreateOutputAddCommentJob(data *WorkflowData, mainJobName string, createIssueJobName string, createDiscussionJobName string, createPullRequestJobName string) (*Job, error) {
	// Start building the job with the fluent builder
	builder := c.NewSafeOutputJobBuilder(data, "add_comment").
		WithConfig(data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil).
		WithStepMetadata("Add Issue Comment", "add_comment").
		WithMainJobName(mainJobName).
		WithScript(getAddCommentScript()).
		WithPermissions(NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite()).
		WithOutputs(map[string]string{
			"comment_id":  "${{ steps.add_comment.outputs.comment_id }}",
			"comment_url": "${{ steps.add_comment.outputs.comment_url }}",
		})

	// Build pre-steps for debugging output
	var preSteps []string
	preSteps = append(preSteps, "      - name: Debug agent outputs\n")
	preSteps = append(preSteps, "        env:\n")
	preSteps = append(preSteps, fmt.Sprintf("          AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	preSteps = append(preSteps, fmt.Sprintf("          AGENT_OUTPUT_TYPES: ${{ needs.%s.outputs.output_types }}\n", mainJobName))
	preSteps = append(preSteps, "        run: |\n")
	preSteps = append(preSteps, "          echo \"Output: $AGENT_OUTPUT\"\n")
	preSteps = append(preSteps, "          echo \"Output types: $AGENT_OUTPUT_TYPES\"\n")
	builder.WithPreSteps(preSteps)

	// Add job-specific environment variables
	builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

	if data.Source != "" {
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE: %q\n", data.Source))
		sourceURL := buildSourceURL(data.Source)
		if sourceURL != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE_URL: %q\n", sourceURL))
		}
	}

	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		if data.SafeOutputs.AddComments.Target != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_COMMENT_TARGET: %q\n", data.SafeOutputs.AddComments.Target))
		}
		if data.SafeOutputs.AddComments.Discussion != nil && *data.SafeOutputs.AddComments.Discussion {
			builder.AddEnvVar("          GITHUB_AW_COMMENT_DISCUSSION: \"true\"\n")
		}

		// Add environment variables for the URLs from other safe output jobs if they exist
		if createIssueJobName != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_CREATED_ISSUE_URL: ${{ needs.%s.outputs.issue_url }}\n", createIssueJobName))
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_CREATED_ISSUE_NUMBER: ${{ needs.%s.outputs.issue_number }}\n", createIssueJobName))
		}
		if createDiscussionJobName != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_CREATED_DISCUSSION_URL: ${{ needs.%s.outputs.discussion_url }}\n", createDiscussionJobName))
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_CREATED_DISCUSSION_NUMBER: ${{ needs.%s.outputs.discussion_number }}\n", createDiscussionJobName))
		}
		if createPullRequestJobName != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_CREATED_PULL_REQUEST_URL: ${{ needs.%s.outputs.pull_request_url }}\n", createPullRequestJobName))
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_CREATED_PULL_REQUEST_NUMBER: ${{ needs.%s.outputs.pull_request_number }}\n", createPullRequestJobName))
		}

		// Set token and target repo
		builder.WithToken(data.SafeOutputs.AddComments.GitHubToken).
			WithTargetRepoSlug(data.SafeOutputs.AddComments.TargetRepoSlug)

		// Build job condition with event check if target is not specified
		jobCondition := BuildSafeOutputType("add_comment")
		if data.SafeOutputs.AddComments.Target == "" {
			eventCondition := buildOr(
				buildOr(
					BuildPropertyAccess("github.event.issue.number"),
					BuildPropertyAccess("github.event.pull_request.number"),
				),
				BuildPropertyAccess("github.event.discussion.number"),
			)
			jobCondition = buildAnd(jobCondition, eventCondition)
		}
		builder.WithCondition(jobCondition)

		// Build the needs list - always depend on mainJobName, and conditionally on the other jobs
		needs := []string{mainJobName}
		if createIssueJobName != "" {
			needs = append(needs, createIssueJobName)
		}
		if createDiscussionJobName != "" {
			needs = append(needs, createDiscussionJobName)
		}
		if createPullRequestJobName != "" {
			needs = append(needs, createPullRequestJobName)
		}
		builder.WithNeeds(needs)
	}

	return builder.Build()
}

// parseCommentsConfig handles add-comment configuration
func (c *Compiler) parseCommentsConfig(outputMap map[string]any) *AddCommentsConfig {
	if configData, exists := outputMap["add-comment"]; exists {
		commentsConfig := &AddCommentsConfig{}
		commentsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &commentsConfig.BaseSafeOutputConfig)

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					commentsConfig.Target = targetStr
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			// Validate that target-repo is not "*" - only definite strings are allowed
			if targetRepoSlug == "*" {
				return nil // Invalid configuration, return nil to cause validation error
			}
			commentsConfig.TargetRepoSlug = targetRepoSlug

			// Parse discussion
			if discussion, exists := configMap["discussion"]; exists {
				if discussionBool, ok := discussion.(bool); ok {
					// Validate that discussion must be true if present
					if !discussionBool {
						return nil // Invalid configuration, return nil to cause validation error
					}
					commentsConfig.Discussion = &discussionBool
				}
			}
		}

		return commentsConfig
	}

	return nil
}
