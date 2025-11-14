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

		if configMap, ok := configData.(map[string]any); ok {
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

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			discussionsConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &discussionsConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "create-discussion:" with no value),
			// still set the default max
			discussionsConfig.Max = 1
		}

		return discussionsConfig
	}

	return nil
}

// buildCreateOutputDiscussionJob creates the create_discussion job
func (c *Compiler) buildCreateOutputDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateDiscussions == nil {
		return nil, fmt.Errorf("safe-outputs.create-discussion configuration is required")
	}

	// Build custom environment variables specific to create-discussion
	var customEnvVars []string

	if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISCUSSION_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateDiscussions.TitlePrefix))
	}
	if data.SafeOutputs.CreateDiscussions.Category != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISCUSSION_CATEGORY: %q\n", data.SafeOutputs.CreateDiscussions.Category))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CreateDiscussions.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "create_discussion",
		StepName:       "Create Output Discussion",
		StepID:         "create_discussion",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCreateDiscussionScript(),
		Permissions:    NewPermissionsContentsReadDiscussionsWrite(),
		Outputs:        outputs,
		Token:          data.SafeOutputs.CreateDiscussions.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CreateDiscussions.TargetRepoSlug,
	})
}
