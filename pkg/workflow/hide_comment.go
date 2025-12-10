package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var hideCommentLog = logger.New("workflow:hide_comment")

// HideCommentConfig holds configuration for hiding comments from agent output
type HideCommentConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
}

// parseHideCommentConfig handles hide-comment configuration
func (c *Compiler) parseHideCommentConfig(outputMap map[string]any) *HideCommentConfig {
	hideCommentLog.Print("Parsing hide-comment configuration")
	if configData, exists := outputMap["hide-comment"]; exists {
		hideCommentConfig := &HideCommentConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			hideCommentLog.Print("Found hide-comment config map")

			// Parse target config (target-repo) with validation
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				return nil // Invalid configuration (e.g., wildcard target-repo), return nil to cause validation error
			}
			hideCommentConfig.SafeOutputTargetConfig = targetConfig

			// Parse common base fields with default max of 5
			c.parseBaseSafeOutputConfig(configMap, &hideCommentConfig.BaseSafeOutputConfig, 5)

			hideCommentLog.Printf("Parsed hide-comment config: max=%d, target_repo=%s",
				hideCommentConfig.Max, hideCommentConfig.TargetRepoSlug)
		} else {
			// If configData is nil or not a map, still set the default max
			hideCommentConfig.Max = 5
		}

		return hideCommentConfig
	}

	return nil
}

// buildHideCommentJob creates the hide_comment job
func (c *Compiler) buildHideCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	hideCommentLog.Printf("Building hide_comment job: main_job=%s", mainJobName)
	if data.SafeOutputs == nil || data.SafeOutputs.HideComment == nil {
		return nil, fmt.Errorf("safe-outputs.hide-comment configuration is required")
	}

	cfg := data.SafeOutputs.HideComment

	maxCount := 5
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables specific to hide-comment
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_HIDE_COMMENT_MAX_COUNT", maxCount)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id": "${{ steps.hide_comment.outputs.comment_id }}",
		"is_hidden":  "${{ steps.hide_comment.outputs.is_hidden }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "hide_comment",
		StepName:       "Hide Comment",
		StepID:         "hide_comment",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getHideCommentScript(),
		Permissions:    NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite(),
		Outputs:        outputs,
		Token:          cfg.GitHubToken,
		Condition:      BuildSafeOutputType("hide_comment"),
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// getHideCommentScript returns the JavaScript implementation
func getHideCommentScript() string {
	return DefaultScriptRegistry.GetWithMode("hide_comment", RuntimeModeGitHubScript)
}
