package workflow

import (
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var runtimeSetupLog = logger.New("workflow:runtime_setup")

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
			runtimeSetupLog.Print("UV detected without Python, automatically adding Python runtime")
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

// updateRequiredRuntime updates the version requirement, choosing the highest version
func updateRequiredRuntime(runtime *Runtime, newVersion string, requirements map[string]*RuntimeRequirement) {
	existing, exists := requirements[runtime.ID]

	if !exists {
		runtimeSetupLog.Printf("Adding new runtime requirement: %s (version=%s)", runtime.ID, newVersion)
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
