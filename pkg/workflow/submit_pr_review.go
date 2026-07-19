package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
)

var submitPRReviewLog = logger.New("workflow:submit_pr_review")

// SubmitPullRequestReviewConfig holds configuration for submitting a GitHub pull request review
// This works in conjunction with create-pull-request-review-comment: all review comments
// are collected and submitted as a single PR review with the configured event type.
// If this safe output type is not configured, review comments default to event: "COMMENT".
type SubmitPullRequestReviewConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
	Footer                 *string  `yaml:"footer,omitempty"`                  // Controls when to show footer in PR review body: "always" (default), "none", or "if-body" (only when review has body text)
	AllowedEvents          []string `yaml:"allowed-events,omitempty"`          // Optional list of allowed review event types: APPROVE, COMMENT, REQUEST_CHANGES. If omitted, all event types are allowed.
	SupersedeOlderReviews  bool     `yaml:"supersede-older-reviews,omitempty"` // When true, dismisses older same-workflow REQUEST_CHANGES reviews after a replacement review is posted.
}

// parseSubmitPullRequestReviewConfig handles submit-pull-request-review configuration
func (c *Compiler) parseSubmitPullRequestReviewConfig(outputMap map[string]any) *SubmitPullRequestReviewConfig {
	configData, exists := outputMap["submit-pull-request-review"]
	if !exists {
		submitPRReviewLog.Printf("Configuration not found")
		return nil
	}
	submitPRReviewLog.Printf("Parsing submit PR review configuration")
	config := &SubmitPullRequestReviewConfig{}
	configMap, ok := configData.(map[string]any)
	if !ok {
		config.Max = defaultIntStr(1)
		return config
	}
	if !c.applySubmitPullRequestReviewConfig(configMap, config) {
		return nil
	}
	submitPRReviewLog.Printf("Parsed submit-pull-request-review config: max=%d, target=%s, target_repo=%s, allowed_events=%v, supersede_older_reviews=%t", templatableIntValue(config.Max), config.Target, config.TargetRepoSlug, config.AllowedEvents, config.SupersedeOlderReviews)
	return config
}

func (c *Compiler) applySubmitPullRequestReviewConfig(configMap map[string]any, config *SubmitPullRequestReviewConfig) bool {
	c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)
	if target, exists := configMap["target"]; exists {
		if targetStr, ok := target.(string); ok {
			config.Target = targetStr
		}
	}
	targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
	if isInvalid {
		return false
	}
	config.TargetRepoSlug = targetRepoSlug
	config.AllowedRepos = ParseStringArrayFromConfig(configMap, "allowed-repos", submitPRReviewLog)
	parseSubmitReviewFooter(configMap, config)
	if !parseSubmitReviewAllowedEvents(configMap, config) {
		return false
	}
	parseSupersedeOlderReviews(configMap, config)
	return true
}

func parseSubmitReviewFooter(configMap map[string]any, config *SubmitPullRequestReviewConfig) {
	footer, exists := configMap["footer"]
	if !exists {
		return
	}
	switch f := footer.(type) {
	case string:
		if f == "always" || f == "none" || f == "if-body" {
			config.Footer = &f
			submitPRReviewLog.Printf("Footer control: %s", f)
		} else {
			submitPRReviewLog.Printf("Invalid footer value: %s (must be 'always', 'none', or 'if-body')", f)
		}
	case bool:
		footerStr := "none"
		if f {
			footerStr = "always"
		}
		config.Footer = &footerStr
		submitPRReviewLog.Printf("Footer control (mapped from bool): %s", footerStr)
	}
}

func parseSubmitReviewAllowedEvents(configMap map[string]any, config *SubmitPullRequestReviewConfig) bool {
	allowedEvents, exists := configMap["allowed-events"]
	if !exists {
		return true
	}
	eventsSlice, ok := allowedEvents.([]any)
	if !ok {
		submitPRReviewLog.Printf("Invalid allowed-events configuration: must be a list of review event types")
		return false
	}
	validEvents := map[string]struct{}{"APPROVE": {}, "COMMENT": {}, "REQUEST_CHANGES": {}}
	for _, e := range eventsSlice {
		if eventStr, ok := e.(string); ok {
			upper := strings.ToUpper(eventStr)
			if setutil.Contains(validEvents, upper) {
				config.AllowedEvents = append(config.AllowedEvents, upper)
			} else {
				submitPRReviewLog.Printf("Ignoring invalid allowed-events value: %s", eventStr)
			}
		}
	}
	if len(config.AllowedEvents) == 0 {
		submitPRReviewLog.Printf("Invalid allowed-events configuration: at least one valid event type is required when allowed-events is specified")
		return false
	}
	return true
}

func parseSupersedeOlderReviews(configMap map[string]any, config *SubmitPullRequestReviewConfig) {
	supersedeOlderReviews, exists := configMap["supersede-older-reviews"]
	if !exists {
		return
	}
	if supersedeEnabled, ok := supersedeOlderReviews.(bool); ok {
		config.SupersedeOlderReviews = supersedeEnabled
	} else {
		submitPRReviewLog.Printf("Invalid supersede-older-reviews value: must be a boolean")
	}
}
