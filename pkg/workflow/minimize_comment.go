package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var minimizeCommentLog = logger.New("workflow:minimize_comment")

// MinimizeCommentConfig holds configuration for minimizing (hiding) comments from agent output
type MinimizeCommentConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
}

// parseMinimizeCommentConfig handles minimize-comment configuration
func (c *Compiler) parseMinimizeCommentConfig(outputMap map[string]any) *MinimizeCommentConfig {
	minimizeCommentLog.Print("Parsing minimize-comment configuration")
	if configData, exists := outputMap["minimize-comment"]; exists {
		minimizeCommentConfig := &MinimizeCommentConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			minimizeCommentLog.Print("Found minimize-comment config map")

			// Parse target config (target-repo) with validation
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				return nil // Invalid configuration (e.g., wildcard target-repo), return nil to cause validation error
			}
			minimizeCommentConfig.SafeOutputTargetConfig = targetConfig

			// Parse common base fields with default max of 5
			c.parseBaseSafeOutputConfig(configMap, &minimizeCommentConfig.BaseSafeOutputConfig, 5)

			minimizeCommentLog.Printf("Parsed minimize-comment config: max=%d, target_repo=%s",
				minimizeCommentConfig.Max, minimizeCommentConfig.TargetRepoSlug)
		} else {
			// If configData is nil or not a map, still set the default max
			minimizeCommentConfig.Max = 5
		}

		return minimizeCommentConfig
	}

	return nil
}

// buildMinimizeCommentJob creates the minimize_comment job
func (c *Compiler) buildMinimizeCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	minimizeCommentLog.Printf("Building minimize_comment job: main_job=%s", mainJobName)
	if data.SafeOutputs == nil || data.SafeOutputs.MinimizeComment == nil {
		return nil, fmt.Errorf("safe-outputs.minimize-comment configuration is required")
	}

	cfg := data.SafeOutputs.MinimizeComment

	maxCount := 5
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables specific to minimize-comment
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_MINIMIZE_COMMENT_MAX_COUNT", maxCount)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id":   "${{ steps.minimize_comment.outputs.comment_id }}",
		"is_minimized": "${{ steps.minimize_comment.outputs.is_minimized }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "minimize_comment",
		StepName:       "Minimize Comment",
		StepID:         "minimize_comment",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getMinimizeCommentScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Token:          cfg.GitHubToken,
		Condition:      BuildSafeOutputType("minimize_comment"),
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// getMinimizeCommentScript returns the JavaScript implementation
func getMinimizeCommentScript() string {
	return embedJavaScript("minimize_comment.cjs")
}
