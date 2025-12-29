package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var runtimeSetupLog = logger.New("workflow:runtime_setup")

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
	Runtime     *Runtime
	Version     string         // Empty string means use default
	ExtraFields map[string]any // Additional 'with' fields from user's setup step (e.g., cache settings)
	GoModFile   string         // Path to go.mod file for Go runtime (Go-specific)
}

// knownRuntimes is the list of all supported runtime configurations (alphabetically sorted by ID)
var knownRuntimes = []*Runtime{
	{
		ID:             "bun",
		Name:           "Bun",
		ActionRepo:     "oven-sh/setup-bun",
		ActionVersion:  "v2",
		VersionField:   "bun-version",
		DefaultVersion: string(constants.DefaultBunVersion),
		Commands:       []string{"bun", "bunx"},
	},
	{
		ID:             "deno",
		Name:           "Deno",
		ActionRepo:     "denoland/setup-deno",
		ActionVersion:  "v2",
		VersionField:   "deno-version",
		DefaultVersion: string(constants.DefaultDenoVersion),
		Commands:       []string{"deno"},
	},
	{
		ID:             "dotnet",
		Name:           ".NET",
		ActionRepo:     "actions/setup-dotnet",
		ActionVersion:  "v4",
		VersionField:   "dotnet-version",
		DefaultVersion: string(constants.DefaultDotNetVersion),
		Commands:       []string{"dotnet"},
	},
	{
		ID:             "elixir",
		Name:           "Elixir",
		ActionRepo:     "erlef/setup-beam",
		ActionVersion:  "v1",
		VersionField:   "elixir-version",
		DefaultVersion: string(constants.DefaultElixirVersion),
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
		DefaultVersion: string(constants.DefaultGoVersion),
		Commands:       []string{"go"},
	},
	{
		ID:             "haskell",
		Name:           "Haskell",
		ActionRepo:     "haskell-actions/setup",
		ActionVersion:  "v2",
		VersionField:   "ghc-version",
		DefaultVersion: string(constants.DefaultHaskellVersion),
		Commands:       []string{"ghc", "ghci", "cabal", "stack"},
	},
	{
		ID:             "java",
		Name:           "Java",
		ActionRepo:     "actions/setup-java",
		ActionVersion:  "v4",
		VersionField:   "java-version",
		DefaultVersion: string(constants.DefaultJavaVersion),
		Commands:       []string{"java", "javac", "mvn", "gradle"},
		ExtraWithFields: map[string]string{
			"distribution": "temurin",
		},
	},
	{
		ID:             "node",
		Name:           "Node.js",
		ActionRepo:     "actions/setup-node",
		ActionVersion:  "v6",
		VersionField:   "node-version",
		DefaultVersion: string(constants.DefaultNodeVersion),
		Commands:       []string{"node", "npm", "npx", "yarn", "pnpm"},
		ExtraWithFields: map[string]string{
			"package-manager-cache": "false", // Disable caching by default to prevent cache poisoning in release workflows
		},
	},
	{
		ID:             "python",
		Name:           "Python",
		ActionRepo:     "actions/setup-python",
		ActionVersion:  "v5",
		VersionField:   "python-version",
		DefaultVersion: string(constants.DefaultPythonVersion),
		Commands:       []string{"python", "python3", "pip", "pip3"},
	},
	{
		ID:             "ruby",
		Name:           "Ruby",
		ActionRepo:     "ruby/setup-ruby",
		ActionVersion:  "v1",
		VersionField:   "ruby-version",
		DefaultVersion: string(constants.DefaultRubyVersion),
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
	runtimeSetupLog.Print("Detecting runtime requirements from workflow data")
	requirements := make(map[string]*RuntimeRequirement) // map of runtime ID -> requirement

	// Detect from custom steps
	if workflowData.CustomSteps != "" {
		detectFromCustomSteps(workflowData.CustomSteps, requirements)
	}

	// Detect from MCP server configurations
	if workflowData.ParsedTools != nil {
		detectFromMCPConfigs(workflowData.ParsedTools, requirements)
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

	// NOTE: We intentionally DO NOT filter out runtimes that already have setup actions.
	// Instead, we will deduplicate the setup actions from CustomSteps in the compiler.
	// This ensures runtime setup steps are always added BEFORE custom steps.
	// The deduplication happens in compiler_yaml.go to remove duplicate setup actions from custom steps.

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

	if runtimeSetupLog.Enabled() {
		runtimeSetupLog.Printf("Detected %d runtime requirements: %v", len(result), runtimeIDs)
	}
	return result
}

// detectFromCustomSteps scans custom steps YAML for runtime commands
func detectFromCustomSteps(customSteps string, requirements map[string]*RuntimeRequirement) {
	log.Print("Scanning custom steps for runtime commands")
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
func detectFromMCPConfigs(tools *ToolsConfig, requirements map[string]*RuntimeRequirement) {
	if tools == nil {
		return
	}

	allTools := tools.ToMap()
	log.Printf("Scanning %d MCP configurations for runtime commands", len(allTools))

	// Special handling for Serena tool - detect language services
	if tools.Serena != nil {
		detectSerenaLanguages(tools.Serena, requirements)
	}

	// Scan custom MCP tools for runtime commands
	for _, tool := range tools.Custom {
		// MCPServerConfig has a Command field directly
		if tool.Command != "" {
			if runtime, found := commandToRuntime[tool.Command]; found {
				updateRequiredRuntime(runtime, "", requirements)
			}
		}
	}
}

// detectSerenaLanguages detects runtime requirements from Serena language configuration
func detectSerenaLanguages(serenaConfig *SerenaToolConfig, requirements map[string]*RuntimeRequirement) {
	if serenaConfig == nil {
		return
	}

	runtimeSetupLog.Print("Detecting Serena language requirements")

	// First, ensure UV is detected (Serena requires uvx)
	uvRuntime := findRuntimeByID("uv")
	if uvRuntime != nil {
		updateRequiredRuntime(uvRuntime, "", requirements)
	}

	// Collect all languages from the configuration
	var languages []string

	// Handle short syntax: ["go", "typescript"]
	if len(serenaConfig.ShortSyntax) > 0 {
		languages = serenaConfig.ShortSyntax
	} else if serenaConfig.Languages != nil {
		// Handle object syntax with languages field
		for langName := range serenaConfig.Languages {
			languages = append(languages, langName)
		}
	}

	runtimeSetupLog.Printf("Detected %d Serena languages: %v", len(languages), languages)

	// Map languages to runtime requirements
	for _, lang := range languages {
		switch lang {
		case "go":
			// Go language service requires Go runtime
			goRuntime := findRuntimeByID("go")
			if goRuntime != nil {
				// Check if there's a version or go-mod-file specified in language config
				version := ""
				goModFile := ""

				// Access structured config directly - no type assertions needed!
				if serenaConfig.Languages != nil {
					if goConfig := serenaConfig.Languages["go"]; goConfig != nil {
						version = goConfig.Version
						goModFile = goConfig.GoModFile
					}
				}

				// Create requirement with go-mod-file if specified
				req := &RuntimeRequirement{
					Runtime:   goRuntime,
					Version:   version,
					GoModFile: goModFile,
				}
				requirements[goRuntime.ID] = req
			}
		case "typescript":
			// TypeScript language service requires Node.js runtime
			nodeRuntime := findRuntimeByID("node")
			if nodeRuntime != nil {
				updateRequiredRuntime(nodeRuntime, "", requirements)
			}
		case "python":
			// Python language service requires Python runtime
			pythonRuntime := findRuntimeByID("python")
			if pythonRuntime != nil {
				updateRequiredRuntime(pythonRuntime, "", requirements)
			}
		case "java":
			// Java language service requires Java runtime
			javaRuntime := findRuntimeByID("java")
			if javaRuntime != nil {
				updateRequiredRuntime(javaRuntime, "", requirements)
			}
		case "rust":
			// Rust language service - no runtime setup needed (uses rustup from Ubuntu)
			// The language service (rust-analyzer) is typically installed separately
		case "csharp":
			// C# language service requires .NET runtime
			dotnetRuntime := findRuntimeByID("dotnet")
			if dotnetRuntime != nil {
				updateRequiredRuntime(dotnetRuntime, "", requirements)
			}
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
	runtimeSetupLog.Printf("Generating runtime setup steps for %d requirements", len(requirements))
	var steps []GitHubActionStep

	for _, req := range requirements {
		steps = append(steps, generateSetupStep(&req))
	}

	return steps
}

// GenerateSerenaLanguageServiceSteps creates installation steps for Serena language services
// This is called after runtime detection to install the language servers needed by Serena
func GenerateSerenaLanguageServiceSteps(tools *ToolsConfig) []GitHubActionStep {
	runtimeSetupLog.Print("Generating Serena language service installation steps")
	var steps []GitHubActionStep

	// Check if Serena is configured
	if tools == nil || tools.Serena == nil {
		return steps
	}

	serenaConfig := tools.Serena

	// Collect all languages from the configuration
	var languages []string

	// Handle short syntax: ["go", "typescript"]
	if len(serenaConfig.ShortSyntax) > 0 {
		languages = serenaConfig.ShortSyntax
	} else if serenaConfig.Languages != nil {
		// Handle object syntax with languages field
		for langName := range serenaConfig.Languages {
			languages = append(languages, langName)
		}
	}

	// Sort languages alphabetically to ensure deterministic order
	sort.Strings(languages)

	runtimeSetupLog.Printf("Found %d Serena languages to install: %v", len(languages), languages)

	// Generate installation steps for each language service
	for _, lang := range languages {
		switch lang {
		case "go":
			// Install gopls for Go language service
			// Check if there's a custom gopls version specified
			goplsVersion := "latest"
			if serenaConfig.Languages != nil {
				if goConfig := serenaConfig.Languages["go"]; goConfig != nil && goConfig.GoplsVersion != "" {
					goplsVersion = goConfig.GoplsVersion
				}
			}
			steps = append(steps, GitHubActionStep{
				"      - name: Install Go language service (gopls)",
				fmt.Sprintf("        run: go install golang.org/x/tools/gopls@%s", goplsVersion),
			})
		case "typescript":
			// Install TypeScript language server
			steps = append(steps, GitHubActionStep{
				"      - name: Install TypeScript language service",
				"        run: npm install -g --silent typescript-language-server typescript",
			})
		case "python":
			// Install Python language server
			steps = append(steps, GitHubActionStep{
				"      - name: Install Python language service",
				"        run: pip install --quiet python-lsp-server",
			})
		case "java":
			// Java language service typically comes with the JDK setup
			// No additional installation needed
			runtimeSetupLog.Print("Java language service (jdtls) typically bundled with JDK, skipping explicit install")
		case "rust":
			// Install rust-analyzer for Rust language service
			steps = append(steps, GitHubActionStep{
				"      - name: Install Rust language service (rust-analyzer)",
				"        run: rustup component add rust-analyzer",
			})
		case "csharp":
			// C# language service typically comes with .NET SDK
			// No additional installation needed
			runtimeSetupLog.Print("C# language service (OmniSharp) typically bundled with .NET SDK, skipping explicit install")
		}
	}

	runtimeSetupLog.Printf("Generated %d Serena language service installation steps", len(steps))
	return steps
}

// generateSetupStep creates a setup step for a given runtime requirement
func generateSetupStep(req *RuntimeRequirement) GitHubActionStep {
	runtime := req.Runtime
	version := req.Version
	runtimeSetupLog.Printf("Generating setup step for runtime: %s, version=%s", runtime.ID, version)
	// Use default version if none specified
	if version == "" {
		version = runtime.DefaultVersion
	}

	// Use SHA-pinned action reference for security if available
	actionRef := GetActionPin(runtime.ActionRepo)

	// If no pin exists (custom action repo), use the action repo with its version
	if actionRef == "" {
		if runtime.ActionVersion != "" {
			actionRef = fmt.Sprintf("%s@%s", runtime.ActionRepo, runtime.ActionVersion)
		} else {
			// Fallback to just the repo name (shouldn't happen in practice)
			actionRef = runtime.ActionRepo
		}
	}

	step := GitHubActionStep{
		fmt.Sprintf("      - name: Setup %s", runtime.Name),
		fmt.Sprintf("        uses: %s", actionRef),
	}

	// Special handling for Go when go-mod-file is explicitly specified
	if runtime.ID == "go" && req.GoModFile != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          go-version-file: %s", req.GoModFile))
		step = append(step, "          cache: true")
		// Add any extra fields from user's setup step (sorted for stable output)
		var extraKeys []string
		for key := range req.ExtraFields {
			extraKeys = append(extraKeys, key)
		}
		sort.Strings(extraKeys)
		for _, key := range extraKeys {
			valueStr := formatYAMLValue(req.ExtraFields[key])
			step = append(step, fmt.Sprintf("          %s: %s", key, valueStr))
		}
		return step
	}

	// Add version field if we have a version
	if version != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          %s: '%s'", runtime.VersionField, version))
	} else if runtime.ID == "uv" {
		// For uv without version, no with block needed (unless there are extra fields)
		if len(req.ExtraFields) == 0 {
			return step
		}
		step = append(step, "        with:")
	}

	// Merge extra fields from runtime configuration and user's setup step
	// User fields take precedence over runtime fields
	// Note: runtime.ExtraWithFields are pre-formatted strings, req.ExtraFields need formatting
	allExtraFields := make(map[string]string)

	// Add runtime extra fields (already formatted)
	for k, v := range runtime.ExtraWithFields {
		allExtraFields[k] = v
	}

	// Add user extra fields (need formatting), these override runtime fields
	for k, v := range req.ExtraFields {
		allExtraFields[k] = formatYAMLValue(v)
	}

	// Output merged extra fields in sorted key order for stable output
	var allKeys []string
	for key := range allExtraFields {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)
	for _, key := range allKeys {
		step = append(step, fmt.Sprintf("          %s: %s", key, allExtraFields[key]))
		log.Printf("  Added extra field to runtime setup: %s = %s", key, allExtraFields[key])
	}

	return step
}

// formatYAMLValue formats a value for YAML output
func formatYAMLValue(value any) string {
	switch v := value.(type) {
	case string:
		// Quote strings if they contain special characters or look like non-string types
		if v == "true" || v == "false" || v == "null" {
			return fmt.Sprintf("'%s'", v)
		}
		// Check if it's a number
		if _, err := fmt.Sscanf(v, "%f", new(float64)); err == nil {
			return fmt.Sprintf("'%s'", v)
		}
		// Return as-is for simple strings, quote for complex ones
		return fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int8:
		return fmt.Sprintf("%d", v)
	case int16:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint:
		return fmt.Sprintf("%d", v)
	case uint8:
		return fmt.Sprintf("%d", v)
	case uint16:
		return fmt.Sprintf("%d", v)
	case uint32:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%v", v)
	case float64:
		return fmt.Sprintf("%v", v)
	default:
		// For other types, convert to string and quote
		return fmt.Sprintf("'%v'", v)
	}
}

// ShouldSkipRuntimeSetup checks if we should skip automatic runtime setup
// Deprecated: Runtime detection now smartly filters out existing runtimes instead of skipping entirely
// This function now always returns false for backward compatibility
func ShouldSkipRuntimeSetup(workflowData *WorkflowData) bool {
	return false
}

// applyRuntimeOverrides applies runtime version overrides from frontmatter
func applyRuntimeOverrides(runtimes map[string]any, requirements map[string]*RuntimeRequirement) {
	runtimeSetupLog.Printf("Applying runtime overrides for %d configured runtimes", len(runtimes))
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
				runtimeSetupLog.Printf("Overriding version for runtime %s: %s", runtimeID, version)
				existing.Version = version
			}

			// If action-repo or action-version is specified, create a custom Runtime
			if actionRepo != "" || actionVersion != "" {
				runtimeSetupLog.Printf("Applying custom action config for runtime %s: repo=%s, version=%s", runtimeID, actionRepo, actionVersion)
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

// DeduplicateRuntimeSetupStepsFromCustomSteps removes runtime setup action steps from custom steps
// to avoid duplication when runtime steps are added before custom steps.
// This function parses the YAML custom steps, removes any steps that use runtime setup actions,
// and returns the deduplicated YAML.
//
// It preserves user-customized setup actions (e.g., with specific versions) and filters the corresponding
// runtime from the requirements so we don't generate a duplicate runtime setup step.
func DeduplicateRuntimeSetupStepsFromCustomSteps(customSteps string, runtimeRequirements []RuntimeRequirement) (string, []RuntimeRequirement, error) {
	if customSteps == "" || len(runtimeRequirements) == 0 {
		return customSteps, runtimeRequirements, nil
	}

	log.Printf("Deduplicating runtime setup steps from custom steps (%d runtimes)", len(runtimeRequirements))

	// Extract version comments from uses lines before unmarshaling
	// This is necessary because YAML treats "# comment" as a comment, not part of the value
	// Format: "uses: action@sha # v1.0.0" -> after unmarshal, only "action@sha" remains
	versionComments := make(map[string]string) // key: action@sha, value: # v1.0.0
	lines := strings.Split(customSteps, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "uses:") && strings.Contains(trimmed, " # ") {
			// Extract the uses value and version comment
			parts := strings.SplitN(trimmed, " # ", 2)
			if len(parts) == 2 {
				usesValue := strings.TrimSpace(strings.TrimPrefix(parts[0], "uses:"))
				versionComment := " # " + parts[1]
				versionComments[usesValue] = versionComment
			}
		}
	}

	// Parse custom steps YAML
	var stepsWrapper map[string]any
	if err := yaml.Unmarshal([]byte(customSteps), &stepsWrapper); err != nil {
		return customSteps, runtimeRequirements, fmt.Errorf("failed to parse custom workflow steps from frontmatter. Custom steps must be valid GitHub Actions step syntax. Example:\nsteps:\n  - name: Setup\n    run: echo 'hello'\n  - name: Build\n    run: make build\nError: %w", err)
	}

	stepsVal, hasSteps := stepsWrapper["steps"]
	if !hasSteps {
		return customSteps, runtimeRequirements, nil
	}

	steps, ok := stepsVal.([]any)
	if !ok {
		return customSteps, runtimeRequirements, nil
	}

	// Build map of action repos to runtime requirements
	actionRepoToReq := make(map[string]*RuntimeRequirement)
	for i := range runtimeRequirements {
		if runtimeRequirements[i].Runtime.ActionRepo != "" {
			actionRepoToReq[runtimeRequirements[i].Runtime.ActionRepo] = &runtimeRequirements[i]
			log.Printf("  Will check steps using action: %s", runtimeRequirements[i].Runtime.ActionRepo)
		}
	}

	// Track which runtimes to filter from requirements (user has custom setup)
	filteredRuntimeIDs := make(map[string]bool)

	// Filter out steps that use runtime setup actions
	// BUT: Preserve steps that have user-specified customizations
	var filteredSteps []any
	removedCount := 0
	preservedCount := 0
	for _, stepAny := range steps {
		step, ok := stepAny.(map[string]any)
		if !ok {
			filteredSteps = append(filteredSteps, stepAny)
			continue
		}

		// Check if this step uses a runtime setup action
		usesVal, hasUses := step["uses"]
		if !hasUses {
			filteredSteps = append(filteredSteps, stepAny)
			continue
		}

		usesStr, ok := usesVal.(string)
		if !ok {
			filteredSteps = append(filteredSteps, stepAny)
			continue
		}

		// Check if this uses string matches any runtime setup action
		shouldRemove := false
		shouldPreserve := false
		for actionRepo, req := range actionRepoToReq {
			if strings.Contains(usesStr, actionRepo) {
				// Check if the step has custom "with" fields that differ from defaults
				withVal, hasWith := step["with"]
				if hasWith {
					withMap, isMap := withVal.(map[string]any)
					if isMap && len(withMap) > 0 {
						// Check if this has actual user customizations beyond defaults
						hasCustomization := false

						// For Go, the standard with fields are: go-version-file and cache
						// These should NOT be considered customizations
						if req.Runtime.ID == "go" {
							// Check if there are fields other than go-version-file and cache
							for key := range withMap {
								if key != "go-version-file" && key != "cache" {
									hasCustomization = true
									break
								}
							}
							// Also check if go-version-file is NOT go.mod (custom path)
							if !hasCustomization {
								if goVersionFile, ok := withMap["go-version-file"]; ok {
									if goVersionFileStr, isStr := goVersionFile.(string); isStr {
										if goVersionFileStr != "go.mod" {
											hasCustomization = true
										}
									}
								}
							}
						} else if req.Runtime.VersionField != "" {
							// For other runtimes, check if user specified a custom version
							if userVersion, hasVersion := withMap[req.Runtime.VersionField]; hasVersion {
								userVersionStr := fmt.Sprintf("%v", userVersion)
								// Check if it differs from default or detected version
								if req.Runtime.DefaultVersion != "" && userVersionStr != req.Runtime.DefaultVersion {
									hasCustomization = true
								} else if req.Version != "" && userVersionStr != req.Version {
									hasCustomization = true
								} else if req.Runtime.DefaultVersion == "" && req.Version == "" {
									// No default and no detected version means user specified it
									hasCustomization = true
								}
							}
						}

						if hasCustomization {
							// User has truly customized the setup action - preserve it
							shouldPreserve = true
							filteredRuntimeIDs[req.Runtime.ID] = true
							log.Printf("  Preserving user-customized runtime setup step: %s", usesStr)
							preservedCount++
							break
						}

						// No customization detected, but capture extra fields to carry over
						// These are fields beyond the version field that should be preserved
						if req.ExtraFields == nil {
							req.ExtraFields = make(map[string]any)
						}
						for key, value := range withMap {
							// Skip the version field as it's handled separately
							if req.Runtime.VersionField != "" && key == req.Runtime.VersionField {
								continue
							}
							// Skip standard Go fields that will be auto-generated
							if req.Runtime.ID == "go" && (key == "go-version-file" || key == "cache") {
								continue
							}
							// Carry over any other fields
							req.ExtraFields[key] = value
							log.Printf("  Capturing extra field from setup step: %s = %v", key, value)
						}
					}
				}

				// No real customization - remove this duplicate but keep extra fields
				shouldRemove = true
				log.Printf("  Removing duplicate runtime setup step: %s", usesStr)
				removedCount++
				break
			}
		}

		if shouldPreserve || !shouldRemove {
			filteredSteps = append(filteredSteps, stepAny)
		}
	}

	if removedCount == 0 && preservedCount == 0 {
		log.Print("  No duplicate runtime setup steps found")
		return customSteps, runtimeRequirements, nil
	}

	log.Printf("  Removed %d duplicate runtime setup steps, preserved %d user-customized steps", removedCount, preservedCount)

	// Filter runtime requirements to exclude those with user-customized setup actions
	var filteredRequirements []RuntimeRequirement
	for _, req := range runtimeRequirements {
		if !filteredRuntimeIDs[req.Runtime.ID] {
			filteredRequirements = append(filteredRequirements, req)
		} else {
			log.Printf("  Excluding runtime %s from generated setup steps (user has custom setup)", req.Runtime.ID)
		}
	}

	// Convert back to YAML
	stepsWrapper["steps"] = filteredSteps
	
	// Restore version comments to steps that have them
	// This must be done before marshaling
	for i, step := range filteredSteps {
		if stepMap, ok := step.(map[string]any); ok {
			if usesVal, hasUses := stepMap["uses"]; hasUses {
				if usesStr, ok := usesVal.(string); ok {
					if versionComment, hasComment := versionComments[usesStr]; hasComment {
						// Add the version comment back
						stepMap["uses"] = usesStr + versionComment
						filteredSteps[i] = stepMap
					}
				}
			}
		}
	}
	
	deduplicatedYAML, err := yaml.Marshal(stepsWrapper)
	if err != nil {
		return customSteps, runtimeRequirements, fmt.Errorf("failed to marshal deduplicated workflow steps to YAML. Step deduplication removes duplicate runtime setup actions (like actions/setup-node) from custom steps to avoid conflicts when automatic runtime detection adds them. This optimization ensures runtime setup steps appear before custom steps. Error: %w", err)
	}

	// Remove quotes from uses values with version comments
	// The YAML marshaller quotes strings containing # (for inline version comments)
	// but GitHub Actions expects unquoted uses values
	deduplicatedStr := unquoteUsesWithComments(string(deduplicatedYAML))

	return deduplicatedStr, filteredRequirements, nil
}
