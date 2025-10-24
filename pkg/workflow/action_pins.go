package workflow

import "strings"

// ActionPin represents a pinned GitHub Action with its commit SHA
type ActionPin struct {
	Repo    string // e.g., "actions/checkout"
	Version string // e.g., "v5" - the golden/default version
	SHA     string // Full commit SHA for the pinned version
}

// actionPins maps action repository to their default pinned versions
// These are the "golden" versions that should be used across all generated workflows
// These SHAs should be updated periodically to use the latest stable versions
var actionPins = map[string]ActionPin{
	// Core actions
	"actions/checkout": {
		Repo:    "actions/checkout",
		Version: "v5",
		SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8", // v5
	},
	"actions/github-script": {
		Repo:    "actions/github-script",
		Version: "v8",
		SHA:     "ed597411d8f924073f98dfc5c65a23a2325f34cd", // v8
	},
	"actions/upload-artifact": {
		Repo:    "actions/upload-artifact",
		Version: "v4",
		SHA:     "ea165f8d65b6e75b540449e92b4886f43607fa02", // v4
	},
	"actions/download-artifact": {
		Repo:    "actions/download-artifact",
		Version: "v5",
		SHA:     "634f93cb2916e3fdff6788551b99b062d0335ce0", // v5
	},
	"actions/cache": {
		Repo:    "actions/cache",
		Version: "v4",
		SHA:     "0057852bfaa89a56745cba8c7296529d2fc39830", // v4
	},

	// Setup actions
	"actions/setup-node": {
		Repo:    "actions/setup-node",
		Version: "v4",
		SHA:     "49933ea5288caeca8642d1e84afbd3f7d6820020", // v4
	},
	"actions/setup-python": {
		Repo:    "actions/setup-python",
		Version: "v5",
		SHA:     "a26af69be951a213d495a4c3e4e4022e16d87065", // v5
	},
	"actions/setup-go": {
		Repo:    "actions/setup-go",
		Version: "v5",
		SHA:     "d35c59abb061a4a6fb18e82ac0862c26744d6ab5", // v5
	},
	"actions/setup-java": {
		Repo:    "actions/setup-java",
		Version: "v4",
		SHA:     "c5195efecf7bdfc987ee8bae7a71cb8b11521c00", // v4
	},
	"actions/setup-dotnet": {
		Repo:    "actions/setup-dotnet",
		Version: "v4",
		SHA:     "67a3573c9a986a3f9c594539f4ab511d57bb3ce9", // v4
	},

	// Third-party setup actions
	"erlef/setup-beam": {
		Repo:    "erlef/setup-beam",
		Version: "v1",
		SHA:     "3559ac3b631a9560f28817e8e7fdde1638664336", // v1
	},
	"haskell-actions/setup": {
		Repo:    "haskell-actions/setup",
		Version: "v2",
		SHA:     "d5d0f498b388e1a0eab1cd150202f664c5738e35", // v2
	},
	"ruby/setup-ruby": {
		Repo:    "ruby/setup-ruby",
		Version: "v1",
		SHA:     "e5517072e87f198d9533967ae13d97c11b604005", // v1.99.0
	},
	"astral-sh/setup-uv": {
		Repo:    "astral-sh/setup-uv",
		Version: "v5",
		SHA:     "e58605a9b6da7c637471fab8847a5e5a6b8df081", // v5
	},

	// GitHub Actions
	"github/codeql-action/upload-sarif": {
		Repo:    "github/codeql-action/upload-sarif",
		Version: "v3",
		SHA:     "562257dc84ee23987d348302b161ee561898ec02", // v3
	},
}

// GetActionPin returns the pinned action reference for a given action repository
// It uses the golden/default version defined in actionPins
// If no pin is found, it returns an empty string
func GetActionPin(actionRepo string) string {
	if pin, exists := actionPins[actionRepo]; exists {
		return actionRepo + "@" + pin.SHA
	}
	// If no pin exists, return empty string to signal that this action is not pinned
	return ""
}

// ApplyActionPinToStep applies SHA pinning to a step map if it contains a "uses" field
// with a pinned action. Returns a modified copy of the step map with pinned references.
// If the step doesn't use an action or the action is not pinned, returns the original map.
func ApplyActionPinToStep(stepMap map[string]any) map[string]any {
	// Check if step has a "uses" field
	uses, hasUses := stepMap["uses"]
	if !hasUses {
		return stepMap
	}

	// Extract uses value as string
	usesStr, ok := uses.(string)
	if !ok {
		return stepMap
	}

	// Extract action repo from uses field (remove @version or @ref)
	actionRepo := extractActionRepo(usesStr)
	if actionRepo == "" {
		return stepMap
	}

	// Check if this action has a pin
	pinnedRef := GetActionPin(actionRepo)
	if pinnedRef == "" {
		// No pin available for this action, return original step
		return stepMap
	}

	// Create a copy of the step map with the pinned reference
	result := make(map[string]any)
	for k, v := range stepMap {
		if k == "uses" {
			result[k] = pinnedRef
		} else {
			result[k] = v
		}
	}

	return result
}

// extractActionRepo extracts the action repository from a uses string
// For example:
//   - "actions/checkout@v4" -> "actions/checkout"
//   - "actions/setup-node@v5" -> "actions/setup-node"
//   - "github/codeql-action/upload-sarif@v3" -> "github/codeql-action/upload-sarif"
//   - "actions/checkout" -> "actions/checkout"
func extractActionRepo(uses string) string {
	// Split on @ to separate repo from version/ref
	idx := strings.Index(uses, "@")
	if idx == -1 {
		// No @ found, return the whole string
		return uses
	}
	// Return everything before the @
	return uses[:idx]
}

// ApplyActionPinsToSteps applies SHA pinning to a slice of step maps
// Returns a new slice with pinned references
func ApplyActionPinsToSteps(steps []any) []any {
	result := make([]any, len(steps))
	for i, step := range steps {
		if stepMap, ok := step.(map[string]any); ok {
			result[i] = ApplyActionPinToStep(stepMap)
		} else {
			result[i] = step
		}
	}
	return result
}
