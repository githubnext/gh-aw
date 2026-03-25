package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var versionLog = logger.New("workflow:version")

// compilerVersion is the single source of truth for the compiler version.
// It is set at runtime by SetVersion (called during CLI initialization) and used:
//   - In generated workflow headers (via GetVersion)
//   - When creating new Compiler instances (NewCompiler reads it via GetVersion)
//
// Initialization flow:
//
//	main.go → cli.SetVersionInfo(v) → workflow.SetVersion(v)   (sets compilerVersion)
//	                                  → NewCompiler()            (reads compilerVersion via GetVersion)
var compilerVersion = "dev"

// isReleaseBuild indicates whether this binary was built as a release.
// This is set at build time via -X linker flag and used to determine
// if version information should be included in generated workflows.
var isReleaseBuild = false

// SetVersion sets the compiler version. Call once during CLI initialization.
// The version is used in generated workflow headers and as the default version
// for new Compiler instances created via NewCompiler.
func SetVersion(v string) {
	versionLog.Printf("Setting compiler version: %s", v)
	compilerVersion = v
}

// GetVersion returns the current compiler version.
func GetVersion() string {
	return compilerVersion
}

// SetIsRelease sets whether this binary was built as a release.
func SetIsRelease(release bool) {
	versionLog.Printf("Setting release build flag: %v", release)
	isReleaseBuild = release
}

// IsRelease returns whether this binary was built as a release.
func IsRelease() bool {
	return isReleaseBuild
}

// IsReleasedVersion checks if a version string represents a released build.
// It relies on the isReleaseBuild flag set at build time via -X linker flag.
func IsReleasedVersion(version string) bool {
	return isReleaseBuild
}
