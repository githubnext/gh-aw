package workflow

import (
	"regexp"
	"strings"
)

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

// normalizeBranchName normalizes a branch name by removing all characters
// that are not a-z, A-Z, 0-9, -, _, or /
// This ensures the branch name is safe for git operations
func normalizeBranchName(branchName string) string {
	// Remove all characters that are not alphanumeric, dash, underscore, or forward slash
	// This matches: [^a-zA-Z0-9\-_/]
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_/]`)
	normalized := re.ReplaceAllString(branchName, "")

	// Clean up consecutive slashes
	normalized = regexp.MustCompile(`/+`).ReplaceAllString(normalized, "/")

	// Remove leading/trailing slashes and dashes
	normalized = strings.Trim(normalized, "/-")

	return normalized
}
