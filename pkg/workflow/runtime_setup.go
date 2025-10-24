package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// Runtime represents configuration for a runtime environment
type Runtime struct {
	ID              string            // Unique identifier (e.g., "node", "python")
	Name            string            // Display name (e.g., "Node.js", "Python")
	ActionRepo      string            // GitHub Actions repository (e.g., "actions/setup-node")
	ActionVersion   string            // Action version (e.g., "v4", without @ prefix)
	VersionField    string            // Field name for version in action (e.g., "node-version")
	DefaultVersion  string            // Default version to use
	Commands        []string          // Commands that indicate this runtime is needed
	ExtraWithFields map[string]string // Additional 'with' fields for the action
}

// RuntimeRequirement represents a detected runtime requirement
type RuntimeRequirement struct {
	Runtime *Runtime
	Version string // Empty string means use default
}

// knownRuntimes is the list of all supported runtime configurations (alphabetically sorted by ID)
var knownRuntimes = []*Runtime{
	{
		ID:             "dotnet",
		Name:           ".NET",
		ActionRepo:     "actions/setup-dotnet",
		ActionVersion:  "v4",
		VersionField:   "dotnet-version",
		DefaultVersion: constants.DefaultDotNetVersion,
		Commands:       []string{"dotnet"},
	},
	{
		ID:             "elixir",
		Name:           "Elixir",
		ActionRepo:     "erlef/setup-beam",
		ActionVersion:  "v1",
		VersionField:   "elixir-version",
		DefaultVersion: constants.DefaultElixirVersion,
		Commands:       []string{"elixir", "mix", "iex"},
		ExtraWithFields: map[string]string{
			"otp-version": "27",
		},
	},
	{
		ID:             "go",
		Name:           "Go",
		ActionRepo:     "actions/setup-go",
		ActionVersion:  "v5",
		VersionField:   "go-version",
		DefaultVersion: "", // Special handling: uses go.mod
		Commands:       []string{"go"},
	},
	{
		ID:             "haskell",
		Name:           "Haskell",
		ActionRepo:     "haskell-actions/setup",
		ActionVersion:  "v2",
		VersionField:   "ghc-version",
		DefaultVersion: constants.DefaultHaskellVersion,
		Commands:       []string{"ghc", "ghci", "cabal", "stack"},
	},
	{
		ID:             "java",
		Name:           "Java",
		ActionRepo:     "actions/setup-java",
		ActionVersion:  "v4",
		VersionField:   "java-version",
		DefaultVersion: constants.DefaultJavaVersion,
		Commands:       []string{"java", "javac", "mvn", "gradle"},
		ExtraWithFields: map[string]string{
			"distribution": "temurin",
		},
	},
	{
		ID:             "node",
		Name:           "Node.js",
		ActionRepo:     "actions/setup-node",
		ActionVersion:  "v4",
		VersionField:   "node-version",
		DefaultVersion: constants.DefaultNodeVersion,
		Commands:       []string{"node", "npm", "npx", "yarn", "pnpm"},
	},
	{
		ID:             "python",
		Name:           "Python",
		ActionRepo:     "actions/setup-python",
		ActionVersion:  "v5",
		VersionField:   "python-version",
		DefaultVersion: constants.DefaultPythonVersion,
		Commands:       []string{"python", "python3", "pip", "pip3"},
	},
	{
		ID:             "ruby",
		Name:           "Ruby",
		ActionRepo:     "ruby/setup-ruby",
		ActionVersion:  "v1",
		VersionField:   "ruby-version",
		DefaultVersion: constants.DefaultRubyVersion,
		Commands:       []string{"ruby", "gem", "bundle"},
	},
	{
		ID:             "uv",
		Name:           "uv",
		ActionRepo:     "astral-sh/setup-uv",
		ActionVersion:  "v5",
		VersionField:   "version",
		DefaultVersion: "", // Uses latest
		Commands:       []string{"uv", "uvx"},
	},
}

// commandToRuntime maps command patterns to runtime configurations
var commandToRuntime map[string]*Runtime

// actionRepoToRuntime maps action repository names to runtime configurations
var actionRepoToRuntime map[string]*Runtime

func init() {
	// Build the command to runtime mapping
	commandToRuntime = make(map[string]*Runtime)
	for _, runtime := range knownRuntimes {
		for _, cmd := range runtime.Commands {
			commandToRuntime[cmd] = runtime
		}
	}

	// Build the action repo to runtime mapping
	actionRepoToRuntime = make(map[string]*Runtime)
	for _, runtime := range knownRuntimes {
		actionRepoToRuntime[runtime.ActionRepo] = runtime
	}
}

// DetectRuntimeRequirements analyzes workflow data to detect required runtimes
func DetectRuntimeRequirements(workflowData *WorkflowData) []RuntimeRequirement {
	requirements := make(map[string]*RuntimeRequirement) // map of runtime ID -> requirement

	// Detect from custom steps
	if workflowData.CustomSteps != "" {
		detectFromCustomSteps(workflowData.CustomSteps, requirements)
	}

	// Detect from MCP server configurations
	if workflowData.Tools != nil {
		detectFromMCPConfigs(workflowData.Tools, requirements)
	}

	// Detect from engine requirements
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		detectFromEngineSteps(workflowData.EngineConfig.Steps, requirements)
	}

	// Apply runtime overrides from frontmatter
	if workflowData.Runtimes != nil {
		applyRuntimeOverrides(workflowData.Runtimes, requirements)
	}

	// Add Python as dependency when uv is detected (uv requires Python)
	if _, hasUV := requirements["uv"]; hasUV {
		if _, hasPython := requirements["python"]; !hasPython {
			pythonRuntime := findRuntimeByID("python")
			if pythonRuntime != nil {
				updateRequiredRuntime(pythonRuntime, "", requirements)
			}
		}
	}

	// Filter out runtimes that already have setup actions in custom steps or engine steps
	if workflowData.CustomSteps != "" {
		filterExistingSetupActions(workflowData.CustomSteps, requirements)
	}
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			if uses, hasUses := step["uses"]; hasUses {
				if usesStr, ok := uses.(string); ok {
					filterExistingSetupAction(usesStr, requirements)
				}
			}
		}
	}

	// Convert map to sorted slice (alphabetically by runtime ID)
	var result []RuntimeRequirement
	var runtimeIDs []string
	for id := range requirements {
		runtimeIDs = append(runtimeIDs, id)
	}
	sort.Strings(runtimeIDs)

	for _, id := range runtimeIDs {
		result = append(result, *requirements[id])
	}

	return result
}

// detectFromCustomSteps scans custom steps YAML for runtime commands
func detectFromCustomSteps(customSteps string, requirements map[string]*RuntimeRequirement) {
	lines := strings.Split(customSteps, "\n")
	for _, line := range lines {
		// Look for run: commands
		if strings.Contains(line, "run:") {
			// Extract the command part
			parts := strings.SplitN(line, "run:", 2)
			if len(parts) == 2 {
				cmdLine := strings.TrimSpace(parts[1])
				detectRuntimeFromCommand(cmdLine, requirements)
			}
		}
	}
}

// detectRuntimeFromCommand scans a command string for runtime indicators
func detectRuntimeFromCommand(cmdLine string, requirements map[string]*RuntimeRequirement) {
	// Split by common shell delimiters and operators
	words := strings.FieldsFunc(cmdLine, func(r rune) bool {
		return r == ' ' || r == '|' || r == '&' || r == ';' || r == '\n' || r == '\t'
	})

	for _, word := range words {
		// Check if this word matches a known command
		if runtime, exists := commandToRuntime[word]; exists {
			// Special handling for "uv pip" to avoid detecting pip separately
			if word == "pip" || word == "pip3" {
				// Check if "uv" appears before this pip command
				uvIndex := -1
				pipIndex := -1
				for i, w := range words {
					if w == "uv" {
						uvIndex = i
					}
					if w == word {
						pipIndex = i
						break
					}
				}
				if uvIndex >= 0 && uvIndex < pipIndex {
					// This is "uv pip", skip pip detection
					continue
				}
			}

			updateRequiredRuntime(runtime, "", requirements)
		}
	}
}

// detectFromMCPConfigs scans MCP server configurations for runtime commands
func detectFromMCPConfigs(tools map[string]any, requirements map[string]*RuntimeRequirement) {
	for _, tool := range tools {
		// Handle structured MCP config with command field
		if toolMap, ok := tool.(map[string]any); ok {
			if command, exists := toolMap["command"]; exists {
				if cmdStr, ok := command.(string); ok {
					if runtime, found := commandToRuntime[cmdStr]; found {
						updateRequiredRuntime(runtime, "", requirements)
					}
				}
			}
		} else if cmdStr, ok := tool.(string); ok {
			// Handle string-format MCP tool (e.g., "npx -y package")
			// Parse the command string to detect runtime
			detectRuntimeFromCommand(cmdStr, requirements)
		}
	}
}

// detectFromEngineSteps scans engine steps for runtime commands
func detectFromEngineSteps(steps []map[string]any, requirements map[string]*RuntimeRequirement) {
	for _, step := range steps {
		if run, hasRun := step["run"]; hasRun {
			if runStr, ok := run.(string); ok {
				detectRuntimeFromCommand(runStr, requirements)
			}
		}
	}
}

// filterExistingSetupActions removes runtimes from requirements if they already have setup actions in the custom steps
func filterExistingSetupActions(customSteps string, requirements map[string]*RuntimeRequirement) {
	for _, runtime := range knownRuntimes {
		// Check if the action repo is referenced in the custom steps
		if strings.Contains(customSteps, runtime.ActionRepo) {
			// Remove this runtime from requirements as it already has a setup action
			delete(requirements, runtime.ID)
		}
	}
}

// filterExistingSetupAction removes a runtime from requirements if it has a setup action
func filterExistingSetupAction(usesStr string, requirements map[string]*RuntimeRequirement) {
	for _, runtime := range knownRuntimes {
		if strings.Contains(usesStr, runtime.ActionRepo) {
			delete(requirements, runtime.ID)
		}
	}
}

// updateRequiredRuntime updates the version requirement, choosing the highest version
func updateRequiredRuntime(runtime *Runtime, newVersion string, requirements map[string]*RuntimeRequirement) {
	existing, exists := requirements[runtime.ID]

	if !exists {
		requirements[runtime.ID] = &RuntimeRequirement{
			Runtime: runtime,
			Version: newVersion,
		}
		return
	}

	// If new version is empty, keep existing
	if newVersion == "" {
		return
	}

	// If existing version is empty, use new version
	if existing.Version == "" {
		existing.Version = newVersion
		return
	}

	// Compare versions and keep the higher one
	if compareVersions(newVersion, existing.Version) > 0 {
		existing.Version = newVersion
	}
}

// GenerateRuntimeSetupSteps creates GitHub Actions steps for runtime setup
func GenerateRuntimeSetupSteps(requirements []RuntimeRequirement) []GitHubActionStep {
	var steps []GitHubActionStep

	for _, req := range requirements {
		steps = append(steps, generateSetupStep(req.Runtime, req.Version))
	}

	return steps
}

// generateSetupStep creates a setup step for a given runtime
func generateSetupStep(runtime *Runtime, version string) GitHubActionStep {
	// Use default version if none specified
	if version == "" {
		version = runtime.DefaultVersion
	}

	// Use SHA-pinned action reference for security
	actionRef := GetActionPin(runtime.ActionRepo, runtime.ActionVersion)

	step := GitHubActionStep{
		fmt.Sprintf("      - name: Setup %s", runtime.Name),
		fmt.Sprintf("        uses: %s", actionRef),
	}

	// Special handling for Go when no version is specified
	if runtime.ID == "go" && version == "" {
		step = append(step, "        with:")
		step = append(step, "          go-version-file: go.mod")
		step = append(step, "          cache: true")
		return step
	}

	// Add version field if we have a version
	if version != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          %s: '%s'", runtime.VersionField, version))
	} else if runtime.ID == "uv" {
		// For uv without version, no with block needed
		return step
	}

	// Add extra fields if present
	for key, value := range runtime.ExtraWithFields {
		step = append(step, fmt.Sprintf("          %s: %s", key, value))
	}

	return step
}

// ShouldSkipRuntimeSetup checks if we should skip automatic runtime setup
// Deprecated: Runtime detection now smartly filters out existing runtimes instead of skipping entirely
// This function now always returns false for backward compatibility
func ShouldSkipRuntimeSetup(workflowData *WorkflowData) bool {
	return false
}

// applyRuntimeOverrides applies runtime version overrides from frontmatter
func applyRuntimeOverrides(runtimes map[string]any, requirements map[string]*RuntimeRequirement) {
	for runtimeID, configAny := range runtimes {
		// Parse runtime configuration
		configMap, ok := configAny.(map[string]any)
		if !ok {
			continue
		}

		// Extract version from config
		versionAny, hasVersion := configMap["version"]
		var version string
		if hasVersion {
			// Convert version to string (handle both string and numeric types)
			switch v := versionAny.(type) {
			case string:
				version = v
			case int:
				version = fmt.Sprintf("%d", v)
			case float64:
				// Check if it's a whole number
				if v == float64(int(v)) {
					version = fmt.Sprintf("%d", int(v))
				} else {
					version = fmt.Sprintf("%g", v)
				}
			default:
				continue
			}
		}

		// Extract action-repo and action-version from config
		actionRepo, _ := configMap["action-repo"].(string)
		actionVersion, _ := configMap["action-version"].(string)

		// Find or create runtime requirement
		if existing, exists := requirements[runtimeID]; exists {
			// Override version for existing requirement
			if hasVersion {
				existing.Version = version
			}

			// If action-repo or action-version is specified, create a custom Runtime
			if actionRepo != "" || actionVersion != "" {
				// Clone the existing runtime to avoid modifying the global knownRuntimes
				customRuntime := &Runtime{
					ID:              existing.Runtime.ID,
					Name:            existing.Runtime.Name,
					ActionRepo:      existing.Runtime.ActionRepo,
					ActionVersion:   existing.Runtime.ActionVersion,
					VersionField:    existing.Runtime.VersionField,
					DefaultVersion:  existing.Runtime.DefaultVersion,
					Commands:        existing.Runtime.Commands,
					ExtraWithFields: existing.Runtime.ExtraWithFields,
				}

				// Apply overrides
				if actionRepo != "" {
					customRuntime.ActionRepo = actionRepo
				}
				if actionVersion != "" {
					customRuntime.ActionVersion = actionVersion
				}

				existing.Runtime = customRuntime
			}
		} else {
			// Check if this is a known runtime
			var runtime *Runtime
			for _, knownRuntime := range knownRuntimes {
				if knownRuntime.ID == runtimeID {
					// Clone the known runtime if we need to customize it
					if actionRepo != "" || actionVersion != "" {
						runtime = &Runtime{
							ID:              knownRuntime.ID,
							Name:            knownRuntime.Name,
							ActionRepo:      knownRuntime.ActionRepo,
							ActionVersion:   knownRuntime.ActionVersion,
							VersionField:    knownRuntime.VersionField,
							DefaultVersion:  knownRuntime.DefaultVersion,
							Commands:        knownRuntime.Commands,
							ExtraWithFields: knownRuntime.ExtraWithFields,
						}

						// Apply overrides
						if actionRepo != "" {
							runtime.ActionRepo = actionRepo
						}
						if actionVersion != "" {
							runtime.ActionVersion = actionVersion
						}
					} else {
						runtime = knownRuntime
					}
					break
				}
			}

			// If runtime is known or we have custom action configuration, create a new requirement
			if runtime != nil {
				requirements[runtimeID] = &RuntimeRequirement{
					Runtime: runtime,
					Version: version,
				}
			}
			// If runtime is unknown and no action-repo specified, skip it (user might have typo)
		}
	}
}

// findRuntimeByID finds a runtime configuration by its ID
func findRuntimeByID(id string) *Runtime {
	for _, runtime := range knownRuntimes {
		if runtime.ID == id {
			return runtime
		}
	}
	return nil
}
