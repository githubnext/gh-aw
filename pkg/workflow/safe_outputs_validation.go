package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

var safeOutputsDomainsValidationLog = newValidationLogger("safe_outputs_domains")

// validateNetworkAllowedDomains validates the allowed domains in network configuration
func (c *Compiler) validateNetworkAllowedDomains(network *NetworkPermissions) error {
	if network == nil || len(network.Allowed) == 0 {
		return nil
	}

	safeOutputsDomainsValidationLog.Printf("Validating %d network allowed domains", len(network.Allowed))

	collector := NewErrorCollector(c.failFast)

	for i, domain := range network.Allowed {
		// "*" means allow all traffic - skip validation
		if domain == "*" {
			safeOutputsDomainsValidationLog.Print("Skipping allow-all wildcard '*'")
			continue
		}

		// Skip ecosystem identifiers - they don't need domain pattern validation
		if isEcosystemIdentifier(domain) {
			safeOutputsDomainsValidationLog.Printf("Skipping ecosystem identifier: %s", domain)
			continue
		}

		if err := validateDomainPattern(domain); err != nil {
			wrappedErr := fmt.Errorf("network.allowed[%d]: %w", i, err)
			if returnErr := collector.Add(wrappedErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
		}
	}

	if err := collector.Error(); err != nil {
		safeOutputsDomainsValidationLog.Printf("Network allowed domains validation failed: %v", err)
		return err
	}

	safeOutputsDomainsValidationLog.Print("Network allowed domains validation passed")
	return nil
}

// isEcosystemIdentifierPattern matches valid ecosystem identifiers like "defaults", "node", "dev-tools"
var isEcosystemIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// isEcosystemIdentifier checks if a domain string is actually an ecosystem identifier
func isEcosystemIdentifier(domain string) bool {
	// Ecosystem identifiers are simple lowercase alphanumeric identifiers with optional hyphens
	// like "defaults", "node", "python", "dev-tools", "default-safe-outputs".
	// They don't contain dots, protocol prefixes, spaces, wildcards, or other special characters.
	return isEcosystemIdentifierPattern.MatchString(domain)
}

// domainPattern validates domain patterns including wildcards
// Valid patterns:
// - Plain domains: github.com, api.github.com
// - Wildcard domains: *.github.com
// Invalid patterns:
// - Multiple wildcards: *.*.github.com
// - Wildcard not at start: github.*.com
// - Empty or malformed domains
var domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

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

// validateDomainPattern validates a single domain pattern
func validateDomainPattern(domain string) error {
	// Check for empty domain
	if domain == "" {
		return NewValidationError(
			"domain",
			"",
			"domain cannot be empty",
			"Provide a valid domain name. Examples:\n  - Plain domain: 'github.com'\n  - Wildcard: '*.github.com'\n  - With protocol: 'https://api.github.com'",
		)
	}

	// Check for invalid protocol prefixes
	// Only http:// and https:// are allowed
	if strings.Contains(domain, "://") {
		if !strings.HasPrefix(domain, "https://") && !strings.HasPrefix(domain, "http://") {
			return NewValidationError(
				"domain",
				domain,
				"domain pattern has invalid protocol, only 'http://' and 'https://' are allowed",
				"Remove the invalid protocol or use 'http://' or 'https://'. Examples:\n  - 'https://api.github.com'\n  - 'http://example.com'\n  - 'github.com' (no protocol)",
			)
		}
	}

	// Strip protocol prefix if present (http:// or https://)
	// This allows protocol-specific domain filtering
	domainWithoutProtocol := domain
	if after, ok := strings.CutPrefix(domain, "https://"); ok {
		domainWithoutProtocol = after
	} else if after, ok := strings.CutPrefix(domain, "http://"); ok {
		domainWithoutProtocol = after
	}

	// Check for wildcard-only pattern
	if domainWithoutProtocol == "*" {
		return NewValidationError(
			"domain",
			domain,
			"wildcard-only domain '*' is not allowed",
			"Use a specific wildcard pattern with a base domain. Examples:\n  - '*.example.com'\n  - '*.github.com'\n  - 'https://*.api.example.com'",
		)
	}

	// Check for wildcard without base domain (must be done before regex)
	if domainWithoutProtocol == "*." {
		return NewValidationError(
			"domain",
			domain,
			"wildcard pattern must have a domain after '*.'",
			"Add a base domain after the wildcard. Examples:\n  - '*.example.com'\n  - '*.github.com'\n  - 'https://*.api.example.com'",
		)
	}

	// Check for multiple wildcards
	if strings.Count(domainWithoutProtocol, "*") > 1 {
		return NewValidationError(
			"domain",
			domain,
			"domain pattern contains multiple wildcards, only one wildcard at the start is allowed",
			"Use a single wildcard at the start of the domain. Examples:\n  - '*.example.com' ✓\n  - '*.*.example.com' ✗ (multiple wildcards)\n  - 'https://*.github.com' ✓",
		)
	}

	// Check for wildcard not at the start (in the domain part)
	if strings.Contains(domainWithoutProtocol, "*") && !strings.HasPrefix(domainWithoutProtocol, "*.") {
		return NewValidationError(
			"domain",
			domain,
			"wildcard must be at the start followed by a dot",
			"Move the wildcard to the beginning of the domain. Examples:\n  - '*.example.com' ✓\n  - 'example.*.com' ✗ (wildcard in middle)\n  - 'https://*.github.com' ✓",
		)
	}

	// Additional validation for wildcard patterns
	if strings.HasPrefix(domainWithoutProtocol, "*.") {
		baseDomain := domainWithoutProtocol[2:] // Remove "*."
		if baseDomain == "" {
			return NewValidationError(
				"domain",
				domain,
				"wildcard pattern must have a domain after '*.'",
				"Add a base domain after the wildcard. Examples:\n  - '*.example.com'\n  - '*.github.com'\n  - 'https://*.api.example.com'",
			)
		}
		// Ensure the base domain doesn't start with a dot
		if strings.HasPrefix(baseDomain, ".") {
			return NewValidationError(
				"domain",
				domain,
				"wildcard pattern has invalid format (extra dot after wildcard)",
				"Use correct wildcard format. Examples:\n  - '*.example.com' ✓\n  - '*.*.example.com' ✗ (extra dot)\n  - 'https://*.github.com' ✓",
			)
		}
	}

	// Validate domain pattern format (without protocol)
	if !domainPattern.MatchString(domainWithoutProtocol) {
		// Provide specific error messages for common issues
		if strings.HasSuffix(domainWithoutProtocol, ".") {
			return NewValidationError(
				"domain",
				domain,
				"domain pattern cannot end with a dot",
				"Remove the trailing dot from the domain. Examples:\n  - 'example.com' ✓\n  - 'example.com.' ✗\n  - '*.github.com' ✓",
			)
		}
		if strings.Contains(domainWithoutProtocol, "..") {
			return NewValidationError(
				"domain",
				domain,
				"domain pattern cannot contain consecutive dots",
				"Remove extra dots from the domain. Examples:\n  - 'api.example.com' ✓\n  - 'api..example.com' ✗\n  - 'sub.api.example.com' ✓",
			)
		}
		if strings.HasPrefix(domainWithoutProtocol, ".") && !strings.HasPrefix(domainWithoutProtocol, "*.") {
			return NewValidationError(
				"domain",
				domain,
				"domain pattern cannot start with a dot (except for wildcard patterns)",
				"Remove the leading dot or use a wildcard. Examples:\n  - 'example.com' ✓\n  - '.example.com' ✗\n  - '*.example.com' ✓",
			)
		}
		// Check for invalid characters (in the domain part, not protocol)
		for _, char := range domainWithoutProtocol {
			if (char < 'a' || char > 'z') &&
				(char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') &&
				char != '-' && char != '.' && char != '*' {
				return NewValidationError(
					"domain",
					domain,
					fmt.Sprintf("domain pattern contains invalid character '%c'", char),
					"Use only alphanumeric characters, hyphens, dots, and wildcards. Examples:\n  - 'api-v2.example.com' ✓\n  - 'api_v2.example.com' ✗ (underscore not allowed)\n  - '*.github.com' ✓",
				)
			}
		}
		return NewValidationError(
			"domain",
			domain,
			"domain pattern is not a valid domain format",
			"Use a valid domain format. Examples:\n  - Plain: 'github.com', 'api.example.com'\n  - Wildcard: '*.github.com', '*.example.com'\n  - With protocol: 'https://api.github.com'",
		)
	}

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
