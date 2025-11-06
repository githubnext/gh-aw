package workflow

import (
	"fmt"
)

// CreateDiscussionsConfig holds configuration for creating GitHub discussions from agent output
type CreateDiscussionsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string `yaml:"title-prefix,omitempty"`
	Category             string `yaml:"category,omitempty"`    // Discussion category ID or name
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository discussions
}

// parseDiscussionsConfig handles create-discussion configuration
func (c *Compiler) parseDiscussionsConfig(outputMap map[string]any) *CreateDiscussionsConfig {
	if configData, exists := outputMap["create-discussion"]; exists {
		discussionsConfig := &CreateDiscussionsConfig{}
		discussionsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &discussionsConfig.BaseSafeOutputConfig)

			// Parse title-prefix using shared helper
			discussionsConfig.TitlePrefix = parseTitlePrefixFromConfig(configMap)

			// Parse category (can be string or number)
			if category, exists := configMap["category"]; exists {
				switch v := category.(type) {
				case string:
					discussionsConfig.Category = v
				case int:
					discussionsConfig.Category = fmt.Sprintf("%d", v)
				case int64:
					discussionsConfig.Category = fmt.Sprintf("%d", v)
				case float64:
					discussionsConfig.Category = fmt.Sprintf("%.0f", v)
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			// Validate that target-repo is not "*" - only definite strings are allowed
			if targetRepoSlug == "*" {
				return nil // Invalid configuration, return nil to cause validation error
			}
			discussionsConfig.TargetRepoSlug = targetRepoSlug
		}

		return discussionsConfig
	}

	return nil
}

// buildCreateOutputDiscussionJob creates the create_discussion job
func (c *Compiler) buildCreateOutputDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	// Start building the job with the fluent builder
	builder := c.NewSafeOutputJobBuilder(data, "create_discussion").
		WithConfig(data.SafeOutputs == nil || data.SafeOutputs.CreateDiscussions == nil).
		WithStepMetadata("Create Output Discussion", "create_discussion").
		WithMainJobName(mainJobName).
		WithScript(getCreateDiscussionScript()).
		WithPermissions(NewPermissionsContentsReadDiscussionsWrite()).
		WithOutputs(map[string]string{
			"discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
			"discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
		})

	// Add job-specific environment variables
	builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

	if data.SafeOutputs != nil && data.SafeOutputs.CreateDiscussions != nil {
		if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_DISCUSSION_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateDiscussions.TitlePrefix))
		}
		if data.SafeOutputs.CreateDiscussions.Category != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_DISCUSSION_CATEGORY: %q\n", data.SafeOutputs.CreateDiscussions.Category))
		}

		// Set token and target repo
		builder.WithToken(data.SafeOutputs.CreateDiscussions.GitHubToken).
			WithTargetRepoSlug(data.SafeOutputs.CreateDiscussions.TargetRepoSlug)
	}

	return builder.Build()
}
