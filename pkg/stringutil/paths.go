package stringutil

import "strings"

// NormalizePath normalizes a file path by resolving . and .. components.
// It splits the path on "/" and processes each component:
// - Empty parts and "." are skipped
// - ".." moves up one directory (if possible)
// - Other parts are added to the result
//
// This is useful for resolving relative paths in bundler operations and
// other file path manipulations where . and .. components need to be resolved.
//
// Examples:
//
//	NormalizePath("a/b/../c")        // returns "a/c"
//	NormalizePath("./a/./b")         // returns "a/b"
//	NormalizePath("a/b/../../c")     // returns "c"
//	NormalizePath("../a/b")          // returns "a/b" (leading .. is ignored)
//	NormalizePath("a//b")            // returns "a/b" (empty parts removed)
func NormalizePath(path string) string {
	// Split path into parts
	parts := strings.Split(path, "/")
	var result []string

	for _, part := range parts {
		if part == "" || part == "." {
			// Skip empty parts and current directory references
			continue
		}
		if part == ".." {
			// Go up one directory
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
		} else {
			result = append(result, part)
		}
	}

	return strings.Join(result, "/")
}
