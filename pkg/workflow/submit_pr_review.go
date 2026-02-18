package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var submitPRReviewLog = logger.New("workflow:submit_pr_review")

// SubmitPullRequestReviewConfig holds configuration for submitting a GitHub pull request review
// This works in conjunction with create-pull-request-review-comment: all review comments
// are collected and submitted as a single PR review with the configured event type.
// If this safe output type is not configured, review comments default to event: "COMMENT".
type SubmitPullRequestReviewConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string  `yaml:"target,omitempty"` // Target PR: "triggering" (default), "*" (use message.pull_request_number), or explicit number e.g. ${{ github.event.inputs.pr_number }}
	Footer               *string `yaml:"footer,omitempty"` // Controls when to show footer in PR review body: "always" (default), "none", or "if-body" (only when review has body text)
}

// parseSubmitPullRequestReviewConfig handles submit-pull-request-review configuration
func (c *Compiler) parseSubmitPullRequestReviewConfig(outputMap map[string]any) *SubmitPullRequestReviewConfig {
	if _, exists := outputMap["submit-pull-request-review"]; !exists {
		submitPRReviewLog.Printf("Configuration not found")
		return nil
	}

	submitPRReviewLog.Printf("Parsing submit PR review configuration")

	configData := outputMap["submit-pull-request-review"]
	config := &SubmitPullRequestReviewConfig{}

	if configMap, ok := configData.(map[string]any); ok {
		// Parse common base fields with default max of 1
		c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)

		// Parse target (same semantics as add-comment / create-pull-request-review-comment)
		if target, exists := configMap["target"]; exists {
			if targetStr, ok := target.(string); ok {
				config.Target = targetStr
			}
		}

		// Parse footer configuration (string: "always"/"none"/"if-body", or bool for backward compat)
		if footer, exists := configMap["footer"]; exists {
			switch f := footer.(type) {
			case string:
				// Validate string values: "always", "none", "if-body"
				if f == "always" || f == "none" || f == "if-body" {
					config.Footer = &f
					submitPRReviewLog.Printf("Footer control: %s", f)
				} else {
					submitPRReviewLog.Printf("Invalid footer value: %s (must be 'always', 'none', or 'if-body')", f)
				}
			case bool:
				// Map boolean to string: true -> "always", false -> "none"
				var footerStr string
				if f {
					footerStr = "always"
				} else {
					footerStr = "none"
				}
				config.Footer = &footerStr
				submitPRReviewLog.Printf("Footer control (mapped from bool): %s", footerStr)
			}
		}
	} else {
		// If configData is nil or not a map, set the default max
		config.Max = 1
	}

	return config
}
