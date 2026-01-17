package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"golang.org/x/mod/semver"
)

var semverLog = logger.New("workflow:semver")

// normalizeSemverLike coerces common GitHub Action tag formats into valid semver strings
// understood by golang.org/x/mod/semver.
//
// Examples:
//   - "v2" -> "v2.0.0"
//   - "2" -> "v2.0.0"
//   - "v2.1" -> "v2.1.0"
//   - "v2.1.3" -> "v2.1.3"
func normalizeSemverLike(v string) string {
	if v == "" {
		return v
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	// Count dot components after the optional leading "v".
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	switch len(parts) {
	case 1:
		// v2
		return v + ".0.0"
	case 2:
		// v2.1
		return v + ".0"
	default:
		return v
	}
}

// compareVersions compares two semantic versions, returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
// Uses golang.org/x/mod/semver for proper semantic version comparison
func compareVersions(v1, v2 string) int {
	semverLog.Printf("Comparing versions: v1=%s, v2=%s", v1, v2)

	v1 = normalizeSemverLike(v1)
	v2 = normalizeSemverLike(v2)

	result := semver.Compare(v1, v2)

	if result > 0 {
		semverLog.Printf("Version comparison result: %s > %s", v1, v2)
	} else if result < 0 {
		semverLog.Printf("Version comparison result: %s < %s", v1, v2)
	} else {
		semverLog.Printf("Version comparison result: %s == %s", v1, v2)
	}

	return result
}

// extractMajorVersion extracts the major version number from a version string
// Examples: "v5.0.0" -> 5, "v6" -> 6, "5.1.0" -> 5
// Uses golang.org/x/mod/semver.Major for proper semantic version parsing
func extractMajorVersion(version string) int {
	version = normalizeSemverLike(version)

	// Get major version string (e.g., "v5")
	majorStr := semver.Major(version)
	if majorStr == "" {
		return 0
	}

	// Parse the integer from the major version string
	// Major returns "v5", we need to extract 5
	var major int
	// Strip 'v' prefix and parse the number
	numStr := strings.TrimPrefix(majorStr, "v")
	if numStr != "" {
		// Parse the number
		for _, ch := range numStr {
			if ch >= '0' && ch <= '9' {
				major = major*10 + int(ch-'0')
			} else {
				break
			}
		}
	}

	return major
}

// isSemverCompatible checks if pinVersion is semver-compatible with requestedVersion
// Semver compatibility means the major version must match
// Examples:
//   - isSemverCompatible("v5.0.0", "v5") -> true
//   - isSemverCompatible("v5.1.0", "v5.0.0") -> true
//   - isSemverCompatible("v6.0.0", "v5") -> false
func isSemverCompatible(pinVersion, requestedVersion string) bool {
	pinVersion = normalizeSemverLike(pinVersion)
	requestedVersion = normalizeSemverLike(requestedVersion)

	// Use semver.Major to get major version strings
	pinMajor := semver.Major(pinVersion)
	requestedMajor := semver.Major(requestedVersion)

	compatible := pinMajor == requestedMajor
	semverLog.Printf("Checking semver compatibility: pin=%s (major=%s), requested=%s (major=%s) -> %v",
		pinVersion, pinMajor, requestedVersion, requestedMajor, compatible)

	return compatible
}
