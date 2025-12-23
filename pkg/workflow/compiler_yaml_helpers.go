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

	// Use require() to import and call the substitute_placeholders script
	yaml.WriteString(indent + "      global.core = core;\n")
	yaml.WriteString(indent + "      global.github = github;\n")
	yaml.WriteString(indent + "      global.context = context;\n")
	yaml.WriteString(indent + "      global.exec = exec;\n")
	yaml.WriteString(indent + "      global.io = io;\n")
	yaml.WriteString(indent + "      const { main } = require('" + SetupActionDestination + "/substitute_placeholders.cjs');\n")
	yaml.WriteString(indent + "      await main();\n")
}
