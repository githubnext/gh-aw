package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var semverLog = logger.New("workflow:semver")

// compareVersions compares two semantic versions, returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
// Note: Non-numeric version parts (e.g., 'beta', 'alpha') default to 0 for comparison purposes
func compareVersions(v1, v2 string) int {
	semverLog.Printf("Comparing versions: v1=%s, v2=%s", v1, v2)

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			_, _ = fmt.Sscanf(parts1[i], "%d", &p1) // Ignore error, defaults to 0 for non-numeric parts
		}
		if i < len(parts2) {
			_, _ = fmt.Sscanf(parts2[i], "%d", &p2) // Ignore error, defaults to 0 for non-numeric parts
		}

		if p1 > p2 {
			semverLog.Printf("Version comparison result: %s > %s", v1, v2)
			return 1
		} else if p1 < p2 {
			semverLog.Printf("Version comparison result: %s < %s", v1, v2)
			return -1
		}
	}

	semverLog.Printf("Version comparison result: %s == %s", v1, v2)
	return 0
}

// extractMajorVersion extracts the major version number from a version string
// Examples: "v5.0.0" -> 5, "v6" -> 6, "5.1.0" -> 5
func extractMajorVersion(version string) int {
	// Strip 'v' prefix if present
	v := strings.TrimPrefix(version, "v")

	// Split by '.' and get the first part
	parts := strings.Split(v, ".")
	if len(parts) > 0 {
		var major int
		fmt.Sscanf(parts[0], "%d", &major)
		return major
	}

	return 0
}

// isSemverCompatible checks if pinVersion is semver-compatible with requestedVersion
// Semver compatibility means the major version must match
// Examples:
//   - isSemverCompatible("v5.0.0", "v5") -> true
//   - isSemverCompatible("v5.1.0", "v5.0.0") -> true
//   - isSemverCompatible("v6.0.0", "v5") -> false
func isSemverCompatible(pinVersion, requestedVersion string) bool {
	pinMajor := extractMajorVersion(pinVersion)
	requestedMajor := extractMajorVersion(requestedVersion)

	compatible := pinMajor == requestedMajor
	semverLog.Printf("Checking semver compatibility: pin=%s (major=%d), requested=%s (major=%d) -> %v",
		pinVersion, pinMajor, requestedVersion, requestedMajor, compatible)

	return compatible
}
