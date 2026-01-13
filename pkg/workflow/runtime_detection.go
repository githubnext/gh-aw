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

	// Detect from Serena language configuration
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.Serena != nil {
		detectFromSerenaLanguages(workflowData.ParsedTools.Serena, requirements)
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

	// Scan custom MCP tools for runtime commands
	// Skip containerized MCP servers as they don't need host runtime setup
	for _, tool := range tools.Custom {
		// Skip if the MCP server is containerized (has Container field set or Type is "docker")
		if tool.Container != "" || tool.Type == "docker" {
			runtimeSetupLog.Printf("Skipping runtime detection for containerized MCP server (container=%s, type=%s)", tool.Container, tool.Type)
			continue
		}

		// For non-containerized custom MCP servers, check the Command field
		if tool.Command != "" {
			if runtime, found := commandToRuntime[tool.Command]; found {
				updateRequiredRuntime(runtime, "", requirements)
			}
		}
	}
}

// detectFromSerenaLanguages detects runtime requirements based on Serena language configuration
// Serena now runs using uvx with HTTP transport in the agent job, requiring language runtimes
// to be installed on the host system
func detectFromSerenaLanguages(serenaConfig *SerenaToolConfig, requirements map[string]*RuntimeRequirement) {
	if serenaConfig == nil {
		return
	}

	// uvx is always required to run Serena
	uvRuntime := findRuntimeByID("uv")
	if uvRuntime != nil {
		runtimeSetupLog.Print("Serena detected, adding uv runtime requirement")
		updateRequiredRuntime(uvRuntime, "", requirements)
	}

	// Map to track which languages we need to process
	languagesToProcess := make(map[string]string) // language name -> version

	// Check short syntax (array of language names)
	if len(serenaConfig.ShortSyntax) > 0 {
		runtimeSetupLog.Printf("Detecting runtimes from Serena short syntax: %v", serenaConfig.ShortSyntax)
		for _, lang := range serenaConfig.ShortSyntax {
			languagesToProcess[lang] = "" // No version specified in short syntax
		}
	}

	// Check long syntax (detailed language configuration)
	if len(serenaConfig.Languages) > 0 {
		runtimeSetupLog.Printf("Detecting runtimes from Serena long syntax: %d languages", len(serenaConfig.Languages))
		for lang, langConfig := range serenaConfig.Languages {
			version := ""
			if langConfig != nil && langConfig.Version != "" {
				version = langConfig.Version
			}
			languagesToProcess[lang] = version
		}
	}

	// Map Serena languages to runtime IDs and add requirements
	for lang, version := range languagesToProcess {
		runtimeID := mapSerenaLanguageToRuntime(lang)
		if runtimeID == "" {
			// Language doesn't map to a known runtime (e.g., bash, markdown, yaml)
			runtimeSetupLog.Printf("Skipping Serena language '%s' - no runtime mapping", lang)
			continue
		}

		runtime := findRuntimeByID(runtimeID)
		if runtime != nil {
			runtimeSetupLog.Printf("Adding runtime requirement for Serena language '%s' -> %s (version=%s)", lang, runtimeID, version)
			updateRequiredRuntime(runtime, version, requirements)
		}
	}
}

// mapSerenaLanguageToRuntime maps a Serena language identifier to a runtime ID
// Returns empty string if the language doesn't require a runtime setup
func mapSerenaLanguageToRuntime(serenaLang string) string {
	// Mapping based on .serena/project.yml language list and available runtimes
	languageMap := map[string]string{
		// Direct mappings
		"go":         "go",
		"typescript": "node", // JavaScript/TypeScript use Node.js
		"python":     "python",
		"java":       "java",
		"ruby":       "ruby",
		"haskell":    "haskell",
		"elixir":     "elixir",
		"rust":       "rust",

		// Alternative names or variants
		"python_jedi":      "python",
		"typescript_vts":   "node",
		"ruby_solargraph":  "ruby",
		"kotlin":           "java", // Kotlin runs on JVM
		"scala":            "java", // Scala runs on JVM
		"csharp":           "dotnet",
		"csharp_omnisharp": "dotnet",
		"erlang":           "elixir", // Erlang uses same runtime setup

		// Languages that don't need runtime setup (return empty string)
		"bash":     "",
		"markdown": "",
		"yaml":     "",

		// Languages not in our runtime definitions (return empty string)
		"cpp":       "",
		"clojure":   "",
		"dart":      "",
		"elm":       "",
		"fortran":   "",
		"julia":     "",
		"lua":       "",
		"nix":       "",
		"perl":      "",
		"php":       "",
		"r":         "",
		"rego":      "",
		"swift":     "",
		"terraform": "",
		"zig":       "",
		"al":        "",
	}

	return languageMap[serenaLang]
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
