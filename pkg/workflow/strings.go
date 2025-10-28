package workflow

import (
	"regexp"
	"strings"
)

var sanitizeNamePattern = regexp.MustCompile(`[^a-z0-9._-]+`)
var multipleHyphens = regexp.MustCompile(`-+`)

// SortStrings sorts a slice of strings in place using bubble sort
func SortStrings(s []string) {
	n := len(s)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if s[j] > s[j+1] {
				s[j], s[j+1] = s[j+1], s[j]
			}
		}
	}
}

// SanitizeWorkflowName sanitizes a workflow name for use in artifact names and file paths.
// It converts the name to lowercase and replaces or removes characters that are invalid
// in YAML artifact names or filesystem paths.
//
// The function performs the following transformations:
//   - Converts to lowercase
//   - Replaces colons, slashes, backslashes, and spaces with hyphens
//   - Replaces any remaining special characters (except dots, underscores, and hyphens) with hyphens
//
// Example:
//
//	SanitizeWorkflowName("My Workflow: Test/Build") // returns "my-workflow--test-build"
func SanitizeWorkflowName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Replace any other non-alphanumeric characters (except . _ -) with "-"
	name = sanitizeNamePattern.ReplaceAllString(name, "-")

	// Consolidate multiple consecutive hyphens into a single hyphen
	name = multipleHyphens.ReplaceAllString(name, "-")

	return name
}

// ShortenCommand creates a short identifier for bash commands.
// It replaces newlines with spaces and truncates to 20 characters if needed.
func ShortenCommand(command string) string {
	// Take first 20 characters and remove newlines
	shortened := strings.ReplaceAll(command, "\n", " ")
	if len(shortened) > 20 {
		shortened = shortened[:20] + "..."
	}
	return shortened
}
