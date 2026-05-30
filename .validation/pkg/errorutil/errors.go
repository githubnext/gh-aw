// Package errorutil provides shared helpers for classifying and inspecting errors
// returned by the GitHub API and gh CLI.
package errorutil

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var errorutilLog = logger.New("errorutil:errors")

// IsNotFoundError reports whether err represents an HTTP 404 / "not found" response.
// It returns false when err is nil.
// The check is case-insensitive and matches both the numeric literal "404" and
// the phrase "not found", which covers all known forms returned by the GitHub API,
// the gh CLI, and the go-gh library.
func IsNotFoundError(err error) bool {
	matched := containsErrorSubstring(err, "404", "not found")
	if matched {
		errorutilLog.Printf("Classified error as not-found (404): %v", err)
	}
	return matched
}

// IsForbiddenError reports whether err represents an HTTP 403 / "forbidden" response.
// It returns false when err is nil.
// The check is case-insensitive and only matches HTTP-style 403 patterns such as
// "HTTP 403" or "403 Forbidden", which avoids misclassifying unrelated errors
// like "forbidden character".
func IsForbiddenError(err error) bool {
	matched := containsHTTPStatusSubstring(err, "403", "forbidden")
	if matched {
		errorutilLog.Printf("Classified error as forbidden (403): %v", err)
	}
	return matched
}

// IsGoneError reports whether err represents an HTTP 410 / "gone" response.
// It returns false when err is nil.
// The check is case-insensitive and only matches HTTP-style 410 patterns such as
// "HTTP 410" or "410 Gone", which avoids misclassifying unrelated errors like
// "connection has gone away".
func IsGoneError(err error) bool {
	matched := containsHTTPStatusSubstring(err, "410", "gone")
	if matched {
		errorutilLog.Printf("Classified error as gone (410): %v", err)
	}
	return matched
}

// containsErrorSubstring reports whether err contains any of the provided
// substrings after lowercasing the full error message for case-insensitive
// matching.
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

// containsHTTPStatusSubstring reports whether err contains a recognized
// HTTP-style status pattern for the provided status code and keyword.
func containsHTTPStatusSubstring(err error, code, keyword string) bool {
	return containsErrorSubstring(
		err,
		"http "+code,
		"http status "+code,
		"status "+code,
		code+": "+keyword,
		code+" "+keyword,
	)
}
