package workflow

import (
	"fmt"
	"strings"
)

// AddReviewerConfig holds configuration for adding reviewers to PRs from agent output
type AddReviewerConfig struct {
	Reviewers      []string `yaml:"reviewers,omitempty"`    // Optional list of allowed reviewers. If omitted, any reviewers are allowed.
	Max            int      `yaml:"max,omitempty"`          // Optional maximum number of reviewers to add (default: 3)
	GitHubToken    string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
	Target         string   `yaml:"target,omitempty"`       // Target for reviewers: "triggering" (default), "*" (any PR), or explicit PR number
	TargetRepoSlug string   `yaml:"target-repo,omitempty"`  // Target repository in format "owner/repo" for cross-repository reviewers
}

// buildAddReviewerJob creates the add_reviewer job
func (c *Compiler) buildAddReviewerJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddReviewer == nil {
		return nil, fmt.Errorf("safe-outputs configuration is required")
	}

	// Handle case where AddReviewer configuration is provided
	var allowedReviewers []string
	maxCount := 3

	if data.SafeOutputs.AddReviewer != nil {
		allowedReviewers = data.SafeOutputs.AddReviewer.Reviewers
		if data.SafeOutputs.AddReviewer.Max > 0 {
			maxCount = data.SafeOutputs.AddReviewer.Max
		}
	}

	// Build custom environment variables specific to add-reviewer
	var customEnvVars []string
	// Pass the allowed reviewers list (empty string if no restrictions)
	allowedReviewersStr := strings.Join(allowedReviewers, ",")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_REVIEWERS_ALLOWED: %q\n", allowedReviewersStr))
	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_REVIEWERS_MAX_COUNT: %d\n", maxCount))

	// Pass the target configuration
	if data.SafeOutputs.AddReviewer != nil && data.SafeOutputs.AddReviewer.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_REVIEWERS_TARGET: %q\n", data.SafeOutputs.AddReviewer.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.AddReviewer.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"reviewers_added": "${{ steps.add_reviewer.outputs.reviewers_added }}",
	}

	var jobCondition = BuildSafeOutputType("add_reviewer")
	if data.SafeOutputs.AddReviewer.Target == "" {
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
		Token:          data.SafeOutputs.AddReviewer.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.AddReviewer.TargetRepoSlug,
	})
}

// parseAddReviewerConfig handles add-reviewer configuration
func (c *Compiler) parseAddReviewerConfig(outputMap map[string]any) *AddReviewerConfig {
	if configData, exists := outputMap["add-reviewer"]; exists {
		addReviewerConfig := &AddReviewerConfig{}
		addReviewerConfig.Max = 3 // Default max is 3

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

			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := max.(int); ok {
					addReviewerConfig.Max = maxInt
				}
			}

			// Parse github-token
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					addReviewerConfig.GitHubToken = tokenStr
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					addReviewerConfig.Target = targetStr
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			addReviewerConfig.TargetRepoSlug = targetRepoSlug
		} else {
			// If configData is nil or not a map (e.g., "add-reviewer:" with no value),
			// still set the default max
			addReviewerConfig.Max = 3
		}

		return addReviewerConfig
	}

	return nil
}
