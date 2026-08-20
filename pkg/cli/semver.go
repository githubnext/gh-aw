package cli

import "github.com/github/gh-aw/pkg/semverutil"

// isSemanticVersionTag checks if a ref string looks like a semantic version tag
// Uses golang.org/x/mod/semver for proper semantic version validation
func isSemanticVersionTag(ref string) bool {
	return semverutil.IsValid(ref)
}

// parseVersion parses a semantic version string and returns a *semverutil.SemanticVersion.
// Uses golang.org/x/mod/semver for proper semantic version parsing.
func parseVersion(v string) *semverutil.SemanticVersion {
	return semverutil.ParseVersion(v)
}
