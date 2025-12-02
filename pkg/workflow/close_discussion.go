package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var closeDiscussionLog = logger.New("workflow:close_discussion")

// CloseDiscussionsConfig holds configuration for closing GitHub discussions from agent output
type CloseDiscussionsConfig struct {
	BaseSafeOutputConfig             `yaml:",inline"`
	SafeOutputTargetConfig           `yaml:",inline"`
	SafeOutputDiscussionFilterConfig `yaml:",inline"`
}

// parseCloseDiscussionsConfig handles close-discussion configuration
func (c *Compiler) parseCloseDiscussionsConfig(outputMap map[string]any) *CloseDiscussionsConfig {
	if configData, exists := outputMap["close-discussion"]; exists {
		closeDiscussionLog.Print("Parsing close-discussion configuration")
		closeDiscussionsConfig := &CloseDiscussionsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target config
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				closeDiscussionLog.Print("Invalid target-repo configuration")
				return nil
			}
			closeDiscussionsConfig.SafeOutputTargetConfig = targetConfig

			// Parse discussion filter config (required-labels, required-title-prefix, required-category)
			closeDiscussionsConfig.SafeOutputDiscussionFilterConfig = ParseDiscussionFilterConfig(configMap)

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &closeDiscussionsConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "close-discussion:" with no value),
			// still set the default max
			closeDiscussionsConfig.Max = 1
		}

		return closeDiscussionsConfig
	}

	return nil
}

// buildCreateOutputCloseDiscussionJob creates the close_discussion job
func (c *Compiler) buildCreateOutputCloseDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	closeDiscussionLog.Printf("Building close_discussion job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.CloseDiscussions == nil {
		return nil, fmt.Errorf("safe-outputs.close-discussion configuration is required")
	}

	cfg := data.SafeOutputs.CloseDiscussions

	// Build custom environment variables specific to close-discussion using shared helpers
	closeJobConfig := CloseJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		SafeOutputFilterConfig: cfg.SafeOutputFilterConfig,
	}
	customEnvVars := BuildCloseJobEnvVars("GH_AW_CLOSE_DISCUSSION", closeJobConfig)

	// Add required-category env var (discussion-specific)
	customEnvVars = append(customEnvVars, BuildRequiredCategoryEnvVar("GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY", cfg.RequiredCategory)...)

	closeDiscussionLog.Printf("Configured %d custom environment variables for discussion close", len(customEnvVars))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"discussion_number": "${{ steps.close_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.close_discussion.outputs.discussion_url }}",
		"comment_url":       "${{ steps.close_discussion.outputs.comment_url }}",
	}

	// Build job condition with discussion event check only for "triggering" target
	// If target is "*" (any discussion) or explicitly set, allow agent to provide discussion_number
	jobCondition := BuildSafeOutputType("close_discussion")
	if cfg.Target == "" || cfg.Target == "triggering" {
		// Only require event discussion context for "triggering" target
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.discussion.number"),
			BuildPropertyAccess("github.event.comment.discussion.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "close_discussion",
		StepName:       "Close Discussion",
		StepID:         "close_discussion",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCloseDiscussionScript(),
		Permissions:    NewPermissionsContentsReadDiscussionsWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
