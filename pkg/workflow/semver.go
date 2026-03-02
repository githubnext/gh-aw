package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"golang.org/x/mod/semver"
)

var semverLog = logger.New("workflow:semver")

// compareVersions compares two semantic versions, returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
// Uses golang.org/x/mod/semver for proper semantic version comparison
func compareVersions(v1, v2 string) int {
	semverLog.Printf("Comparing versions: v1=%s, v2=%s", v1, v2)

	// Ensure versions have 'v' prefix for semver package
	if !strings.HasPrefix(v1, "v") {
		v1 = "v" + v1
	}
	if !strings.HasPrefix(v2, "v") {
		v2 = "v" + v2
	}

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

// isSemverCompatible checks if pinVersion is semver-compatible with requestedVersion
// Semver compatibility means the major version must match
// Examples:
//   - isSemverCompatible("v5.0.0", "v5") -> true
//   - isSemverCompatible("v5.1.0", "v5.0.0") -> true
//   - isSemverCompatible("v6.0.0", "v5") -> false
func isSemverCompatible(pinVersion, requestedVersion string) bool {
	// Ensure versions have 'v' prefix for semver package
	if !strings.HasPrefix(pinVersion, "v") {
		pinVersion = "v" + pinVersion
	}
	if !strings.HasPrefix(requestedVersion, "v") {
		requestedVersion = "v" + requestedVersion
	}

	// Use semver.Major to get major version strings
	pinMajor := semver.Major(pinVersion)
	requestedMajor := semver.Major(requestedVersion)

	compatible := pinMajor == requestedMajor
	semverLog.Printf("Checking semver compatibility: pin=%s (major=%s), requested=%s (major=%s) -> %v",
		pinVersion, pinMajor, requestedVersion, requestedMajor, compatible)

	return compatible
}
