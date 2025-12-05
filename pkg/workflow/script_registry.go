// Package workflow provides a ScriptRegistry for managing JavaScript script bundling.
//
// # Script Registry Pattern
//
// The ScriptRegistry eliminates the repetitive sync.Once pattern found throughout
// the codebase for lazy script bundling. Instead of declaring separate variables
// and getter functions for each script, register scripts once and retrieve them
// by name.
//
// # Before (repetitive pattern):
//
//	var (
//	    createIssueScript     string
//	    createIssueScriptOnce sync.Once
//	)
//
//	func getCreateIssueScript() string {
//	    createIssueScriptOnce.Do(func() {
//	        sources := GetJavaScriptSources()
//	        bundled, err := BundleJavaScriptFromSources(createIssueScriptSource, sources, "")
//	        if err != nil {
//	            createIssueScript = createIssueScriptSource
//	        } else {
//	            createIssueScript = bundled
//	        }
//	    })
//	    return createIssueScript
//	}
//
// # After (using registry):
//
//	// Registration at package init
//	DefaultScriptRegistry.Register("create_issue", createIssueScriptSource)
//
//	// Usage anywhere
//	script := DefaultScriptRegistry.Get("create_issue")
//
// # Benefits
//
//   - Eliminates ~15 lines of boilerplate per script (variable pair + getter function)
//   - Centralizes bundling logic
//   - Consistent error handling
//   - Thread-safe lazy initialization
//   - Easy to add new scripts
package workflow

import (
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var registryLog = logger.New("workflow:script_registry")

// scriptEntry holds the source and bundled versions of a script
type scriptEntry struct {
	source  string
	bundled string
	mode    RuntimeMode // Runtime mode for bundling
	once    sync.Once
}

// ScriptRegistry manages lazy bundling of JavaScript scripts.
// It provides a centralized place to register source scripts and retrieve
// bundled versions on-demand with caching.
//
// Thread-safe: All operations use internal synchronization.
//
// Usage:
//
//	registry := NewScriptRegistry()
//	registry.Register("my_script", myScriptSource)
//	bundled := registry.Get("my_script")
type ScriptRegistry struct {
	mu      sync.RWMutex
	scripts map[string]*scriptEntry
}

// NewScriptRegistry creates a new empty script registry.
func NewScriptRegistry() *ScriptRegistry {
	return &ScriptRegistry{
		scripts: make(map[string]*scriptEntry),
	}
}

// Register adds a script source to the registry.
// The script will be bundled lazily on first access via Get().
// Scripts registered this way default to RuntimeModeGitHubScript.
//
// Parameters:
//   - name: Unique identifier for the script (e.g., "create_issue", "add_comment")
//   - source: The raw JavaScript source code (typically from go:embed)
//
// If a script with the same name already exists, it will be overwritten.
// This is useful for testing but should be avoided in production.
func (r *ScriptRegistry) Register(name string, source string) {
	r.RegisterWithMode(name, source, RuntimeModeGitHubScript)
}

// RegisterWithMode adds a script source to the registry with a specific runtime mode.
// The script will be bundled lazily on first access via Get().
//
// Parameters:
//   - name: Unique identifier for the script (e.g., "create_issue", "add_comment")
//   - source: The raw JavaScript source code (typically from go:embed)
//   - mode: Runtime mode for bundling (GitHub Script or Node.js)
//
// If a script with the same name already exists, it will be overwritten.
// This is useful for testing but should be avoided in production.
func (r *ScriptRegistry) RegisterWithMode(name string, source string, mode RuntimeMode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if registryLog.Enabled() {
		registryLog.Printf("Registering script: %s (%d bytes, mode: %s)", name, len(source), mode)
	}

	r.scripts[name] = &scriptEntry{
		source: source,
		mode:   mode,
	}
}

// Get retrieves a bundled script by name.
// Bundling is performed lazily on first access and cached for subsequent calls.
//
// If bundling fails, the original source is returned as a fallback.
// If the script is not registered, an empty string is returned.
//
// Thread-safe: Multiple goroutines can call Get concurrently.
func (r *ScriptRegistry) Get(name string) string {
	r.mu.RLock()
	entry, exists := r.scripts[name]
	r.mu.RUnlock()

	if !exists {
		if registryLog.Enabled() {
			registryLog.Printf("Script not found: %s", name)
		}
		return ""
	}

	entry.once.Do(func() {
		if registryLog.Enabled() {
			registryLog.Printf("Bundling script: %s (mode: %s)", name, entry.mode)
		}

		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptWithMode(entry.source, sources, "", entry.mode)
		if err != nil {
			registryLog.Printf("Bundling failed for %s, using source as-is: %v", name, err)
			entry.bundled = entry.source
		} else {
			if registryLog.Enabled() {
				registryLog.Printf("Successfully bundled %s: %d bytes", name, len(bundled))
			}
			entry.bundled = bundled
		}
	})

	return entry.bundled
}

// GetSource retrieves the original (unbundled) source for a script.
// Useful for testing or when bundling is not needed.
func (r *ScriptRegistry) GetSource(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.scripts[name]
	if !exists {
		return ""
	}
	return entry.source
}

// Has checks if a script is registered in the registry.
func (r *ScriptRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.scripts[name]
	return exists
}

// Names returns a list of all registered script names.
// Useful for debugging and testing.
func (r *ScriptRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.scripts))
	for name := range r.scripts {
		names = append(names, name)
	}
	return names
}

// DefaultScriptRegistry is the global script registry used by the workflow package.
// Scripts are registered during package initialization via init() functions.
var DefaultScriptRegistry = NewScriptRegistry()

// GetScript retrieves a bundled script from the default registry.
// This is a convenience function equivalent to DefaultScriptRegistry.Get(name).
func GetScript(name string) string {
	return DefaultScriptRegistry.Get(name)
}
