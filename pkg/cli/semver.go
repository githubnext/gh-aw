package cli

import (
	"fmt"
	"regexp"
	"strings"
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
func isSemanticVersionTag(ref string) bool {
	// Match v1.0.0, v1.0, 1.0.0, etc.
	semverPattern := regexp.MustCompile(`^v?\d+(\.\d+)*(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$`)
	return semverPattern.MatchString(ref)
}

// parseVersion parses a semantic version string
func parseVersion(v string) *semanticVersion {
	// Remove leading 'v' if present
	v = strings.TrimPrefix(v, "v")

	// Match semantic version pattern
	re := regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([a-zA-Z0-9.]+))?`)
	matches := re.FindStringSubmatch(v)
	if matches == nil {
		return nil
	}

	ver := &semanticVersion{raw: v}

	if matches[1] != "" {
		_, _ = fmt.Sscanf(matches[1], "%d", &ver.major)
	}
	if matches[2] != "" {
		_, _ = fmt.Sscanf(matches[2], "%d", &ver.minor)
	}
	if matches[3] != "" {
		_, _ = fmt.Sscanf(matches[3], "%d", &ver.patch)
	}
	if matches[4] != "" {
		ver.pre = matches[4]
	}

	return ver
}

// isNewer returns true if this version is newer than the other
func (v *semanticVersion) isNewer(other *semanticVersion) bool {
	if v.major != other.major {
		return v.major > other.major
	}
	if v.minor != other.minor {
		return v.minor > other.minor
	}
	if v.patch != other.patch {
		return v.patch > other.patch
	}
	// If versions are equal but one has a prerelease tag, prefer the one without
	if v.pre == "" && other.pre != "" {
		return true
	}
	if v.pre != "" && other.pre == "" {
		return false
	}
	// Both have prerelease or both don't - consider equal
	return false
}
