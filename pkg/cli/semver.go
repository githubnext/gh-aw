package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
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
	return semverutil.IsValid(ref)
}

// parseVersion parses a semantic version string
// Uses golang.org/x/mod/semver for proper semantic version parsing
func parseVersion(v string) *semanticVersion {
	semverLog.Printf("Parsing semantic version: %s", v)
	parsed := semverutil.ParseVersion(v)
	if parsed == nil {
		semverLog.Printf("Invalid semantic version: %s", v)
		return nil
	}
	return &semanticVersion{
		major: parsed.Major,
		minor: parsed.Minor,
		patch: parsed.Patch,
		pre:   parsed.Pre,
		raw:   parsed.Raw,
	}
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
	isNewer := semverutil.Compare(v.raw, other.raw) > 0
	semverLog.Printf("Version comparison: %s vs %s, isNewer=%v", v.raw, other.raw, isNewer)
	return isNewer
}
