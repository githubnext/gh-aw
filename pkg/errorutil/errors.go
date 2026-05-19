// Package errorutil provides shared helpers for classifying and inspecting errors
// returned by the GitHub API and gh CLI.
package errorutil

import "strings"

// IsNotFoundError reports whether err represents an HTTP 404 / "not found" response.
// It returns false when err is nil.
// The check is case-insensitive and matches both the numeric literal "404" and
// the phrase "not found", which covers all known forms returned by the GitHub API,
// the gh CLI, and the go-gh library.
func IsNotFoundError(err error) bool {
	return containsErrorSubstring(err, "404", "not found")
}

// IsForbiddenError reports whether err represents an HTTP 403 / "forbidden" response.
// It returns false when err is nil.
// The check is case-insensitive and matches both the numeric literal "403" and
// the phrase "forbidden", which covers known forms returned by the GitHub API
// and the gh CLI.
func IsForbiddenError(err error) bool {
	return containsErrorSubstring(err, "403", "forbidden")
}

// IsGoneError reports whether err represents an HTTP 410 / "gone" response.
// It returns false when err is nil.
// The check is case-insensitive and matches both the numeric literal "410" and
// the phrase "gone", which covers known forms returned by the GitHub API and
// the gh CLI when workflow logs have expired.
func IsGoneError(err error) bool {
	return containsErrorSubstring(err, "410", "gone")
}

func containsErrorSubstring(err error, substrings ...string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, substring := range substrings {
		if strings.Contains(msg, substring) {
			return true
		}
	}
	return false
}
