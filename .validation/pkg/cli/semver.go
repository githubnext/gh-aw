package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
)

var semverLog = logger.New("cli:semver")

// isSemanticVersionTag checks if a ref string looks like a semantic version tag
// Uses golang.org/x/mod/semver for proper semantic version validation
func isSemanticVersionTag(ref string) bool {
	return semverutil.IsValid(ref)
}

// parseVersion parses a semantic version string and returns a *semverutil.SemanticVersion.
// Uses golang.org/x/mod/semver for proper semantic version parsing.
func parseVersion(v string) *semverutil.SemanticVersion {
	semverLog.Printf("Parsing semantic version: %s", v)
	parsed := semverutil.ParseVersion(v)
	if parsed == nil {
		semverLog.Printf("Invalid semantic version: %s", v)
	}
	return parsed
}
