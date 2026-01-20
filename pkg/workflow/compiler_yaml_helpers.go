package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var compilerYamlHelpersLog = logger.New("workflow:compiler_yaml_helpers")

// GetWorkflowIDFromPath extracts the workflow ID from a markdown file path.
// The workflow ID is the filename without the .md extension.
// Example: "/path/to/ai-moderator.md" -> "ai-moderator"
func GetWorkflowIDFromPath(markdownPath string) string {
	return strings.TrimSuffix(filepath.Base(markdownPath), ".md")
}

// convertStepToYAML converts a step map to YAML format.
// This is a method wrapper around the package-level ConvertStepToYAML function.
func (c *Compiler) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// getInstallationVersion returns the version that will be installed for the given engine.
// This matches the logic in BuildStandardNpmEngineInstallSteps.
func getInstallationVersion(data *WorkflowData, engine CodingAgentEngine) string {
	engineID := engine.GetID()
	compilerYamlHelpersLog.Printf("Getting installation version for engine: %s", engineID)

	// If version is specified in engine config, use it
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		compilerYamlHelpersLog.Printf("Using engine config version: %s", data.EngineConfig.Version)
		return data.EngineConfig.Version
	}

	// Otherwise, use the default version for the engine
	switch engineID {
	case "copilot":
		return string(constants.DefaultCopilotVersion)
	case "claude":
		return string(constants.DefaultClaudeCodeVersion)
	case "codex":
		return string(constants.DefaultCodexVersion)
	default:
		// Custom or unknown engines don't have a default version
		compilerYamlHelpersLog.Printf("No default version for custom engine: %s", engineID)
		return ""
	}
}

// generatePlaceholderSubstitutionStep generates a JavaScript-based step that performs
// safe placeholder substitution using the substitute_placeholders script.
// This replaces the multiple sed commands with a single JavaScript step.
func generatePlaceholderSubstitutionStep(yaml *strings.Builder, expressionMappings []*ExpressionMapping, indent string) {
	if len(expressionMappings) == 0 {
		return
	}

	compilerYamlHelpersLog.Printf("Generating placeholder substitution step with %d mappings", len(expressionMappings))

	// Use actions/github-script to perform the substitutions
	yaml.WriteString(indent + "- name: Substitute placeholders\n")
	fmt.Fprintf(yaml, indent+"  uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString(indent + "  env:\n")
	yaml.WriteString(indent + "    GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")

	// Add all environment variables
	for _, mapping := range expressionMappings {
		fmt.Fprintf(yaml, indent+"    %s: ${{ %s }}\n", mapping.EnvVar, mapping.Content)
	}

	yaml.WriteString(indent + "  with:\n")
	yaml.WriteString(indent + "    script: |\n")

	// Use require() to load script from copied files
	yaml.WriteString(indent + "      const substitutePlaceholders = require('" + SetupActionDestination + "/substitute_placeholders.cjs');\n")
	yaml.WriteString(indent + "      \n")
	yaml.WriteString(indent + "      // Call the substitution function\n")
	yaml.WriteString(indent + "      return await substitutePlaceholders({\n")
	yaml.WriteString(indent + "        file: process.env.GH_AW_PROMPT,\n")
	yaml.WriteString(indent + "        substitutions: {\n")

	for i, mapping := range expressionMappings {
		comma := ","
		if i == len(expressionMappings)-1 {
			comma = ""
		}
		fmt.Fprintf(yaml, indent+"          %s: process.env.%s%s\n", mapping.EnvVar, mapping.EnvVar, comma)
	}

	yaml.WriteString(indent + "        }\n")
	yaml.WriteString(indent + "      });\n")
}

// generateCheckoutActionsFolder generates the checkout step for the actions folder
// when running in dev mode and not using the action-tag feature. This is used to
// checkout the local actions before running the setup action.
//
// Additionally supports the action-folder feature to specify extra folders to checkout
// beyond the default actions folder. The action-folder value can be:
// - A single folder name as a string (e.g., "custom-actions")
// - Multiple folders as a comma-separated string (e.g., "folder1,folder2")
// - An array of folder names (e.g., ["folder1", "folder2"])
//
// Also automatically includes engine-specific folders:
// - Claude engine: .claude folder
// - Codex engine: .codex folder
// - Copilot engine: no additional folder (already using .github)
//
// Returns a slice of strings that can be appended to a steps array, where each
// string represents a line of YAML for the checkout step. Returns nil if:
// - Not in dev or script mode
// - action-tag feature is specified (uses remote actions instead)
func (c *Compiler) generateCheckoutActionsFolder(data *WorkflowData) []string {
	// Check if action-tag is specified - if so, we're using remote actions
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				// action-tag is set, use remote actions - no checkout needed
				return nil
			}
		}
	}

	// Get additional folders from action-folder feature and engine-specific folders
	additionalFolders := getActionFolders(data)

	// Script mode: checkout .github folder from githubnext/gh-aw to /tmp/gh-aw/actions-source/
	if c.actionMode.IsScript() {
		result := []string{
			"      - name: Checkout actions folder\n",
			fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")),
			"        with:\n",
			"          repository: githubnext/gh-aw\n",
			"          sparse-checkout: |\n",
			"            actions\n",
		}

		// Add additional folders
		for _, folder := range additionalFolders {
			result = append(result, fmt.Sprintf("            %s\n", folder))
		}

		result = append(result,
			"          path: /tmp/gh-aw/actions-source\n",
			"          depth: 1\n",
			"          persist-credentials: false\n",
		)
		return result
	}

	// Dev mode: checkout local actions folder
	if c.actionMode.IsDev() {
		result := []string{
			"      - name: Checkout actions folder\n",
			fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")),
			"        with:\n",
			"          sparse-checkout: |\n",
			"            actions\n",
		}

		// Add additional folders
		for _, folder := range additionalFolders {
			result = append(result, fmt.Sprintf("            %s\n", folder))
		}

		result = append(result,
			"          persist-credentials: false\n",
		)
		return result
	}

	// Release mode or other modes: no checkout needed
	return nil
}

// getActionFolders extracts additional folder names from the action-folder feature
// and automatically adds engine-specific folders based on the engine being used.
// Returns a slice of folder names (may be empty if feature not specified and no engine-specific folders).
func getActionFolders(data *WorkflowData) []string {
	if data == nil {
		return nil
	}

	var folders []string

	// Add engine-specific folders automatically
	engineID := getEngineID(data)
	engineFolder := getEngineFolderName(engineID)
	if engineFolder != "" {
		folders = append(folders, engineFolder)
	}

	// Add folders from action-folder feature
	if data.Features != nil {
		actionFolderVal, exists := data.Features["action-folder"]
		if exists && actionFolderVal != nil {
			// Handle different value types
			switch val := actionFolderVal.(type) {
			case string:
				// Single string or comma-separated string
				if val != "" {
					// Split by comma and trim whitespace
					parts := strings.Split(val, ",")
					for _, part := range parts {
						trimmed := strings.TrimSpace(part)
						if trimmed != "" {
							folders = append(folders, trimmed)
						}
					}
				}
			case []any:
				// Array of values
				for _, item := range val {
					if strItem, ok := item.(string); ok && strItem != "" {
						folders = append(folders, strings.TrimSpace(strItem))
					}
				}
			case []string:
				// Array of strings (less common but possible)
				for _, item := range val {
					if item != "" {
						folders = append(folders, strings.TrimSpace(item))
					}
				}
			}
		}
	}

	return folders
}

// getEngineID extracts the engine ID from WorkflowData
func getEngineID(data *WorkflowData) string {
	if data == nil {
		return ""
	}

	// Try EngineConfig first (preferred)
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		return data.EngineConfig.ID
	}

	// Fall back to legacy AI field
	if data.AI != "" {
		return data.AI
	}

	return ""
}

// getEngineFolderName returns the folder name for an engine
// Returns empty string for copilot (uses .github) or unknown engines
func getEngineFolderName(engineID string) string {
	switch engineID {
	case "claude":
		return ".claude"
	case "codex":
		return ".codex"
	case "copilot":
		// Copilot uses .github which is already checked out as the default
		return ""
	default:
		// Unknown or custom engines don't have a specific folder
		return ""
	}
}

// generateGitHubScriptWithRequire generates a github-script step that loads a module using require().
// Instead of repeating the global variable assignments inline, it uses the setup_globals helper function.
//
// Parameters:
//   - scriptPath: The path to the .cjs file to require (e.g., "check_stop_time.cjs")
//
// Returns a string containing the complete script content to be used in a github-script action's "script:" field.
func generateGitHubScriptWithRequire(scriptPath string) string {
	var script strings.Builder

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	script.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	script.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	script.WriteString("            const { main } = require('" + SetupActionDestination + "/" + scriptPath + "');\n")
	script.WriteString("            await main();\n")

	return script.String()
}

// generateSetupStep generates the setup step based on the action mode.
// In script mode, it runs the setup.sh script directly from the checked-out source.
// In other modes (dev/release), it uses the setup action.
//
// Parameters:
//   - setupActionRef: The action reference for setup action (e.g., "./actions/setup" or "githubnext/gh-aw/actions/setup@sha")
//   - destination: The destination path where files should be copied (e.g., SetupActionDestination)
//
// Returns a slice of strings representing the YAML lines for the setup step.
func (c *Compiler) generateSetupStep(setupActionRef string, destination string) []string {
	// Script mode: run the setup.sh script directly
	if c.actionMode.IsScript() {
		return []string{
			"      - name: Setup Scripts\n",
			"        run: |\n",
			"          bash /tmp/gh-aw/actions-source/actions/setup/setup.sh\n",
			"        env:\n",
			fmt.Sprintf("          INPUT_DESTINATION: %s\n", destination),
		}
	}

	// Dev/Release mode: use the setup action
	return []string{
		"      - name: Setup Scripts\n",
		fmt.Sprintf("        uses: %s\n", setupActionRef),
		"        with:\n",
		fmt.Sprintf("          destination: %s\n", destination),
	}
}
