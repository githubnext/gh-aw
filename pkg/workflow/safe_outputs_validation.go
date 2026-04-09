package workflow

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

var safeOutputsDomainsValidationLog = newValidationLogger("safe_outputs_domains")

// validateSafeOutputsAllowedDomains validates the allowed-domains configuration in safe-outputs.
// Supports ecosystem identifiers (e.g., "python", "node", "default-safe-outputs") like network.allowed.
func (c *Compiler) validateSafeOutputsAllowedDomains(config *SafeOutputsConfig) error {
	if config == nil || len(config.AllowedDomains) == 0 {
		return nil
	}

	safeOutputsDomainsValidationLog.Printf("Validating %d allowed domains", len(config.AllowedDomains))

	collector := NewErrorCollector(c.failFast)

	for i, domain := range config.AllowedDomains {
		// Skip ecosystem identifiers - they don't need domain pattern validation
		if isEcosystemIdentifier(domain) {
			safeOutputsDomainsValidationLog.Printf("Skipping ecosystem identifier: %s", domain)
			continue
		}

		if err := validateDomainPattern(domain); err != nil {
			wrappedErr := fmt.Errorf("safe-outputs.allowed-domains[%d]: %w", i, err)
			if returnErr := collector.Add(wrappedErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
		}
	}

	if err := collector.Error(); err != nil {
		safeOutputsDomainsValidationLog.Printf("Safe outputs allowed domains validation failed: %v", err)
		return err
	}

	safeOutputsDomainsValidationLog.Print("Safe outputs allowed domains validation passed")
	return nil
}

var safeOutputsTargetValidationLog = newValidationLogger("safe_outputs_target")

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
	if config.SubmitPullRequestReview != nil {
		configs = append(configs, targetConfig{"submit-pull-request-review", config.SubmitPullRequestReview.Target})
	}
	if config.ReplyToPullRequestReviewComment != nil {
		configs = append(configs, targetConfig{"reply-to-pull-request-review-comment", config.ReplyToPullRequestReviewComment.Target})
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
	if stringutil.IsPositiveInteger(target) {
		safeOutputsTargetValidationLog.Printf("Target for %s is a valid number: %s", configName, target)
		return nil
	}

	// Build a helpful suggestion based on the invalid value
	suggestion := ""
	if target == "event" || strings.Contains(target, "github.event") {
		suggestion = "\n\nDid you mean to use \"${{ github.event.issue.number }}\" instead of \"" + target + "\"?"
	}

	// Invalid target value
	return fmt.Errorf(
		"invalid target value for %s: %q\n\nValid target values are:\n  - \"triggering\" (default) - targets the triggering issue/PR/discussion\n  - \"*\" - targets any item specified in the output\n  - A positive integer (e.g., \"123\")\n  - A GitHub Actions expression (e.g., \"${{ github.event.issue.number }}\")%s",
		configName,
		target,
		suggestion,
	)
}

// isGitHubExpression checks if a string is a valid GitHub Actions expression
// A valid expression must have properly balanced ${{ and }} markers
func isGitHubExpression(s string) bool {
	// Must contain both opening and closing markers
	if !strings.Contains(s, "${{") || !strings.Contains(s, "}}") {
		return false
	}

	// Basic validation: opening marker must come before closing marker
	openIndex := strings.Index(s, "${{")
	closeIndex := strings.Index(s, "}}")

	// The closing marker must come after the opening marker
	// and there must be something between them
	return openIndex >= 0 && closeIndex > openIndex+3
}

var safeOutputsMaxValidationLog = newValidationLogger("safe_outputs_max")

// isInvalidMaxValue returns true if n is not a valid max field value.
// Valid values are positive integers (n > 0) or -1 (unlimited).
// Invalid values are 0 and negative integers except -1.
func isInvalidMaxValue(n int) bool {
	if n == -1 {
		return false // -1 = unlimited, explicitly allowed by spec
	}
	return n <= 0
}

// maxInvalidErrSuffix is the common suffix of max validation error messages.
const maxInvalidErrSuffix = "\n\nThe max field controls how many times this safe output can be triggered.\nProvide a positive integer (e.g., max: 1 or max: 5) or -1 for unlimited"

// validateSafeOutputsMax validates that all max fields in safe-outputs configs hold valid values.
// Valid values are positive integers (n > 0) or -1 (unlimited per spec).
// 0 and other negative values are rejected.
// GitHub Actions expressions (e.g. "${{ inputs.max }}") are not evaluable at compile time
// and are therefore skipped.
func validateSafeOutputsMax(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}

	safeOutputsMaxValidationLog.Print("Validating safe-outputs max fields")

	val := reflect.ValueOf(config).Elem()

	// Iterate over sorted field names for deterministic error reporting.
	sortedFieldNames := make([]string, 0, len(safeOutputFieldMapping))
	for fieldName := range safeOutputFieldMapping {
		sortedFieldNames = append(sortedFieldNames, fieldName)
	}
	sort.Strings(sortedFieldNames)

	// Validate max on all named safe output fields that embed BaseSafeOutputConfig
	for _, fieldName := range sortedFieldNames {
		toolName := safeOutputFieldMapping[fieldName]
		field := val.FieldByName(fieldName)
		if !field.IsValid() || field.IsNil() {
			continue
		}

		elem := field.Elem()
		baseCfgField := elem.FieldByName("BaseSafeOutputConfig")
		if !baseCfgField.IsValid() {
			continue
		}

		maxField := baseCfgField.FieldByName("Max")
		if !maxField.IsValid() || maxField.IsNil() {
			continue
		}

		maxPtr, ok := maxField.Interface().(*string)
		if !ok || maxPtr == nil || isExpressionString(*maxPtr) {
			continue
		}

		n, err := strconv.Atoi(*maxPtr)
		if err != nil {
			continue
		}

		if isInvalidMaxValue(n) {
			toolDisplayName := strings.ReplaceAll(toolName, "_", "-")
			safeOutputsMaxValidationLog.Printf("Invalid max value %d for %s", n, toolDisplayName)
			return fmt.Errorf(
				"safe-outputs.%s: max must be a positive integer or -1 (unlimited), got %d%s",
				toolDisplayName, n, maxInvalidErrSuffix,
			)
		}
	}

	// Validate max on dispatch_repository tools (different structure: map of tools).
	// Use sorted tool names for deterministic error reporting.
	if config.DispatchRepository != nil {
		sortedToolNames := make([]string, 0, len(config.DispatchRepository.Tools))
		for toolName := range config.DispatchRepository.Tools {
			sortedToolNames = append(sortedToolNames, toolName)
		}
		sort.Strings(sortedToolNames)

		for _, toolName := range sortedToolNames {
			tool := config.DispatchRepository.Tools[toolName]
			if tool == nil || tool.Max == nil || isExpressionString(*tool.Max) {
				continue
			}

			n, err := strconv.Atoi(*tool.Max)
			if err != nil {
				continue
			}

			if isInvalidMaxValue(n) {
				safeOutputsMaxValidationLog.Printf("Invalid max value %d for dispatch_repository tool %s", n, toolName)
				return fmt.Errorf(
					"safe-outputs.dispatch_repository.%s: max must be a positive integer or -1 (unlimited), got %d%s",
					toolName, n, maxInvalidErrSuffix,
				)
			}
		}
	}

	safeOutputsMaxValidationLog.Print("Safe-outputs max fields validation passed")
	return nil
}
