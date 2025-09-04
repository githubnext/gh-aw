package workflow

import (
	"fmt"
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

// GetExecutionConfig returns the execution configuration for custom steps
func (e *CustomEngine) GetExecutionConfig(workflowData *WorkflowData, logFile string) ExecutionConfig {
	// The custom engine doesn't execute itself - the steps are handled directly by the compiler
	// This method is called but the actual execution logic is handled in the compiler
	config := ExecutionConfig{
		StepName: "Custom Steps Execution",
		Command:  "echo \"Custom steps are handled directly by the compiler\"",
		Environment: map[string]string{
			"WORKFLOW_NAME": workflowData.Name,
		},
	}

	// If the engine configuration has custom steps, include them in the execution config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		config.Steps = workflowData.EngineConfig.Steps
	}

	return config
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

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, isLast, renderer)
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
