package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeOutputsDomainsValidationLog = logger.New("workflow:safe_outputs_domains_validation")

// domainPattern validates domain patterns including wildcards
// Valid patterns:
// - Plain domains: github.com, api.github.com
// - Wildcard domains: *.github.com
// Invalid patterns:
// - Multiple wildcards: *.*.github.com
// - Wildcard not at start: github.*.com
// - Empty or malformed domains
var domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// validateSafeOutputsAllowedDomains validates the allowed-domains configuration in safe-outputs
func validateSafeOutputsAllowedDomains(config *SafeOutputsConfig) error {
	if config == nil || len(config.AllowedDomains) == 0 {
		return nil
	}

	safeOutputsDomainsValidationLog.Printf("Validating %d allowed domains", len(config.AllowedDomains))

	for i, domain := range config.AllowedDomains {
		if err := validateDomainPattern(domain); err != nil {
			return fmt.Errorf("safe-outputs.allowed-domains[%d]: %w", i, err)
		}
	}

	return nil
}

// validateDomainPattern validates a single domain pattern
func validateDomainPattern(domain string) error {
	// Check for empty domain
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Check for wildcard-only pattern
	if domain == "*" {
		return fmt.Errorf("wildcard-only domain '*' is not allowed, use a specific wildcard pattern like '*.example.com'")
	}

	// Check for wildcard without base domain (must be done before regex)
	if domain == "*." {
		return fmt.Errorf("wildcard pattern '%s' must have a domain after '*.' (e.g., '*.example.com')", domain)
	}

	// Check for multiple wildcards
	if strings.Count(domain, "*") > 1 {
		return fmt.Errorf("domain pattern '%s' contains multiple wildcards, only one wildcard at the start is allowed (e.g., '*.example.com')", domain)
	}

	// Check for wildcard not at the start
	if strings.Contains(domain, "*") && !strings.HasPrefix(domain, "*.") {
		return fmt.Errorf("domain pattern '%s' has wildcard in invalid position, wildcard must be at the start followed by a dot (e.g., '*.example.com')", domain)
	}

	// Additional validation for wildcard patterns
	if strings.HasPrefix(domain, "*.") {
		baseDomain := domain[2:] // Remove "*."
		if baseDomain == "" {
			return fmt.Errorf("wildcard pattern '%s' must have a domain after '*.' (e.g., '*.example.com')", domain)
		}
		// Ensure the base domain doesn't start with a dot
		if strings.HasPrefix(baseDomain, ".") {
			return fmt.Errorf("wildcard pattern '%s' has invalid format, use '*.example.com' instead of '*.*.example.com'", domain)
		}
	}

	// Validate domain pattern format
	if !domainPattern.MatchString(domain) {
		// Provide specific error messages for common issues
		if strings.HasSuffix(domain, ".") {
			return fmt.Errorf("domain pattern '%s' cannot end with a dot", domain)
		}
		if strings.Contains(domain, "..") {
			return fmt.Errorf("domain pattern '%s' cannot contain consecutive dots", domain)
		}
		if strings.HasPrefix(domain, ".") && !strings.HasPrefix(domain, "*.") {
			return fmt.Errorf("domain pattern '%s' cannot start with a dot (except for wildcard patterns like '*.example.com')", domain)
		}
		// Check for invalid characters
		for _, char := range domain {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '.' || char == '*') {
				return fmt.Errorf("domain pattern '%s' contains invalid character '%c', only alphanumeric, hyphens, dots, and wildcards are allowed", domain, char)
			}
		}
		return fmt.Errorf("domain pattern '%s' is not a valid domain format", domain)
	}

	return nil
}
