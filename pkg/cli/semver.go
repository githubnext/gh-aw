package cli

import (
	"strings"

	"golang.org/x/mod/semver"
)

// semanticVersion represents a parsed semantic version
type semanticVersion struct {
	major int
	minor int
	patch int
	pre   string
	raw   string
}

// isSemanticVersionTag checks if a ref string looks like a semantic version tag
// Uses golang.org/x/mod/semver for proper semantic version validation
func isSemanticVersionTag(ref string) bool {
	// Ensure ref has 'v' prefix for semver package
	if !strings.HasPrefix(ref, "v") {
		ref = "v" + ref
	}
	return semver.IsValid(ref)
}

// parseVersion parses a semantic version string
// Uses golang.org/x/mod/semver for proper semantic version parsing
func parseVersion(v string) *semanticVersion {
	// Ensure version has 'v' prefix for semver package
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	// Check if valid semantic version
	if !semver.IsValid(v) {
		return nil
	}

	ver := &semanticVersion{raw: strings.TrimPrefix(v, "v")}

	// Use semver.Canonical to get normalized version
	canonical := semver.Canonical(v)

	// Parse major, minor, patch from canonical form
	// Canonical format is always vMAJOR.MINOR.PATCH
	parts := strings.Split(strings.TrimPrefix(canonical, "v"), ".")
	if len(parts) >= 1 {
		ver.major = parseInt(parts[0])
	}
	if len(parts) >= 2 {
		ver.minor = parseInt(parts[1])
	}
	if len(parts) >= 3 {
		ver.patch = parseInt(parts[2])
	}

	// Get prerelease if any
	prerelease := semver.Prerelease(v)
	// semver.Prerelease includes the leading hyphen, strip it
	ver.pre = strings.TrimPrefix(prerelease, "-")

	return ver
}

// parseInt parses an integer from a string, returns 0 on error
func parseInt(s string) int {
	var result int
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			result = result*10 + int(ch-'0')
		} else {
			break
		}
	}
	return result
}

// isPreciseVersion returns true if this version has explicit minor and patch components
// For example, "v6.0.0" is precise, but "v6" is not
func (v *semanticVersion) isPreciseVersion() bool {
	// Check if raw version has at least two dots (major.minor.patch format)
	// or at least one dot for major.minor format
	// "v6" -> not precise
	// "v6.0" -> somewhat precise (has minor)
	// "v6.0.0" -> precise (has minor and patch)
	versionPart := strings.TrimPrefix(v.raw, "v")
	dotCount := strings.Count(versionPart, ".")
	return dotCount >= 2 // Require at least major.minor.patch
}

// isNewer returns true if this version is newer than the other
// Uses golang.org/x/mod/semver.Compare for proper semantic version comparison
func (v *semanticVersion) isNewer(other *semanticVersion) bool {
	// Ensure versions have 'v' prefix for semver package
	v1 := "v" + v.raw
	v2 := "v" + other.raw

	// Use semver.Compare for comparison
	result := semver.Compare(v1, v2)

	return result > 0
}
