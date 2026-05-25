package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
)

var versionGateLog = logger.New("workflow:version_gate")

// versionAtLeast returns true when versionToCheck is at or above minVersion.
//
// If versionToCheck is empty, defaultVersion is used. "latest" always returns true.
// Non-semver strings (e.g. branch names) return false (conservative).
func versionAtLeast(versionToCheck, defaultVersion, minVersion string) bool {
	if versionToCheck == "" {
		versionGateLog.Printf("versionAtLeast: empty version, using default=%s", defaultVersion)
		versionToCheck = defaultVersion
	}
	if strings.EqualFold(versionToCheck, "latest") {
		versionGateLog.Printf("versionAtLeast: 'latest' satisfies min=%s", minVersion)
		return true
	}
	result := semverutil.Compare(versionToCheck, minVersion) >= 0
	versionGateLog.Printf("versionAtLeast: version=%s min=%s result=%v", versionToCheck, minVersion, result)
	return result
}
