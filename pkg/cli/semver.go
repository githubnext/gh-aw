package cli

import (
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/github/gh-aw/pkg/logger"
)

var semverLog = logger.New("cli:semver")

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
	semverLog.Printf("Parsing semantic version: %s", v)
	// Ensure version has 'v' prefix for semver package
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	// Check if valid semantic version
	if !semver.IsValid(v) {
		semverLog.Printf("Invalid semantic version: %s", v)
		return nil
	}

	ver := &semanticVersion{raw: strings.TrimPrefix(v, "v")}

	// Use semver.Canonical to get normalized version
	canonical := semver.Canonical(v)

	// Parse major, minor, patch from canonical form
	// Strip prerelease and build metadata before splitting, since semver.Canonical
	// preserves the prerelease suffix (e.g. "v1.2.3-beta.1" stays "v1.2.3-beta.1")
	corePart := strings.TrimPrefix(canonical, "v")
	if idx := strings.IndexAny(corePart, "-+"); idx >= 0 {
		corePart = corePart[:idx]
	}
	parts := strings.Split(corePart, ".")
	// Parse the numeric components; strconv.Atoi returns 0 on error, matching
	// the previous behavior where non-numeric input produced 0.
	if len(parts) >= 1 {
		ver.major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		ver.minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		ver.patch, _ = strconv.Atoi(parts[2])
	}

	// Get prerelease if any
	prerelease := semver.Prerelease(v)
	// semver.Prerelease includes the leading hyphen, strip it
	ver.pre = strings.TrimPrefix(prerelease, "-")

	return ver
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

	isNewer := result > 0
	semverLog.Printf("Version comparison: %s vs %s, isNewer=%v", v.raw, other.raw, isNewer)
	return isNewer
}
