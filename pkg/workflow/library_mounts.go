// Package workflow provides shared library mounting for agent containers.
//
// This file contains functions to determine which shared library directories
// need to be mounted into the agent container to support mounted utilities
// like gh, yq, and date that depend on system libraries.
//
// The mounting strategy uses Option B from the design: mount only required
// library directories (more selective than mounting entire /usr/lib).
// This minimizes security surface area while ensuring mounted binaries work.
package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var libraryMountsLog = logger.New("workflow:library_mounts")

// MountedBinaries is the list of binaries that are mounted from /usr/bin into the container.
// These binaries require shared libraries from /usr/lib to function correctly.
var MountedBinaries = []string{
	"/usr/bin/date",
	"/usr/bin/gh",
	"/usr/bin/yq",
}

// LibraryDirectories are the library directories required by mounted binaries.
// These directories contain shared libraries (.so files) that the binaries depend on.
// The x86_64-linux-gnu directory is the standard location on Ubuntu/Debian runners.
var LibraryDirectories = []string{
	"/usr/lib/x86_64-linux-gnu",
}

// GetLibraryMounts returns the AWF mount arguments for shared library directories.
// These mounts are required when binaries from /usr/bin are mounted into the container.
// The libraries are mounted read-only for security.
func GetLibraryMounts() []string {
	var mounts []string
	for _, dir := range LibraryDirectories {
		// Mount as read-only for security
		mount := dir + ":" + dir + ":ro"
		mounts = append(mounts, "--mount", mount)
	}
	libraryMountsLog.Printf("Added %d library directory mounts for shared library support", len(LibraryDirectories))
	return mounts
}

// HasMountedBinaries returns true if the workflow mounts any binaries that require shared libraries.
// Currently this is always true for copilot engine with firewall enabled, since we mount gh, yq, date.
func HasMountedBinaries() bool {
	return len(MountedBinaries) > 0
}
