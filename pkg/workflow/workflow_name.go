package workflow

import (
	"regexp"
	"strings"
)

var (
	// Regular expressions for workflow name sanitization
	identifierNonAlphanumeric  = regexp.MustCompile(`[^a-z0-9-]`)
	identifierMultipleHyphens  = regexp.MustCompile(`-+`)
)

// SanitizeWorkflowName sanitizes a workflow name for use in artifact names and file paths
// Removes or replaces characters that are invalid in YAML artifact names or filesystem paths
func SanitizeWorkflowName(name string) string {
	// Replace colons, slashes, and other problematic characters with hyphens
	sanitized := strings.ReplaceAll(name, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	// Remove any remaining special characters that might cause issues
	sanitized = strings.Map(func(r rune) rune {
		// Allow alphanumeric, hyphens, underscores, and periods
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, sanitized)
	return sanitized
}

// ConvertToIdentifier converts a workflow name to a valid identifier format
// by converting to lowercase and replacing spaces with hyphens
func ConvertToIdentifier(name string) string {
	// Convert to lowercase
	identifier := strings.ToLower(name)
	// Replace spaces and other common separators with hyphens
	identifier = strings.ReplaceAll(identifier, " ", "-")
	identifier = strings.ReplaceAll(identifier, "_", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	identifier = identifierNonAlphanumeric.ReplaceAllString(identifier, "")
	// Remove any double hyphens that might have been created
	identifier = identifierMultipleHyphens.ReplaceAllString(identifier, "-")
	// Remove leading/trailing hyphens
	identifier = strings.Trim(identifier, "-")

	// If the result is empty, return a default identifier
	if identifier == "" {
		identifier = "github-agentic-workflow"
	}

	return identifier
}
