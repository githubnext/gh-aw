package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var closeDiscussionLog = logger.New("workflow:close_discussion")

// CloseDiscussionsConfig holds configuration for closing GitHub discussions from agent output
type CloseDiscussionsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	RequiredLabels       []string `yaml:"required-labels,omitempty"`       // Required labels for closing
	RequiredTitlePrefix  string   `yaml:"required-title-prefix,omitempty"` // Required title prefix for closing
	RequiredCategory     string   `yaml:"required-category,omitempty"`     // Required category for closing
	Target               string   `yaml:"target,omitempty"`                // Target for close: "triggering" (default), "*" (any discussion), or explicit number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`           // Target repository for cross-repo operations
}

// parseCloseDiscussionsConfig handles close-discussion configuration
func (c *Compiler) parseCloseDiscussionsConfig(outputMap map[string]any) *CloseDiscussionsConfig {
	if configData, exists := outputMap["close-discussion"]; exists {
		closeDiscussionLog.Print("Parsing close-discussion configuration")
		closeDiscussionsConfig := &CloseDiscussionsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse required-labels
			if requiredLabels, exists := configMap["required-labels"]; exists {
				if labelList, ok := requiredLabels.([]any); ok {
					for _, label := range labelList {
						if labelStr, ok := label.(string); ok {
							closeDiscussionsConfig.RequiredLabels = append(closeDiscussionsConfig.RequiredLabels, labelStr)
						}
					}
				}
				closeDiscussionLog.Printf("Required labels configured: %v", closeDiscussionsConfig.RequiredLabels)
			}

			// Parse required-title-prefix
			if requiredTitlePrefix, exists := configMap["required-title-prefix"]; exists {
				if prefix, ok := requiredTitlePrefix.(string); ok {
					closeDiscussionsConfig.RequiredTitlePrefix = prefix
					closeDiscussionLog.Printf("Required title prefix configured: %q", prefix)
				}
			}

			// Parse required-category
			if requiredCategory, exists := configMap["required-category"]; exists {
				if category, ok := requiredCategory.(string); ok {
					closeDiscussionsConfig.RequiredCategory = category
					closeDiscussionLog.Printf("Required category configured: %q", category)
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					closeDiscussionsConfig.Target = targetStr
					closeDiscussionLog.Printf("Target configured: %q", targetStr)
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				closeDiscussionLog.Print("Invalid target-repo configuration")
				return nil // Invalid configuration, return nil to cause validation error
			}
			if targetRepoSlug != "" {
				closeDiscussionLog.Printf("Target repository configured: %s", targetRepoSlug)
			}
			closeDiscussionsConfig.TargetRepoSlug = targetRepoSlug

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

	// Build custom environment variables specific to close-discussion
	var customEnvVars []string

	if len(data.SafeOutputs.CloseDiscussions.RequiredLabels) > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS: %q\n", strings.Join(data.SafeOutputs.CloseDiscussions.RequiredLabels, ",")))
	}
	if data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_DISCUSSION_REQUIRED_TITLE_PREFIX: %q\n", data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix))
	}
	if data.SafeOutputs.CloseDiscussions.RequiredCategory != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY: %q\n", data.SafeOutputs.CloseDiscussions.RequiredCategory))
	}
	if data.SafeOutputs.CloseDiscussions.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_DISCUSSION_TARGET: %q\n", data.SafeOutputs.CloseDiscussions.Target))
	}
	closeDiscussionLog.Printf("Configured %d custom environment variables for discussion close", len(customEnvVars))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CloseDiscussions.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"discussion_number": "${{ steps.close_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.close_discussion.outputs.discussion_url }}",
		"comment_url":       "${{ steps.close_discussion.outputs.comment_url }}",
	}

	// Build job condition with discussion event check if target is not specified
	jobCondition := BuildSafeOutputType("close_discussion")
	if data.SafeOutputs.CloseDiscussions != nil && data.SafeOutputs.CloseDiscussions.Target == "" {
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
		Token:          data.SafeOutputs.CloseDiscussions.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CloseDiscussions.TargetRepoSlug,
	})
}
