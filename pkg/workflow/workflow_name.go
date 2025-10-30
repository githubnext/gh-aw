package workflow

import (
	"regexp"
)

var (
	// Regular expressions for identifier sanitization (used in SanitizeIdentifier)
	identifierNonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]`)
)

// SanitizeIdentifier sanitizes a workflow name to create a safe identifier
// suitable for use as a user agent string or similar context.
// It converts to lowercase, replaces spaces and underscores with hyphens,
// removes non-alphanumeric characters (except hyphens), and consolidates multiple hyphens.
// Returns "github-agentic-workflow" if the result would be empty.
func SanitizeIdentifier(name string) string {
	preReplacements := map[string]string{
		" ": "-",
		"_": "-",
	}
	identifier := sanitizeToKebabCase(name, preReplacements, identifierNonAlphanumeric, "", true)

	// Return default if empty after sanitization
	if identifier == "" {
		return "github-agentic-workflow"
	}

	return identifier
}
