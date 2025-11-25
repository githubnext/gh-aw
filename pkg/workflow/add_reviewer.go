package workflow

import (
	"fmt"
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

	cfg := data.SafeOutputs.AddReviewer

	// Determine max count (default is 3)
	maxCount := 3
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Use the shared list-based builder
	return c.buildListSafeOutputJob(data, ListSafeOutputJobParams{
		JobName:        "add_reviewer",
		StepName:       "Add Reviewers",
		StepID:         "add_reviewer",
		MainJobName:    mainJobName,
		EnvPrefix:      "REVIEWERS",
		AllowedItems:   cfg.Reviewers,
		MaxCount:       maxCount,
		Target:         cfg.Target,
		TargetRepoSlug: cfg.TargetRepoSlug,
		Script:         getAddReviewerScript(),
		Permissions:    NewPermissionsContentsReadPRWrite(),
		Token:          cfg.GitHubToken,
		OutputKey:      "reviewers_added",
		// Reviewers can only be added to PRs
		TriggeringContextConditions: []ConditionNode{
			BuildPropertyAccess("github.event.pull_request.number"),
		},
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
				if maxInt, ok := parseIntValue(max); ok {
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
