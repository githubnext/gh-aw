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
