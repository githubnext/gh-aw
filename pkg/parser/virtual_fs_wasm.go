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
	parserLog.Printf("SetVirtualFiles: registering %d virtual files", len(files))
	virtualFiles = files
}

// ClearVirtualFiles removes all virtual files.
func ClearVirtualFiles() {
	parserLog.Print("ClearVirtualFiles: clearing virtual filesystem")
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
		// Check builtin virtual files first (embedded engine .md files etc.)
		builtinVirtualFilesMu.RLock()
		defer builtinVirtualFilesMu.RUnlock()
		builtinContent, builtinOK := (*builtinVirtualFiles)[path]
		if builtinOK {
			parserLog.Printf("readFileFunc: resolved builtin virtual file: %s", path)
			return builtinContent, nil
		}
		if virtualFiles != nil {
			if content, ok := virtualFiles[path]; ok {
				parserLog.Printf("readFileFunc: resolved user virtual file: %s", path)
				return content, nil
			}
		}
		parserLog.Printf("readFileFunc: file not found in virtual filesystem: %s", path)
		return nil, fmt.Errorf("file not found in virtual filesystem: %s", path)
	}
}
