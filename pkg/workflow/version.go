package workflow

import (
	"strings"
)

// compilerVersion holds the version of the compiler, set at runtime.
// This is used to include version information in generated workflow headers.
var compilerVersion = "dev"

// isReleaseBuild indicates whether this binary was built as a release.
// This is set at build time via -X linker flag and used to determine
// if version information should be included in generated workflows.
var isReleaseBuild = false

// SetVersion sets the compiler version for inclusion in generated workflow headers.
// Only non-dev versions are included in the generated headers.
func SetVersion(v string) {
	compilerVersion = v
}

// GetVersion returns the current compiler version.
func GetVersion() string {
	return compilerVersion
}

// SetIsRelease sets whether this binary was built as a release.
func SetIsRelease(release bool) {
	isReleaseBuild = release
}

// IsRelease returns whether this binary was built as a release.
func IsRelease() bool {
	return isReleaseBuild
}

// IsReleasedVersion checks if a version string represents a released build.
// It now primarily relies on the isReleaseBuild flag set at build time,
// but also validates that the version matches semantic versioning format (x.y.z or x.y.z-prerelease)
// and excludes development builds (containing "dev", "dirty", or "test") as a fallback.
func IsReleasedVersion(version string) bool {
	// Primary check: use the build-time flag if available
	// If the binary was built as a release, trust that flag
	if isReleaseBuild {
		return true
	}

	// Fallback to heuristic checks for backward compatibility
	// and for cases where the flag wasn't set (e.g., custom builds)
	if version == "" {
		return false
	}
	// Filter out development/test versions
	excludePatterns := []string{"dev", "dirty", "test"}
	for _, pattern := range excludePatterns {
		if strings.Contains(version, pattern) {
			return false
		}
	}

	// Validate semantic version format: must start with digit and contain at least one dot
	// Examples: "1.2.3", "1.2.3-beta.1", "0.1.0-rc.2+build.123"
	// Non-examples: "e63fd5a", "abc", "v1.2.3" (should start with digit)
	if len(version) == 0 {
		return false
	}

	// Must start with a digit
	if version[0] < '0' || version[0] > '9' {
		return false
	}

	// Must contain at least one dot (to have major.minor.patch)
	if !strings.Contains(version, ".") {
		return false
	}

	// Extract the version core (before any prerelease or metadata)
	versionCore := version
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		versionCore = version[:idx]
	}

	// Validate that the core contains only digits and dots
	// Split by dots and ensure at least 2 parts (major.minor at minimum)
	parts := strings.Split(versionCore, ".")
	if len(parts) < 2 {
		return false
	}

	// Each part should be numeric
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}

	return true
}
