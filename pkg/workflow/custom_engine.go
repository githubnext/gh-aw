package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// CustomEngine represents a custom agentic engine that executes user-defined GitHub Actions steps
type CustomEngine struct {
	BaseEngine
}

// NewCustomEngine creates a new CustomEngine instance
func NewCustomEngine() *CustomEngine {
	return &CustomEngine{
		BaseEngine: BaseEngine{
			id:                     "custom",
			displayName:            "Custom Steps",
			description:            "Executes user-defined GitHub Actions steps",
			experimental:           false,
			supportsToolsWhitelist: false,
			supportsHTTPTransport:  false,
			supportsMaxTurns:       true, // Custom engine supports max-turns for consistency
		},
	}
}

// GetInstallationSteps returns empty installation steps since custom engine doesn't need installation
func (e *CustomEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing custom steps
func (e *CustomEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	var steps []GitHubActionStep

	// Generate each custom step if they exist, with environment variables
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		// Check if we need environment section for any step - always true now for GITHUB_AW_PROMPT
		hasEnvSection := true

		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := e.convertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}

			// Check if this step needs environment variables injected
			stepStr := stepYAML
			if hasEnvSection {
				// Add environment variables to all steps (both run and uses)
				stepStr = strings.TrimRight(stepYAML, "\n")
				stepStr += "\n        env:\n"

				// Always add GITHUB_AW_PROMPT for agentic workflows
				stepStr += "          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n"

				// Add GITHUB_AW_SAFE_OUTPUTS if safe-outputs feature is used
				if workflowData.SafeOutputs != nil {
					stepStr += "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n"
				}

				// Add GITHUB_AW_MAX_TURNS if max-turns is configured
				if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
					stepStr += fmt.Sprintf("          GITHUB_AW_MAX_TURNS: %s\n", workflowData.EngineConfig.MaxTurns)
				}

				// Add custom environment variables from engine config
				if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
					for key, value := range workflowData.EngineConfig.Env {
						stepStr += fmt.Sprintf("          %s: %s\n", key, value)
					}
				}
			}

			// Split the step YAML into lines to create a GitHubActionStep
			stepLines := strings.Split(stepStr, "\n")
			steps = append(steps, GitHubActionStep(stepLines))
		}
	}

	// Add a step to ensure the log file exists for consistency with other engines
	logStepLines := []string{
		"      - name: Ensure log file exists",
		"        run: |",
		"          echo \"Custom steps execution completed\" >> " + logFile,
		"          touch " + logFile,
	}
	steps = append(steps, GitHubActionStep(logStepLines))

	return steps
}

// convertStepToYAML converts a step map to YAML string - temporary helper
func (e *CustomEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	// Simple YAML generation for steps - this mirrors the compiler logic
	var stepYAML []string

	// Add step name
	if name, hasName := stepMap["name"]; hasName {
		if nameStr, ok := name.(string); ok {
			stepYAML = append(stepYAML, fmt.Sprintf("      - name: %s", nameStr))
		}
	}

	// Add id field if present
	if id, hasID := stepMap["id"]; hasID {
		if idStr, ok := id.(string); ok {
			stepYAML = append(stepYAML, fmt.Sprintf("        id: %s", idStr))
		}
	}

	// Add continue-on-error field if present
	if continueOnError, hasContinueOnError := stepMap["continue-on-error"]; hasContinueOnError {
		// Handle both string and boolean values for continue-on-error
		switch v := continueOnError.(type) {
		case bool:
			stepYAML = append(stepYAML, fmt.Sprintf("        continue-on-error: %t", v))
		case string:
			stepYAML = append(stepYAML, fmt.Sprintf("        continue-on-error: %s", v))
		}
	}

	// Add uses action
	if uses, hasUses := stepMap["uses"]; hasUses {
		if usesStr, ok := uses.(string); ok {
			stepYAML = append(stepYAML, fmt.Sprintf("        uses: %s", usesStr))
		}
	}

	// Add run command
	if run, hasRun := stepMap["run"]; hasRun {
		if runStr, ok := run.(string); ok {
			stepYAML = append(stepYAML, "        run: |")
			// Split command into lines and indent them properly
			runLines := strings.Split(runStr, "\n")
			for _, line := range runLines {
				stepYAML = append(stepYAML, "          "+line)
			}
		}
	}

	// Add with parameters
	if with, hasWith := stepMap["with"]; hasWith {
		if withMap, ok := with.(map[string]any); ok {
			stepYAML = append(stepYAML, "        with:")
			// Sort keys for stable output
			keys := make([]string, 0, len(withMap))
			for key := range withMap {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				stepYAML = append(stepYAML, fmt.Sprintf("          %s: %v", key, withMap[key]))
			}
		}
	}

	return strings.Join(stepYAML, "\n"), nil
}

// RenderMCPConfig renders MCP configuration using shared logic with Claude engine
func (e *CustomEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	// Custom engine uses the same MCP configuration generation as Claude
	yaml.WriteString("          cat > /tmp/mcp-config/mcp-servers.json << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Generate configuration for each MCP tool using shared logic
	for i, toolName := range mcpTools {
		isLast := i == len(mcpTools)-1

		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubMCPConfig(yaml, githubTool, isLast)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderCustomMCPConfig(yaml, toolName, toolConfig, isLast); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
			}
		}
	}

	yaml.WriteString("            }\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF\n")
}

// renderGitHubMCPConfig generates the GitHub MCP server configuration using shared logic
func (e *CustomEngine) renderGitHubMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Always use Docker-based GitHub MCP server (services mode has been removed)
	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"${{ secrets.GITHUB_TOKEN }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderCustomMCPConfig generates custom MCP server configuration using shared logic
func (e *CustomEngine) renderCustomMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	fmt.Fprintf(yaml, "              \"%s\": {\n", toolName)

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel: "                ",
		Format:      "json",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}

// ParseLogMetrics implements basic log parsing for custom engine
func (e *CustomEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics

	lines := strings.Split(logContent, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Count errors and warnings
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			metrics.ErrorCount++
		}
		if strings.Contains(lowerLine, "warning") {
			metrics.WarningCount++
		}
	}

	return metrics
}

// GetLogParserScript returns the JavaScript script name for parsing custom engine logs
func (e *CustomEngine) GetLogParserScript() string {
	return "parse_custom_log"
}
