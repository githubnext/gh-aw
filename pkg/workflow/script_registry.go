package workflow

import (
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
)

var registryLog = logger.New("workflow:script_registry")

// scriptEntry holds metadata about a registered script.
type scriptEntry struct {
	actionPath string // Optional path to custom action (e.g., "./actions/create-issue")
}

// ScriptRegistry manages script metadata and custom action paths.
type ScriptRegistry struct {
	mu      sync.RWMutex
	scripts map[string]*scriptEntry
}

// NewScriptRegistry creates a new empty script registry.
func NewScriptRegistry() *ScriptRegistry {
	registryLog.Print("Creating new script registry")
	return &ScriptRegistry{
		scripts: make(map[string]*scriptEntry),
	}
}

// GetActionPath retrieves the custom action path for a script, if registered.
// Returns an empty string if the script doesn't have a custom action path.
func (r *ScriptRegistry) GetActionPath(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.scripts[name]
	if !exists {
		if registryLog.Enabled() {
			registryLog.Printf("GetActionPath: script not found: %s", name)
		}
		return ""
	}
	if registryLog.Enabled() && entry.actionPath != "" {
		registryLog.Printf("GetActionPath: returning action path for %s: %s", name, entry.actionPath)
	}
	return entry.actionPath
}

// DefaultScriptRegistry is the global script registry used by the workflow package.
// Scripts are registered during package initialization via init() functions.
var DefaultScriptRegistry = NewScriptRegistry()

// GetAllScriptFilenames returns a sorted list of all .cjs filenames from the JavaScript sources.
// This is used by the build system to discover which files need to be embedded in custom actions.
func GetAllScriptFilenames() []string {
	registryLog.Print("Getting all script filenames from JavaScript sources")
	sources := GetJavaScriptSources()
	filenames := make([]string, 0, len(sources))

	for filename := range sources {
		if strings.HasSuffix(filename, ".cjs") {
			filenames = append(filenames, filename)
		}
	}

	registryLog.Printf("Found %d .cjs files in JavaScript sources", len(filenames))

	sortedFilenames := make([]string, len(filenames))
	copy(sortedFilenames, filenames)
	for i := range sortedFilenames {
		for j := i + 1; j < len(sortedFilenames); j++ {
			if sortedFilenames[i] > sortedFilenames[j] {
				sortedFilenames[i], sortedFilenames[j] = sortedFilenames[j], sortedFilenames[i]
			}
		}
	}

	return sortedFilenames
}
