package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionPinsLog = logger.New("workflow:action_pins")

// ActionPin represents a pinned GitHub Action with its commit SHA
type ActionPin struct {
	Repo    string // e.g., "actions/checkout"
	Version string // e.g., "v5" - the golden/default version
	SHA     string // Full commit SHA for the pinned version
}

// defaultActionVersions maps action repositories to their default/recommended versions
// These versions are used when no specific version is requested
var defaultActionVersions = map[string]string{
	"actions/ai-inference":               "v1",
	"actions/cache":                      "v4",
	"actions/checkout":                   "v5",
	"actions/download-artifact":          "v6",
	"actions/github-script":              "v8",
	"actions/setup-dotnet":               "v4",
	"actions/setup-go":                   "v5",
	"actions/setup-java":                 "v4",
	"actions/setup-node":                 "v6",
	"actions/setup-python":               "v5",
	"actions/upload-artifact":            "v5",
	"astral-sh/setup-uv":                 "v5",
	"denoland/setup-deno":                "v2",
	"erlef/setup-beam":                   "v1",
	"github/codeql-action/upload-sarif":  "v3",
	"haskell-actions/setup":              "v2",
	"oven-sh/setup-bun":                  "v2",
	"ruby/setup-ruby":                    "v1",
}

// getDefaultVersion returns the default version for a given action repository
func getDefaultVersion(repo string) string {
	if version, ok := defaultActionVersions[repo]; ok {
		return version
	}
	return ""
}

// GetActionPin returns the pinned action reference for a given action repository
// It uses the default version and dynamically resolves the SHA using WorkflowData's resolver
// If no default version exists or resolution fails, it returns an action reference without SHA
// The returned reference includes a comment with the version tag (e.g., "repo@sha # v1")
// If data is nil or resolution is not available, returns "repo@version" without SHA
func GetActionPin(actionRepo string, data *WorkflowData) string {
	// Get default version for this action
	version := getDefaultVersion(actionRepo)
	if version == "" {
		actionPinsLog.Printf("No default version defined for %s", actionRepo)
		return ""
	}

	// If data is nil or no resolver available, return action@version without SHA
	if data == nil || data.ActionResolver == nil {
		actionPinsLog.Printf("No resolver available for %s@%s, returning version reference", actionRepo, version)
		return fmt.Sprintf("%s@%s", actionRepo, version)
	}

	// Use GetActionPinWithData to resolve the SHA
	pinnedRef, err := GetActionPinWithData(actionRepo, version, data)
	if err != nil {
		actionPinsLog.Printf("Failed to get action pin for %s@%s: %v, falling back to version reference", actionRepo, version, err)
		// Fallback to version reference without SHA
		return fmt.Sprintf("%s@%s", actionRepo, version)
	}

	if pinnedRef == "" {
		// Resolution returned empty, use version reference
		return fmt.Sprintf("%s@%s", actionRepo, version)
	}

	return pinnedRef
}

// GetActionPinWithData returns the pinned action reference for a given action@version
// It uses dynamic resolution via ActionResolver
// If strictMode is true and resolution fails, it returns an error
// The returned reference includes a comment with the version tag (e.g., "repo@sha # v1")
func GetActionPinWithData(actionRepo, version string, data *WorkflowData) (string, error) {
	// Use dynamic resolution if resolver is available
	if data.ActionResolver != nil {
		sha, err := data.ActionResolver.ResolveSHA(actionRepo, version)
		if err == nil && sha != "" {
			// Successfully resolved, save cache
			if data.ActionCache != nil {
				_ = data.ActionCache.Save()
			}
			return actionRepo + "@" + sha + " # " + version, nil
		}

		// Resolution failed
		if data.StrictMode {
			errMsg := fmt.Sprintf("Unable to pin action %s@%s: resolution failed", actionRepo, version)
			fmt.Fprint(os.Stderr, console.FormatErrorMessage(errMsg))
			return "", fmt.Errorf("%s", errMsg)
		}

		// In non-strict mode, emit warning and return empty string
		warningMsg := fmt.Sprintf("Unable to pin action %s@%s: resolution failed", actionRepo, version)
		fmt.Fprint(os.Stderr, console.FormatWarningMessage(warningMsg))
		return "", nil
	}

	// No resolver available
	if data.StrictMode {
		errMsg := fmt.Sprintf("Unable to pin action %s@%s: no resolver available", actionRepo, version)
		fmt.Fprint(os.Stderr, console.FormatErrorMessage(errMsg))
		return "", fmt.Errorf("%s", errMsg)
	}

	// In non-strict mode, emit warning and return empty string
	warningMsg := fmt.Sprintf("Unable to pin action %s@%s: no resolver available", actionRepo, version)
	fmt.Fprint(os.Stderr, console.FormatWarningMessage(warningMsg))
	return "", nil
}

// ApplyActionPinToStep applies SHA pinning to a step map if it contains a "uses" field
// with a pinned action. Returns a modified copy of the step map with pinned references.
// If the step doesn't use an action or the action is not pinned, returns the original map.
func ApplyActionPinToStep(stepMap map[string]any, data *WorkflowData) map[string]any {
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

	// Extract action repo and version from uses field
	actionRepo := extractActionRepo(usesStr)
	if actionRepo == "" {
		return stepMap
	}

	version := extractActionVersion(usesStr)
	if version == "" {
		// No version specified, can't pin
		return stepMap
	}

	// Try to get pinned SHA
	pinnedRef, err := GetActionPinWithData(actionRepo, version, data)
	if err != nil {
		// In strict mode, this would have already been handled by GetActionPinWithData
		// In normal mode, we just return the original step
		return stepMap
	}

	if pinnedRef == "" {
		// No pin available for this action, return original step
		return stepMap
	}

	actionPinsLog.Printf("Pinning action: %s@%s -> %s", actionRepo, version, pinnedRef)

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
//   - "actions/checkout@v5" -> "actions/checkout"
//   - "actions/setup-node@v6" -> "actions/setup-node"
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

// extractActionVersion extracts the version from a uses string
// For example:
//   - "actions/checkout@v5" -> "v4"
//   - "actions/setup-node@v6" -> "v5"
//   - "actions/checkout" -> ""
func extractActionVersion(uses string) string {
	// Split on @ to separate repo from version/ref
	idx := strings.Index(uses, "@")
	if idx == -1 {
		// No @ found, no version
		return ""
	}
	// Return everything after the @
	return uses[idx+1:]
}

// ApplyActionPinsToSteps applies SHA pinning to a slice of step maps
// Returns a new slice with pinned references
func ApplyActionPinsToSteps(steps []any, data *WorkflowData) []any {
	result := make([]any, len(steps))
	for i, step := range steps {
		if stepMap, ok := step.(map[string]any); ok {
			result[i] = ApplyActionPinToStep(stepMap, data)
		} else {
			result[i] = step
		}
	}
	return result
}

// GetActionPinByRepo returns the ActionPin for a given repository, if it exists
// It uses the default version for the repository and resolves the SHA dynamically
func GetActionPinByRepo(repo string, data *WorkflowData) (ActionPin, bool) {
	version := getDefaultVersion(repo)
	if version == "" {
		return ActionPin{}, false
	}

	// Try to resolve the SHA using the provided WorkflowData
	sha := ""
	if data != nil && data.ActionResolver != nil {
		resolvedSHA, err := data.ActionResolver.ResolveSHA(repo, version)
		if err == nil && resolvedSHA != "" {
			sha = resolvedSHA
			// Save cache
			if data.ActionCache != nil {
				_ = data.ActionCache.Save()
			}
		}
	}

	// Return the ActionPin with the version (SHA may be empty if resolution failed)
	return ActionPin{
		Repo:    repo,
		Version: version,
		SHA:     sha,
	}, true
}
