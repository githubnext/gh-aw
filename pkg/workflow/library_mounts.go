// Package workflow provides shared library mounting logic for agent containers.
//
// This file contains functions for mounting essential shared libraries from
// /usr/lib into the agent container to support utilities that depend on
// system libraries. The approach uses selective library directory mounting
// (mounting specific library subdirectories rather than the entire /usr/lib)
// to minimize security surface area while ensuring that mounted binaries
// from /usr/bin can access their required library dependencies.
//
// Key library directories mounted:
//   - /usr/lib/x86_64-linux-gnu: Main shared library directory on x86_64 Linux
//   - /lib/x86_64-linux-gnu: Alternative library location for some core libs
//
// This supports utilities like curl, grep, sed, jq, yq, and gh that depend
// on shared libraries such as libcurl, libz, libpcre, and others.
package workflow

// GetLibraryMountArgs returns the AWF mount arguments for shared library directories.
// These mounts enable mounted /usr/bin utilities to access their shared library dependencies.
//
// The function returns mount arguments for the primary library directories that contain
// the shared libraries (.so files) required by common utilities mounted from /usr/bin.
//
// Library directories mounted (readonly):
//   - /usr/lib/x86_64-linux-gnu: Primary location for shared libraries on x86_64 Linux
//   - /lib/x86_64-linux-gnu: Alternative location for some core system libraries
//
// Security considerations:
//   - All library mounts are read-only to prevent modification
//   - Only essential library directories are mounted (not the entire /usr/lib tree)
//   - This approach minimizes the attack surface while providing necessary library access
func GetLibraryMountArgs() []string {
	var args []string

	// Mount the primary x86_64 library directory (contains most shared libraries)
	// This includes libraries like libcurl.so, libz.so, libpcre.so, libjq.so, etc.
	args = append(args, "--mount", "/usr/lib/x86_64-linux-gnu:/usr/lib/x86_64-linux-gnu:ro")

	// Mount the alternative library location for core system libraries
	// Some utilities may link against libraries in this directory
	args = append(args, "--mount", "/lib/x86_64-linux-gnu:/lib/x86_64-linux-gnu:ro")

	return args
}

// GetBinaryMountArgs returns the AWF mount arguments for essential /usr/bin utilities.
// This centralizes the binary mount configuration that was previously duplicated
// in the engine files.
//
// The binaries are organized by priority based on usage frequency in workflows:
//
// Essential utilities (most commonly used):
//   - cat, curl, date, find, gh, grep, jq, yq
//
// Common utilities (frequently used for file operations):
//   - cp, cut, diff, head, ls, mkdir, rm, sed, sort, tail, wc, which
//
// All mounts are read-only to prevent modification of host binaries.
func GetBinaryMountArgs() []string {
	var args []string

	// Essential utilities (most commonly used)
	args = append(args, "--mount", "/usr/bin/cat:/usr/bin/cat:ro")
	args = append(args, "--mount", "/usr/bin/curl:/usr/bin/curl:ro")
	args = append(args, "--mount", "/usr/bin/date:/usr/bin/date:ro")
	args = append(args, "--mount", "/usr/bin/find:/usr/bin/find:ro")
	args = append(args, "--mount", "/usr/bin/gh:/usr/bin/gh:ro")
	args = append(args, "--mount", "/usr/bin/grep:/usr/bin/grep:ro")
	args = append(args, "--mount", "/usr/bin/jq:/usr/bin/jq:ro")
	args = append(args, "--mount", "/usr/bin/yq:/usr/bin/yq:ro")

	// Common utilities (frequently used for file operations)
	args = append(args, "--mount", "/usr/bin/cp:/usr/bin/cp:ro")
	args = append(args, "--mount", "/usr/bin/cut:/usr/bin/cut:ro")
	args = append(args, "--mount", "/usr/bin/diff:/usr/bin/diff:ro")
	args = append(args, "--mount", "/usr/bin/head:/usr/bin/head:ro")
	args = append(args, "--mount", "/usr/bin/ls:/usr/bin/ls:ro")
	args = append(args, "--mount", "/usr/bin/mkdir:/usr/bin/mkdir:ro")
	args = append(args, "--mount", "/usr/bin/rm:/usr/bin/rm:ro")
	args = append(args, "--mount", "/usr/bin/sed:/usr/bin/sed:ro")
	args = append(args, "--mount", "/usr/bin/sort:/usr/bin/sort:ro")
	args = append(args, "--mount", "/usr/bin/tail:/usr/bin/tail:ro")
	args = append(args, "--mount", "/usr/bin/wc:/usr/bin/wc:ro")
	args = append(args, "--mount", "/usr/bin/which:/usr/bin/which:ro")

	return args
}

// GetAllUtilityMountArgs returns all mount arguments needed for utilities and their
// library dependencies. This combines binary mounts and library mounts.
//
// Use this function to get the complete set of mounts needed for mounted
// /usr/bin utilities to work correctly inside the AWF container.
func GetAllUtilityMountArgs() []string {
	var args []string

	// First add binary mounts
	args = append(args, GetBinaryMountArgs()...)

	// Then add library mounts for shared library dependencies
	args = append(args, GetLibraryMountArgs()...)

	return args
}
