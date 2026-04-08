package parser

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
)

var virtualFsLog = logger.New("parser:virtual_fs")

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
	virtualFsLog.Printf("Registering builtin virtual file: %s (%d bytes)", path, len(content))
	builtinVirtualFiles[path] = content
}

// BuiltinVirtualFileExists returns true if the given path is registered as a builtin virtual file.
func BuiltinVirtualFileExists(path string) bool {
	builtinVirtualFilesMu.RLock()
	defer builtinVirtualFilesMu.RUnlock()
	_, ok := builtinVirtualFiles[path]
	virtualFsLog.Printf("BuiltinVirtualFileExists: path=%s exists=%t", path, ok)
	return ok
}

// builtinFrontmatterCache caches the result of parsing frontmatter for builtin virtual files.
// Builtin files are immutable (registered once at startup), so the parse result is stable
// across the lifetime of the process. This avoids repeated YAML parsing for frequently
// imported engine definition files (e.g. @builtin:engines/copilot.md).
// Cached values are shared read-only *FrontmatterResult references; callers must not mutate
// the cached result or any contained maps/slices.
var builtinFrontmatterCache sync.Map // map[string]*FrontmatterResult

// GetBuiltinFrontmatterCache returns the cached FrontmatterResult for a builtin virtual file.
// Returns (result, true) if cached, (nil, false) if not yet cached.
//
// IMPORTANT: The returned *FrontmatterResult is a shared, read-only reference.
// Callers MUST NOT mutate the result or any of its fields (Frontmatter map, slices, etc.).
// Use ExtractFrontmatterFromContent directly when you need a mutable copy.
func GetBuiltinFrontmatterCache(path string) (*FrontmatterResult, bool) {
	v, ok := builtinFrontmatterCache.Load(path)
	if !ok {
		virtualFsLog.Printf("Frontmatter cache miss: path=%s", path)
		return nil, false
	}
	virtualFsLog.Printf("Frontmatter cache hit: path=%s", path)
	return v.(*FrontmatterResult), true
}

// SetBuiltinFrontmatterCache stores a FrontmatterResult for a builtin virtual file.
// The stored result becomes shared and read-only — callers MUST NOT mutate it
// (or its contained maps/slices) after this call.
// Uses LoadOrStore so concurrent races are safe; the winning value is returned.
func SetBuiltinFrontmatterCache(path string, result *FrontmatterResult) *FrontmatterResult {
	actual, loaded := builtinFrontmatterCache.LoadOrStore(path, result)
	if loaded {
		virtualFsLog.Printf("Frontmatter cache already populated (race): path=%s", path)
	} else {
		virtualFsLog.Printf("Frontmatter cache stored: path=%s", path)
	}
	return actual.(*FrontmatterResult)
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
