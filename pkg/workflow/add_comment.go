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
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil {
		return nil, fmt.Errorf("safe-outputs.add-comment configuration is required")
	}

	// Build pre-steps for debugging output
	var preSteps []string
	preSteps = append(preSteps, "      - name: Debug agent outputs\n")
	preSteps = append(preSteps, "        env:\n")
	preSteps = append(preSteps, fmt.Sprintf("          AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	preSteps = append(preSteps, fmt.Sprintf("          AGENT_OUTPUT_TYPES: ${{ needs.%s.outputs.output_types }}\n", mainJobName))
	preSteps = append(preSteps, "        run: |\n")
	preSteps = append(preSteps, "          echo \"Output: $AGENT_OUTPUT\"\n")
	preSteps = append(preSteps, "          echo \"Output types: $AGENT_OUTPUT_TYPES\"\n")

	// Build custom environment variables specific to add-comment
	var customEnvVars []string

	// Add workflow metadata (name, source, and fingerprint)
	customEnvVars = append(customEnvVars, buildWorkflowMetadataEnvVarsWithFingerprint(data.Name, data.Source, data.Fingerprint)...)

	// Pass the comment target configuration
	if data.SafeOutputs.AddComments.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_TARGET: %q\n", data.SafeOutputs.AddComments.Target))
	}
	// Pass the discussion flag configuration
	if data.SafeOutputs.AddComments.Discussion != nil && *data.SafeOutputs.AddComments.Discussion {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_COMMENT_DISCUSSION: \"true\"\n")
	}
	// Add environment variables for the URLs from other safe output jobs if they exist
	if createIssueJobName != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATED_ISSUE_URL: ${{ needs.%s.outputs.issue_url }}\n", createIssueJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATED_ISSUE_NUMBER: ${{ needs.%s.outputs.issue_number }}\n", createIssueJobName))
	}
	if createDiscussionJobName != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATED_DISCUSSION_URL: ${{ needs.%s.outputs.discussion_url }}\n", createDiscussionJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATED_DISCUSSION_NUMBER: ${{ needs.%s.outputs.discussion_number }}\n", createDiscussionJobName))
	}
	if createPullRequestJobName != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATED_PULL_REQUEST_URL: ${{ needs.%s.outputs.pull_request_url }}\n", createPullRequestJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATED_PULL_REQUEST_NUMBER: ${{ needs.%s.outputs.pull_request_number }}\n", createPullRequestJobName))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.AddComments.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id":  "${{ steps.add_comment.outputs.comment_id }}",
		"comment_url": "${{ steps.add_comment.outputs.comment_url }}",
	}

	// Build job condition with event check if target is not specified
	jobCondition := BuildSafeOutputType("add_comment")
	if data.SafeOutputs.AddComments != nil && data.SafeOutputs.AddComments.Target == "" {
		eventCondition := buildOr(
			buildOr(
				BuildPropertyAccess("github.event.issue.number"),
				BuildPropertyAccess("github.event.pull_request.number"),
			),
			BuildPropertyAccess("github.event.discussion.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

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

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "add_comment",
		StepName:       "Add Issue Comment",
		StepID:         "add_comment",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAddCommentScript(),
		Permissions:    NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Needs:          needs,
		PreSteps:       preSteps,
		Token:          token,
		TargetRepoSlug: data.SafeOutputs.AddComments.TargetRepoSlug,
	})
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
