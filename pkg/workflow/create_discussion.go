package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var discussionLog = logger.New("workflow:create_discussion")

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
		discussionLog.Print("Parsing create-discussion configuration")
		discussionsConfig := &CreateDiscussionsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix using shared helper
			discussionsConfig.TitlePrefix = parseTitlePrefixFromConfig(configMap)
			if discussionsConfig.TitlePrefix != "" {
				discussionLog.Printf("Title prefix configured: %q", discussionsConfig.TitlePrefix)
			}

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
				discussionLog.Printf("Discussion category configured: %q", discussionsConfig.Category)
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				discussionLog.Print("Invalid target-repo configuration")
				return nil // Invalid configuration, return nil to cause validation error
			}
			if targetRepoSlug != "" {
				discussionLog.Printf("Target repository configured: %s", targetRepoSlug)
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
	discussionLog.Printf("Building create_discussion job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.CreateDiscussions == nil {
		return nil, fmt.Errorf("safe-outputs.create-discussion configuration is required")
	}

	// Build custom environment variables specific to create-discussion using shared helpers
	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_DISCUSSION_TITLE_PREFIX", data.SafeOutputs.CreateDiscussions.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildCategoryEnvVar("GH_AW_DISCUSSION_CATEGORY", data.SafeOutputs.CreateDiscussions.Category)...)
	discussionLog.Printf("Configured %d custom environment variables for discussion creation", len(customEnvVars))

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
