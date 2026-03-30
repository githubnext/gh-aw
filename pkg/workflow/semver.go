package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
)

var semverLog = logger.New("workflow:semver")

// isValidVersionTag checks if a string is a valid action version tag.
// Supports vmajor, vmajor.minor, and vmajor.minor.patch formats only.
func isValidVersionTag(s string) bool {
	return semverutil.IsActionVersionTag(s)
}

// compareVersions compares two semantic versions, returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
// Uses golang.org/x/mod/semver for proper semantic version comparison
func compareVersions(v1, v2 string) int {
	semverLog.Printf("Comparing versions: v1=%s, v2=%s", v1, v2)
	return semverutil.Compare(v1, v2)
}

// isSemverCompatible checks if pinVersion is semver-compatible with requestedVersion
// Semver compatibility means the major version must match
// Examples:
//   - isSemverCompatible("v5.0.0", "v5") -> true
//   - isSemverCompatible("v5.1.0", "v5.0.0") -> true
//   - isSemverCompatible("v6.0.0", "v5") -> false
func isSemverCompatible(pinVersion, requestedVersion string) bool {
	return semverutil.IsCompatible(pinVersion, requestedVersion)
}
