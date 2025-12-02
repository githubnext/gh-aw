package workflow

import (
	"fmt"
)

// AddReviewerConfig holds configuration for adding reviewers to PRs from agent output
type AddReviewerConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Reviewers              []string `yaml:"reviewers,omitempty"` // Optional list of allowed reviewers. If omitted, any reviewers are allowed.
}

// buildAddReviewerJob creates the add_reviewer job
func (c *Compiler) buildAddReviewerJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddReviewer == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	cfg := data.SafeOutputs.AddReviewer

	// Handle max count with default of 3
	maxCount := 3
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables using shared helpers
	listJobConfig := ListJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		Allowed:                cfg.Reviewers,
	}
	customEnvVars := BuildListJobEnvVars("GH_AW_REVIEWERS", listJobConfig, maxCount)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"reviewers_added": "${{ steps.add_reviewer.outputs.reviewers_added }}",
	}

	var jobCondition = BuildSafeOutputType("add_reviewer")
	if cfg.Target == "" {
		// Only run if in PR context when target is not specified
		prCondition := BuildPropertyAccess("github.event.pull_request.number")
		jobCondition = buildAnd(jobCondition, prCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "add_reviewer",
		StepName:       "Add Reviewers",
		StepID:         "add_reviewer",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAddReviewerScript(),
		Permissions:    NewPermissionsContentsReadPRWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// parseAddReviewerConfig handles add-reviewer configuration
func (c *Compiler) parseAddReviewerConfig(outputMap map[string]any) *AddReviewerConfig {
	if configData, exists := outputMap["add-reviewer"]; exists {
		addReviewerConfig := &AddReviewerConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse reviewers (supports both string and array)
			if reviewers, exists := configMap["reviewers"]; exists {
				if reviewerStr, ok := reviewers.(string); ok {
					// Single string format
					addReviewerConfig.Reviewers = []string{reviewerStr}
				} else if reviewersArray, ok := reviewers.([]any); ok {
					// Array format
					var reviewerStrings []string
					for _, reviewer := range reviewersArray {
						if reviewerStr, ok := reviewer.(string); ok {
							reviewerStrings = append(reviewerStrings, reviewerStr)
						}
					}
					addReviewerConfig.Reviewers = reviewerStrings
				}
			}

			// Parse target config (target, target-repo)
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			addReviewerConfig.SafeOutputTargetConfig = targetConfig

			// Parse common base fields (github-token, max) with default max of 3
			c.parseBaseSafeOutputConfig(configMap, &addReviewerConfig.BaseSafeOutputConfig, 3)
		} else {
			// If configData is nil or not a map (e.g., "add-reviewer:" with no value),
			// still set the default max
			addReviewerConfig.Max = 3
		}

		return addReviewerConfig
	}

	return nil
}
