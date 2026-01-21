package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeOutputsTargetValidationLog = logger.New("workflow:safe_outputs_target_validation")

// validateSafeOutputsTarget validates target fields in all safe-outputs configurations
// Valid target values:
//   - "" (empty/default) - uses "triggering" behavior
//   - "triggering" - targets the triggering issue/PR/discussion
//   - "*" - targets any item specified in the output
//   - A positive integer as a string (e.g., "123")
//   - A GitHub Actions expression (e.g., "${{ github.event.issue.number }}")
func validateSafeOutputsTarget(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}

	safeOutputsTargetValidationLog.Print("Validating safe-outputs target fields")

	// List of configs to validate - each with a name for error messages
	type targetConfig struct {
		name   string
		target string
	}

	var configs []targetConfig

	// Collect all target fields from various safe-output configurations
	if config.UpdateIssues != nil {
		configs = append(configs, targetConfig{"update-issue", config.UpdateIssues.Target})
	}
	if config.UpdateDiscussions != nil {
		configs = append(configs, targetConfig{"update-discussion", config.UpdateDiscussions.Target})
	}
	if config.UpdatePullRequests != nil {
		configs = append(configs, targetConfig{"update-pull-request", config.UpdatePullRequests.Target})
	}
	if config.CloseIssues != nil {
		configs = append(configs, targetConfig{"close-issue", config.CloseIssues.Target})
	}
	if config.CloseDiscussions != nil {
		configs = append(configs, targetConfig{"close-discussion", config.CloseDiscussions.Target})
	}
	if config.ClosePullRequests != nil {
		configs = append(configs, targetConfig{"close-pull-request", config.ClosePullRequests.Target})
	}
	if config.AddLabels != nil {
		configs = append(configs, targetConfig{"add-labels", config.AddLabels.Target})
	}
	if config.RemoveLabels != nil {
		configs = append(configs, targetConfig{"remove-labels", config.RemoveLabels.Target})
	}
	if config.AddReviewer != nil {
		configs = append(configs, targetConfig{"add-reviewer", config.AddReviewer.Target})
	}
	if config.AssignMilestone != nil {
		configs = append(configs, targetConfig{"assign-milestone", config.AssignMilestone.Target})
	}
	if config.AssignToAgent != nil {
		configs = append(configs, targetConfig{"assign-to-agent", config.AssignToAgent.Target})
	}
	if config.AssignToUser != nil {
		configs = append(configs, targetConfig{"assign-to-user", config.AssignToUser.Target})
	}
	if config.LinkSubIssue != nil {
		configs = append(configs, targetConfig{"link-sub-issue", config.LinkSubIssue.Target})
	}
	if config.HideComment != nil {
		configs = append(configs, targetConfig{"hide-comment", config.HideComment.Target})
	}
	if config.MarkPullRequestAsReadyForReview != nil {
		configs = append(configs, targetConfig{"mark-pull-request-as-ready-for-review", config.MarkPullRequestAsReadyForReview.Target})
	}
	if config.AddComments != nil {
		configs = append(configs, targetConfig{"add-comment", config.AddComments.Target})
	}
	if config.CreatePullRequestReviewComments != nil {
		configs = append(configs, targetConfig{"create-pull-request-review-comment", config.CreatePullRequestReviewComments.Target})
	}
	if config.PushToPullRequestBranch != nil {
		configs = append(configs, targetConfig{"push-to-pull-request-branch", config.PushToPullRequestBranch.Target})
	}

	// Validate each target field
	for _, cfg := range configs {
		if err := validateTargetValue(cfg.name, cfg.target); err != nil {
			return err
		}
	}

	safeOutputsTargetValidationLog.Printf("Validated %d target fields", len(configs))
	return nil
}

// validateTargetValue validates a single target value
func validateTargetValue(configName, target string) error {
	// Empty or "triggering" are always valid
	if target == "" || target == "triggering" {
		return nil
	}

	// "*" is valid (any item)
	if target == "*" {
		return nil
	}

	// Check if it's a GitHub Actions expression
	if isGitHubExpression(target) {
		safeOutputsTargetValidationLog.Printf("Target for %s is a GitHub Actions expression", configName)
		return nil
	}

	// Check if it's a positive integer
	if isPositiveInteger(target) {
		safeOutputsTargetValidationLog.Printf("Target for %s is a valid number: %s", configName, target)
		return nil
	}

	// Invalid target value
	return fmt.Errorf(
		"invalid target value for %s: %q\n\nValid target values are:\n  - \"triggering\" (default) - targets the triggering issue/PR/discussion\n  - \"*\" - targets any item specified in the output\n  - A positive integer (e.g., \"123\")\n  - A GitHub Actions expression (e.g., \"${{ github.event.issue.number }}\")\n\nDid you mean to use \"${{ github.event.issue.number }}\" instead of \"event\"?",
		configName,
		target,
	)
}

// isGitHubExpression checks if a string contains a GitHub Actions expression
func isGitHubExpression(s string) bool {
	// GitHub Actions expressions are in the format ${{ ... }}
	return strings.Contains(s, "${{") && strings.Contains(s, "}}")
}

// isPositiveInteger checks if a string is a positive integer
func isPositiveInteger(s string) bool {
	// Must not be empty
	if s == "" {
		return false
	}

	// Must not have leading zeros (except "0" itself, but that's not positive)
	if len(s) > 1 && s[0] == '0' {
		return false
	}

	// Must be numeric and > 0
	num, err := strconv.ParseInt(s, 10, 64)
	return err == nil && num > 0
}
