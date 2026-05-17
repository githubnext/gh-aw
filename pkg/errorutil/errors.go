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
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found")
}
