package workflow

import (
	"regexp"
	"strings"
)

var sanitizeNamePattern = regexp.MustCompile(`[^a-z0-9._-]+`)
var multipleHyphens = regexp.MustCompile(`-+`)

// sanitizeToKebabCase is a helper function that converts a string to kebab-case
// with customizable character filtering and replacement rules.
//
// Parameters:
//   - input: the string to sanitize
//   - preReplacements: map of characters/strings to replace with hyphens before pattern matching
//   - allowedPattern: regex pattern for allowed characters (others are replaced/removed)
//   - replacementChar: what to replace disallowed characters with (typically "-" or "")
//   - trimHyphens: whether to trim leading and trailing hyphens
//
// Returns the sanitized kebab-case string
func sanitizeToKebabCase(input string, preReplacements map[string]string, allowedPattern *regexp.Regexp, replacementChar string, trimHyphens bool) string {
	result := strings.ToLower(input)

	// Apply pre-replacements (e.g., spaces, colons, slashes to hyphens)
	for old, new := range preReplacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Replace disallowed characters with the replacement character
	result = allowedPattern.ReplaceAllString(result, replacementChar)

	// Consolidate multiple consecutive hyphens into a single hyphen
	result = multipleHyphens.ReplaceAllString(result, "-")

	// Trim leading/trailing hyphens if requested
	if trimHyphens {
		result = strings.Trim(result, "-")
	}

	return result
}

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

// SortPermissionScopes sorts a slice of PermissionScope in place using bubble sort
func SortPermissionScopes(s []PermissionScope) {
	n := len(s)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if string(s[j]) > string(s[j+1]) {
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
	preReplacements := map[string]string{
		":":  "-",
		"\\": "-",
		"/":  "-",
		" ":  "-",
	}
	return sanitizeToKebabCase(name, preReplacements, sanitizeNamePattern, "-", false)
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
