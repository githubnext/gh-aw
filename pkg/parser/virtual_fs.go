package parser

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// builtinVirtualFiles holds embedded built-in files registered at startup.
// Keys use the "@builtin:" path prefix (e.g. "@builtin:engines/copilot.md").
// The map is populated once and then read-only; concurrent reads are safe.
var (
	builtinVirtualFiles   map[string][]byte
	builtinVirtualFilesMu sync.RWMutex
)

// RegisterBuiltinVirtualFile registers an embedded file under a canonical builtin path.
// Paths must start with BuiltinPathPrefix ("@builtin:"); it panics if they do not.
// If the same path is registered twice with identical content the call is a no-op.
// Registering the same path with different content panics to surface configuration errors early.
// This function is safe for concurrent use.
func RegisterBuiltinVirtualFile(path string, content []byte) {
	if !strings.HasPrefix(path, BuiltinPathPrefix) {
		panic(fmt.Sprintf("RegisterBuiltinVirtualFile: path %q does not start with %q", path, BuiltinPathPrefix))
	}
	builtinVirtualFilesMu.Lock()
	defer builtinVirtualFilesMu.Unlock()
	if builtinVirtualFiles == nil {
		builtinVirtualFiles = make(map[string][]byte)
	}
	if existing, ok := builtinVirtualFiles[path]; ok {
		if string(existing) != string(content) {
			panic(fmt.Sprintf("RegisterBuiltinVirtualFile: path %q already registered with different content", path))
		}
		return // idempotent: same content, no-op
	}
	builtinVirtualFiles[path] = content
}

// BuiltinVirtualFileExists returns true if the given path is registered as a builtin virtual file.
func BuiltinVirtualFileExists(path string) bool {
	builtinVirtualFilesMu.RLock()
	defer builtinVirtualFilesMu.RUnlock()
	_, ok := builtinVirtualFiles[path]
	return ok
}

// BuiltinPathPrefix is the path prefix used for embedded builtin files.
// Paths with this prefix bypass filesystem resolution and security checks.
const BuiltinPathPrefix = "@builtin:"

// readFileFunc is the function used to read file contents throughout the parser.
// In wasm builds, this is overridden to read from a virtual filesystem
// populated by the browser via SetVirtualFiles.
// In native builds, builtin virtual files are checked first, then os.ReadFile.
var readFileFunc = func(path string) ([]byte, error) {
	builtinVirtualFilesMu.RLock()
	content, ok := builtinVirtualFiles[path]
	builtinVirtualFilesMu.RUnlock()
	if ok {
		return content, nil
	}
	return os.ReadFile(path)
}

// ReadFile reads a file using the parser's file reading function, which
// checks the virtual filesystem first in wasm builds. Use this instead of
// os.ReadFile when reading files that may be provided as virtual files.
func ReadFile(path string) ([]byte, error) {
	return readFileFunc(path)
}
