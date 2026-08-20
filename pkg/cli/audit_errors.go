package cli

// audit_errors.go: permission/authentication error detection shared across the
// audit pipeline.

import (
	"strings"

	"github.com/github/gh-aw/pkg/errorutil"
)

// isPermissionErrorStr checks if a string contains known permission/authentication markers.
// It delegates to the shared classifier and augments with gh CLI specific hints
// that are only emitted in audit command contexts.
func isPermissionErrorStr(s string) bool {
	if errorutil.IsAuthError(s) {
		return true
	}
	lower := strings.ToLower(s)
	return strings.Contains(lower, "exit status 4") ||
		strings.Contains(lower, "permission") ||
		strings.Contains(lower, "gh auth login") ||
		strings.Contains(lower, "to use github cli in a github actions workflow")
}

// isPermissionError checks if an error is related to permissions/authentication.
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	return isPermissionErrorStr(err.Error())
}
