package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionPinsLog = logger.New("workflow:action_pins")

//go:embed data/action_pins.json
var actionPinsJSON []byte

// ActionPin represents a pinned GitHub Action with its commit SHA
type ActionPin struct {
	Repo    string `json:"repo"`    // e.g., "actions/checkout"
	Version string `json:"version"` // e.g., "v5" - the golden/default version
	SHA     string `json:"sha"`     // Full commit SHA for the pinned version
}

// ActionPinsData represents the structure of the embedded JSON file
type ActionPinsData struct {
	Version     string               `json:"version"`
	Description string               `json:"description"`
	Actions     map[string]ActionPin `json:"actions"`
}

// getActionPins unmarshals and returns the action pins from the embedded JSON
// This is called on-demand rather than cached globally
func getActionPins() map[string]ActionPin {
	actionPinsLog.Print("Unmarshaling action pins from embedded JSON")

	var data ActionPinsData
	if err := json.Unmarshal(actionPinsJSON, &data); err != nil {
		actionPinsLog.Printf("Failed to unmarshal action pins JSON: %v", err)
		panic(fmt.Sprintf("failed to load action pins: %v", err))
	}

	actionPinsLog.Printf("Successfully unmarshaled %d action pins from JSON", len(data.Actions))
	return data.Actions
}

// GetActionPin returns the pinned action reference for a given action repository
// It uses the golden/default version defined in actionPins
// If no pin is found, it returns an empty string
func GetActionPin(actionRepo string) string {
	actionPins := getActionPins()
	if pin, exists := actionPins[actionRepo]; exists {
		return actionRepo + "@" + pin.SHA
	}
	// If no pin exists, return empty string to signal that this action is not pinned
	return ""
}

// GetActionPinWithData returns the pinned action reference for a given action@version
// It tries dynamic resolution first, then falls back to hardcoded pins
// If strictMode is true and resolution fails, it returns an error
func GetActionPinWithData(actionRepo, version string, data *WorkflowData) (string, error) {
	// First try dynamic resolution if resolver is available
	if data.ActionResolver != nil {
		sha, err := data.ActionResolver.ResolveSHA(actionRepo, version)
		if err == nil && sha != "" {
			// Successfully resolved, save cache
			if data.ActionCache != nil {
				_ = data.ActionCache.Save()
			}
			return actionRepo + "@" + sha, nil
		}
	}

	// Dynamic resolution failed, try hardcoded pins
	actionPins := getActionPins()
	if pin, exists := actionPins[actionRepo]; exists {
		// Check if the version matches the hardcoded version
		if pin.Version == version {
			return actionRepo + "@" + pin.SHA, nil
		}
		// Version mismatch, but we can still use the hardcoded SHA if we're not in strict mode
		if !data.StrictMode {
			warningMsg := fmt.Sprintf("Unable to resolve %s@%s dynamically, using hardcoded pin for %s@%s",
				actionRepo, version, actionRepo, pin.Version)
			fmt.Fprint(os.Stderr, console.FormatWarningMessage(warningMsg))
			return actionRepo + "@" + pin.SHA, nil
		}
	}

	// No pin available
	if data.StrictMode {
		errMsg := fmt.Sprintf("Unable to pin action %s@%s", actionRepo, version)
		if data.ActionResolver != nil {
			errMsg = fmt.Sprintf("Unable to pin action %s@%s: resolution failed", actionRepo, version)
		}
		fmt.Fprint(os.Stderr, console.FormatErrorMessage(errMsg))
		return "", fmt.Errorf("%s", errMsg)
	}

	// In non-strict mode, emit warning and return empty string
	warningMsg := fmt.Sprintf("Unable to pin action %s@%s", actionRepo, version)
	if data.ActionResolver != nil {
		warningMsg = fmt.Sprintf("Unable to pin action %s@%s: resolution failed", actionRepo, version)
	}
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

// extractActionVersion extracts the version from a uses string
// For example:
//   - "actions/checkout@v4" -> "v4"
//   - "actions/setup-node@v5" -> "v5"
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

// GetAllActionPinsSorted returns all action pins sorted by repository name
func GetAllActionPinsSorted() []ActionPin {
	actionPins := getActionPins()

	// Collect all pins into a slice
	pins := make([]ActionPin, 0, len(actionPins))
	for _, pin := range actionPins {
		pins = append(pins, pin)
	}

	// Sort by repository name for consistent output
	for i := 0; i < len(pins); i++ {
		for j := i + 1; j < len(pins); j++ {
			if pins[i].Repo > pins[j].Repo {
				pins[i], pins[j] = pins[j], pins[i]
			}
		}
	}

	return pins
}

// GetActionPinByRepo returns the ActionPin for a given repository, if it exists
func GetActionPinByRepo(repo string) (ActionPin, bool) {
	actionPins := getActionPins()
	pin, exists := actionPins[repo]
	return pin, exists
}
