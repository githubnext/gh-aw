package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/semverutil"
)

// versionAtLeast returns true when versionToCheck is at or above minVersion.
//
// If versionToCheck is empty, defaultVersion is used. "latest" always returns true.
// Non-semver strings (e.g. branch names) return false (conservative).
func versionAtLeast(versionToCheck, defaultVersion, minVersion string) bool {
	if versionToCheck == "" {
		versionToCheck = defaultVersion
	}
	if strings.EqualFold(versionToCheck, "latest") {
		return true
	}
	return semverutil.Compare(versionToCheck, minVersion) >= 0
}
