package workflow

import (
	"regexp"
	"strings"
)

var sanitizeNamePattern = regexp.MustCompile(`[^a-z0-9._-]+`)
var multipleHyphens = regexp.MustCompile(`-+`)

// SanitizeOptions configures the behavior of the SanitizeName function.
type SanitizeOptions struct {
	// PreserveSpecialChars is a list of special characters to preserve during sanitization.
	// Common characters include '.', '_'. If nil or empty, only alphanumeric and hyphens are preserved.
	PreserveSpecialChars []rune

	// TrimHyphens controls whether leading and trailing hyphens are removed from the result.
	// When true, hyphens at the start and end of the sanitized name are trimmed.
	TrimHyphens bool

	// DefaultValue is returned when the sanitized name is empty after all transformations.
	// If empty string, no default is applied.
	DefaultValue string
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

// SanitizeName sanitizes a string for use as an identifier, file name, or similar context.
// It provides configurable behavior through the SanitizeOptions parameter.
//
// The function performs the following transformations:
//   - Converts to lowercase
//   - Replaces common separators (colons, slashes, backslashes, spaces) with hyphens
//   - Replaces underscores with hyphens unless preserved in opts.PreserveSpecialChars
//   - Removes or replaces characters based on opts.PreserveSpecialChars
//   - Consolidates multiple consecutive hyphens into a single hyphen
//   - Optionally trims leading/trailing hyphens (controlled by opts.TrimHyphens)
//   - Returns opts.DefaultValue if the result is empty (controlled by opts.DefaultValue)
//
// Example:
//
//	// Preserve dots and underscores (like SanitizeWorkflowName)
//	opts := &SanitizeOptions{
//	    PreserveSpecialChars: []rune{'.', '_'},
//	}
//	SanitizeName("My.Workflow_Name", opts) // returns "my.workflow_name"
//
//	// Trim hyphens and use default (like SanitizeIdentifier)
//	opts := &SanitizeOptions{
//	    TrimHyphens:  true,
//	    DefaultValue: "default-name",
//	}
//	SanitizeName("@@@", opts) // returns "default-name"
func SanitizeName(name string, opts *SanitizeOptions) string {
	// Handle nil options
	if opts == nil {
		opts = &SanitizeOptions{}
	}

	// Convert to lowercase
	result := strings.ToLower(name)

	// Replace common separators with hyphens
	result = strings.ReplaceAll(result, ":", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, " ", "-")

	// Check if underscores should be preserved
	preserveUnderscore := false
	for _, char := range opts.PreserveSpecialChars {
		if char == '_' {
			preserveUnderscore = true
			break
		}
	}

	// Replace underscores with hyphens if not preserved
	if !preserveUnderscore {
		result = strings.ReplaceAll(result, "_", "-")
	}

	// Build character preservation pattern based on options
	preserveChars := "a-z0-9-" // Always preserve alphanumeric and hyphens
	if len(opts.PreserveSpecialChars) > 0 {
		for _, char := range opts.PreserveSpecialChars {
			// Escape special regex characters
			switch char {
			case '.', '_':
				preserveChars += string(char)
			}
		}
	}

	// Create pattern for characters to remove/replace
	pattern := regexp.MustCompile(`[^` + preserveChars + `]+`)

	// Replace unwanted characters with hyphens or empty based on context
	if len(opts.PreserveSpecialChars) > 0 {
		// Replace with hyphens (SanitizeWorkflowName behavior)
		result = pattern.ReplaceAllString(result, "-")
	} else {
		// Remove completely (SanitizeIdentifier behavior)
		result = pattern.ReplaceAllString(result, "")
	}

	// Consolidate multiple consecutive hyphens into a single hyphen
	result = multipleHyphens.ReplaceAllString(result, "-")

	// Optionally trim leading/trailing hyphens
	if opts.TrimHyphens {
		result = strings.Trim(result, "-")
	}

	// Return default value if result is empty
	if result == "" && opts.DefaultValue != "" {
		return opts.DefaultValue
	}

	return result
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
