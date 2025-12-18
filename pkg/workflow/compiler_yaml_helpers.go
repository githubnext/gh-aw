package workflow

import (
"fmt"
"strings"

"github.com/githubnext/gh-aw/pkg/constants"
)

// convertStepToYAML converts a step map to YAML format.
// This is a method wrapper around the package-level ConvertStepToYAML function.
func (c *Compiler) convertStepToYAML(stepMap map[string]any) (string, error) {
return ConvertStepToYAML(stepMap)
}

// getInstallationVersion returns the version that will be installed for the given engine.
// This matches the logic in BuildStandardNpmEngineInstallSteps.
func getInstallationVersion(data *WorkflowData, engine CodingAgentEngine) string {
// If version is specified in engine config, use it
if data.EngineConfig != nil && data.EngineConfig.Version != "" {
return data.EngineConfig.Version
}

// Otherwise, use the default version for the engine
switch engine.GetID() {
case "copilot":
return string(constants.DefaultCopilotVersion)
case "claude":
return string(constants.DefaultClaudeCodeVersion)
case "codex":
return string(constants.DefaultCodexVersion)
default:
// Custom or unknown engines don't have a default version
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

// Use actions/github-script to perform the substitutions
yaml.WriteString(indent + "- name: Substitute placeholders\n")
yaml.WriteString(indent + "  uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1\n")
yaml.WriteString(indent + "  env:\n")
yaml.WriteString(indent + "    GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")

// Add all environment variables
for _, mapping := range expressionMappings {
fmt.Fprintf(yaml, indent+"    %s: ${{ %s }}\n", mapping.EnvVar, mapping.Content)
}

yaml.WriteString(indent + "  with:\n")
yaml.WriteString(indent + "    script: |\n")

// Emit the substitute_placeholders script inline and call it
script := getSubstitutePlaceholdersScript()
scriptLines := strings.Split(script, "\n")
for _, line := range scriptLines {
yaml.WriteString(indent + "      " + line + "\n")
}

// Call the function with parameters
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
