package workflow

// SanitizeIdentifier sanitizes a workflow name to create a safe identifier
// suitable for use as a user agent string or similar context.
// It converts to lowercase, replaces spaces and underscores with hyphens,
// removes non-alphanumeric characters (except hyphens), and consolidates multiple hyphens.
// Returns "github-agentic-workflow" if the result would be empty.
//
// This function uses the unified SanitizeName function with options configured
// to trim leading/trailing hyphens, and return a default value for empty results.
// Hyphens are preserved by default in SanitizeName, not via PreserveSpecialChars.
func SanitizeIdentifier(name string) string {
	return SanitizeName(name, &SanitizeOptions{
		PreserveSpecialChars: []rune{},
		TrimHyphens:          true,
		DefaultValue:         "github-agentic-workflow",
	})
}
