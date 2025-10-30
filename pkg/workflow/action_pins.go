package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionPinsLog = logger.New("workflow:action_pins")

//go:embed data/action_pins.json
var actionPinsJSON []byte

const (
	// CustomActionPinsFileName is the name of the custom action pins file in .github/aw/
	CustomActionPinsFileName = "action-pins.json"
	// NonBuiltinPinsFileName is the name of the file that stores pins not in the builtin list
	NonBuiltinPinsFileName = "action-pins-custom.json"
)

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

// ActionPinManager manages both builtin and custom action pins
type ActionPinManager struct {
	builtinPins map[string]ActionPin // Builtin pins from embedded JSON
	customPins  map[string]ActionPin // Custom pins from .github/aw/action-pins.json
	mergedPins  map[string]ActionPin // Merged pins (custom overrides builtin)
	repoRoot    string               // Repository root directory
}

// NewActionPinManager creates a new action pin manager
func NewActionPinManager(repoRoot string) *ActionPinManager {
	return &ActionPinManager{
		builtinPins: make(map[string]ActionPin),
		customPins:  make(map[string]ActionPin),
		mergedPins:  make(map[string]ActionPin),
		repoRoot:    repoRoot,
	}
}

// LoadBuiltinPins loads the builtin action pins from embedded JSON
func (m *ActionPinManager) LoadBuiltinPins() error {
	actionPinsLog.Print("Loading builtin action pins from embedded JSON")

	var data ActionPinsData
	if err := json.Unmarshal(actionPinsJSON, &data); err != nil {
		actionPinsLog.Printf("Failed to unmarshal builtin action pins JSON: %v", err)
		return fmt.Errorf("failed to load builtin action pins: %w", err)
	}

	for key, pin := range data.Actions {
		m.builtinPins[key] = pin
	}

	actionPinsLog.Printf("Successfully loaded %d builtin action pins", len(m.builtinPins))
	return nil
}

// LoadCustomPins loads custom action pins from .github/aw/action-pins.json
func (m *ActionPinManager) LoadCustomPins() error {
	customPinsPath := filepath.Join(m.repoRoot, ".github", "aw", CustomActionPinsFileName)
	actionPinsLog.Printf("Loading custom action pins from: %s", customPinsPath)

	data, err := os.ReadFile(customPinsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Custom pins file doesn't exist, that's OK
			actionPinsLog.Print("Custom action pins file does not exist, using only builtin pins")
			return nil
		}
		actionPinsLog.Printf("Failed to read custom action pins file: %v", err)
		return fmt.Errorf("failed to read custom action pins file: %w", err)
	}

	var customData ActionPinsData
	if err := json.Unmarshal(data, &customData); err != nil {
		actionPinsLog.Printf("Failed to unmarshal custom action pins JSON: %v", err)
		return fmt.Errorf("failed to unmarshal custom action pins JSON: %w", err)
	}

	for key, pin := range customData.Actions {
		m.customPins[key] = pin
	}

	actionPinsLog.Printf("Successfully loaded %d custom action pins", len(m.customPins))
	return nil
}

// MergePins merges builtin and custom pins, detecting conflicts
func (m *ActionPinManager) MergePins() error {
	actionPinsLog.Print("Merging builtin and custom action pins")

	// Start with builtin pins
	for key, pin := range m.builtinPins {
		m.mergedPins[key] = pin
	}

	// Merge custom pins, checking for conflicts
	for key, customPin := range m.customPins {
		if builtinPin, exists := m.builtinPins[key]; exists {
			// Conflict: same repo+version in both builtin and custom
			// This is an error because we don't know which one to use
			if builtinPin.SHA != customPin.SHA {
				actionPinsLog.Printf("Conflict detected for %s: builtin SHA %s vs custom SHA %s",
					key, builtinPin.SHA, customPin.SHA)
				return fmt.Errorf("conflict: custom pin %s@%s (SHA: %s) conflicts with builtin pin (SHA: %s)",
					customPin.Repo, customPin.Version, customPin.SHA, builtinPin.SHA)
			}
			// Same SHA, no conflict, skip
			actionPinsLog.Printf("Duplicate pin for %s with same SHA, skipping", key)
		} else {
			// No conflict, add custom pin
			m.mergedPins[key] = customPin
		}
	}

	actionPinsLog.Printf("Successfully merged pins: %d total (%d builtin, %d custom)",
		len(m.mergedPins), len(m.builtinPins), len(m.customPins))
	return nil
}

// GetMergedPins returns all merged pins as a slice
func (m *ActionPinManager) GetMergedPins() []ActionPin {
	pins := make([]ActionPin, 0, len(m.mergedPins))
	for _, pin := range m.mergedPins {
		pins = append(pins, pin)
	}

	// Sort by version (descending) then by repo name (ascending)
	for i := 0; i < len(pins); i++ {
		for j := i + 1; j < len(pins); j++ {
			// Compare versions first (descending)
			if pins[i].Version < pins[j].Version {
				pins[i], pins[j] = pins[j], pins[i]
			} else if pins[i].Version == pins[j].Version {
				// Same version, sort by repo name (ascending)
				if pins[i].Repo > pins[j].Repo {
					pins[i], pins[j] = pins[j], pins[i]
				}
			}
		}
	}

	return pins
}

// FindPin finds a pin by repo and version in the merged pins
func (m *ActionPinManager) FindPin(repo, version string) (ActionPin, bool) {
	if version != "" {
		// Try to find exact match with version first
		for _, pin := range m.mergedPins {
			if pin.Repo == repo && pin.Version == version {
				return pin, true
			}
		}
	}

	// Try to find any version of this repo
	for _, pin := range m.mergedPins {
		if pin.Repo == repo {
			return pin, true
		}
	}

	return ActionPin{}, false
}

// SaveNonBuiltinPins saves pins that are not in the builtin list to a separate file
func (m *ActionPinManager) SaveNonBuiltinPins(resolvedPins []ActionPin) error {
	nonBuiltinPinsPath := filepath.Join(m.repoRoot, ".github", "aw", NonBuiltinPinsFileName)
	actionPinsLog.Printf("Saving non-builtin action pins to: %s", nonBuiltinPinsPath)

	// Filter out builtin pins
	nonBuiltinPins := make(map[string]ActionPin)
	for _, pin := range resolvedPins {
		key := pin.Repo
		if _, isBuiltin := m.builtinPins[key]; !isBuiltin {
			nonBuiltinPins[key] = pin
		}
	}

	if len(nonBuiltinPins) == 0 {
		actionPinsLog.Print("No non-builtin pins to save")
		// If no non-builtin pins, remove the file if it exists
		if err := os.Remove(nonBuiltinPinsPath); err != nil && !os.IsNotExist(err) {
			actionPinsLog.Printf("Failed to remove non-builtin pins file: %v", err)
		}
		return nil
	}

	// Create the data structure
	data := ActionPinsData{
		Version:     "1.0",
		Description: "Custom action pins resolved during compilation (not in builtin list)",
		Actions:     nonBuiltinPins,
	}

	// Ensure directory exists
	dir := filepath.Dir(nonBuiltinPinsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		actionPinsLog.Printf("Failed to create directory for non-builtin pins: %v", err)
		return fmt.Errorf("failed to create directory for non-builtin pins: %w", err)
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		actionPinsLog.Printf("Failed to marshal non-builtin pins: %v", err)
		return fmt.Errorf("failed to marshal non-builtin pins: %w", err)
	}

	// Write to file
	if err := os.WriteFile(nonBuiltinPinsPath, jsonData, 0644); err != nil {
		actionPinsLog.Printf("Failed to write non-builtin pins file: %v", err)
		return fmt.Errorf("failed to write non-builtin pins file: %w", err)
	}

	actionPinsLog.Printf("Successfully saved %d non-builtin action pins", len(nonBuiltinPins))
	return nil
}

// Legacy functions for backward compatibility

// getActionPins unmarshals and returns the action pins from the embedded JSON
// Returns a sorted slice of action pins (by version descending, then by repo name)
// This is called on-demand rather than cached globally
// DEPRECATED: Use ActionPinManager instead
func getActionPins() []ActionPin {
	actionPinsLog.Print("Unmarshaling action pins from embedded JSON")

	var data ActionPinsData
	if err := json.Unmarshal(actionPinsJSON, &data); err != nil {
		actionPinsLog.Printf("Failed to unmarshal action pins JSON: %v", err)
		panic(fmt.Sprintf("failed to load action pins: %v", err))
	}

	// Convert map to sorted slice
	pins := make([]ActionPin, 0, len(data.Actions))
	for _, pin := range data.Actions {
		pins = append(pins, pin)
	}

	// Sort by version (descending) then by repo name (ascending)
	for i := 0; i < len(pins); i++ {
		for j := i + 1; j < len(pins); j++ {
			// Compare versions first (descending)
			if pins[i].Version < pins[j].Version {
				pins[i], pins[j] = pins[j], pins[i]
			} else if pins[i].Version == pins[j].Version {
				// Same version, sort by repo name (ascending)
				if pins[i].Repo > pins[j].Repo {
					pins[i], pins[j] = pins[j], pins[i]
				}
			}
		}
	}

	actionPinsLog.Printf("Successfully unmarshaled and sorted %d action pins from JSON", len(pins))
	return pins
}

// GetActionPin returns the pinned action reference for a given action repository
// It uses the golden/default version defined in actionPins
// If no pin is found, it returns an empty string
func GetActionPin(actionRepo string) string {
	actionPins := getActionPins()
	for _, pin := range actionPins {
		if pin.Repo == actionRepo {
			return actionRepo + "@" + pin.SHA
		}
	}
	// If no pin exists, return empty string to signal that this action is not pinned
	return ""
}

// GetActionPinWithData returns the pinned action reference for a given action@version
// It follows the new algorithm:
// 1. Check merged pins (builtin + custom) for exact match
// 2. If not found, use gh CLI to resolve SHA (with warning/error based on strict mode)
// 3. Replace version with SHA
func GetActionPinWithData(actionRepo, version string, data *WorkflowData) (string, error) {
	// First, try to find in merged pins (builtin + custom)
	if data.ActionPinManager != nil {
		if pin, found := data.ActionPinManager.FindPin(actionRepo, version); found {
			actionPinsLog.Printf("Found pin for %s@%s in merged pins: %s", actionRepo, version, pin.SHA)
			return actionRepo + "@" + pin.SHA, nil
		}
	}

	// Not in merged pins, try dynamic resolution using gh CLI
	if data.ActionResolver != nil {
		sha, err := data.ActionResolver.ResolveSHA(actionRepo, version)
		if err == nil && sha != "" {
			actionPinsLog.Printf("Successfully resolved %s@%s to SHA: %s", actionRepo, version, sha)
			// Successfully resolved, save to cache and track as non-builtin
			if data.ActionCache != nil {
				_ = data.ActionCache.Save()
			}
			// Add to resolved pins for later saving
			if data.ResolvedPins == nil {
				data.ResolvedPins = make([]ActionPin, 0)
			}
			data.ResolvedPins = append(data.ResolvedPins, ActionPin{
				Repo:    actionRepo,
				Version: version,
				SHA:     sha,
			})
			return actionRepo + "@" + sha, nil
		}
		// Resolution failed
		if data.StrictMode {
			errMsg := fmt.Sprintf("Unable to pin action %s@%s: resolution failed", actionRepo, version)
			fmt.Fprint(os.Stderr, console.FormatErrorMessage(errMsg))
			return "", fmt.Errorf("%s", errMsg)
		}
		// In non-strict mode, emit warning
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

// GetActionPinByRepo returns the ActionPin for a given repository, if it exists
func GetActionPinByRepo(repo string) (ActionPin, bool) {
	actionPins := getActionPins()
	for _, pin := range actionPins {
		if pin.Repo == repo {
			return pin, true
		}
	}
	return ActionPin{}, false
}
