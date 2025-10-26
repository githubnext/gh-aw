package workflow

import (
	"regexp"
	"strings"
)

var (
	// Regular expressions for identifier conversion (used in ConvertToIdentifier)
	identifierNonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]`)
	identifierMultipleHyphens = regexp.MustCompile(`-+`)
)

// ConvertToIdentifier converts a workflow name to a valid identifier format
// suitable for use as a user agent string or similar identifier.
// It converts to lowercase, replaces spaces and underscores with hyphens,
// removes non-alphanumeric characters (except hyphens), and consolidates multiple hyphens.
// Returns "github-agentic-workflow" if the result would be empty.
func ConvertToIdentifier(name string) string {
	// Convert to lowercase
	identifier := strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	identifier = strings.ReplaceAll(identifier, " ", "-")
	identifier = strings.ReplaceAll(identifier, "_", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	identifier = identifierNonAlphanumeric.ReplaceAllString(identifier, "")

	// Consolidate multiple hyphens into single hyphen
	identifier = identifierMultipleHyphens.ReplaceAllString(identifier, "-")

	// Remove leading/trailing hyphens
	identifier = strings.Trim(identifier, "-")

	// Return default if empty after sanitization
	if identifier == "" {
		return "github-agentic-workflow"
	}

	return identifier
}
