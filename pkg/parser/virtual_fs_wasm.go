//go:build js || wasm

package parser

import "fmt"

// virtualFiles holds in-memory file contents for wasm builds.
// Keys are resolved file paths (e.g. "shared/elastic-tools.md").
var virtualFiles map[string][]byte

// SetVirtualFiles populates the virtual filesystem for wasm import resolution.
// Call this before compiling a workflow that uses imports.
// The keys should be file paths relative to the workflow directory
// (e.g. "shared/elastic-tools.md").
func SetVirtualFiles(files map[string][]byte) {
	virtualFiles = files
}

// ClearVirtualFiles removes all virtual files.
func ClearVirtualFiles() {
	virtualFiles = nil
}

// VirtualFileExists checks if a path exists in the virtual filesystem.
func VirtualFileExists(path string) bool {
	if virtualFiles == nil {
		return false
	}
	_, ok := virtualFiles[path]
	return ok
}

func init() {
	// Override readFileFunc in wasm builds to check virtual files first.
	readFileFunc = func(path string) ([]byte, error) {
		if virtualFiles != nil {
			if content, ok := virtualFiles[path]; ok {
				return content, nil
			}
		}
		return nil, fmt.Errorf("file not found in virtual filesystem: %s", path)
	}
}
