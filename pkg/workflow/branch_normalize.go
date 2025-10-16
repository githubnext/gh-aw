package workflow

import (
	"regexp"
	"strings"
)

// normalizeBranchName converts a branch name to be valid for git
// Valid characters: alphanumeric, -, _, /, .
// Max length: 128 characters
// The function removes/replaces invalid characters and truncates to max length
func normalizeBranchName(branchName string) string {
	// Replace any sequence of invalid characters with a single dash
	// Valid characters are: a-z, A-Z, 0-9, -, _, /, .
	invalidCharsRegex := regexp.MustCompile(`[^a-zA-Z0-9\-_/.]+`)
	normalized := invalidCharsRegex.ReplaceAllString(branchName, "-")

	// Remove leading and trailing dashes
	normalized = strings.Trim(normalized, "-")

	// Truncate to max 128 characters
	if len(normalized) > 128 {
		normalized = normalized[:128]
	}

	// Ensure it doesn't end with a dash after truncation
	normalized = strings.TrimRight(normalized, "-")

	return normalized
}
