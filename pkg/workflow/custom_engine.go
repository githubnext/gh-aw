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

// GetExecutionSteps returns the GitHub Actions steps for executing custom steps
func (e *CustomEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	var steps []GitHubActionStep

	// Generate each custom step if they exist, with environment variables
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			// Create a copy of the step to avoid modifying the original
			stepCopy := make(map[string]any)
			for k, v := range step {
				stepCopy[k] = v
			}

			// Prepare environment variables to merge
			envVars := make(map[string]any)

			// Always add GITHUB_AW_PROMPT for agentic workflows
			envVars["GITHUB_AW_PROMPT"] = "/tmp/aw-prompts/prompt.txt"

			// Add GITHUB_AW_SAFE_OUTPUTS if safe-outputs feature is used
			if workflowData.SafeOutputs != nil {
				envVars["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"

				// Add staged flag if specified
				if workflowData.SafeOutputs.Staged != nil && *workflowData.SafeOutputs.Staged {
					envVars["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "true"
				}
			}

			// Add GITHUB_AW_MAX_TURNS if max-turns is configured
			if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
				envVars["GITHUB_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
			}

			// Add custom environment variables from engine config
			if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
				for key, value := range workflowData.EngineConfig.Env {
					envVars[key] = value
				}
			}

			// Merge environment variables into the step
			if len(envVars) > 0 {
				if existingEnv, exists := stepCopy["env"]; exists {
					// If step already has env section, merge them
					if envMap, ok := existingEnv.(map[string]any); ok {
						for key, value := range envVars {
							envMap[key] = value
						}
						stepCopy["env"] = envMap
					} else {
						// If env is not a map, replace it with our combined env
						stepCopy["env"] = envVars
					}
				} else {
					// If no env section exists, add our env vars
					stepCopy["env"] = envVars
				}
			}

			stepYAML, err := e.convertStepToYAML(stepCopy)
			if err != nil {
				// Log error but continue with other steps
				continue
			}

			// Split the step YAML into lines to create a GitHubActionStep
			stepLines := strings.Split(strings.TrimRight(stepYAML, "\n"), "\n")

			// Remove empty lines at the end
			for len(stepLines) > 0 && strings.TrimSpace(stepLines[len(stepLines)-1]) == "" {
				stepLines = stepLines[:len(stepLines)-1]
			}

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

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *CustomEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// RenderMCPConfig renders MCP configuration using shared logic with Claude engine
func (e *CustomEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
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
		case "playwright":
			playwrightTool := tools["playwright"]
			e.renderPlaywrightMCPConfig(yaml, playwrightTool, isLast, workflowData.NetworkPermissions)
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

// RenderMCPConfigFromConfigurations renders MCP configuration using pre-computed configurations
func (e *CustomEngine) RenderMCPConfigFromConfigurations(yaml *strings.Builder, configurations []MCPServerConfiguration, workflowData *WorkflowData) {
	// Custom engine uses the same MCP configuration generation as Claude
	yaml.WriteString("          cat > /tmp/mcp-config/mcp-servers.json << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Generate configuration for each MCP server
	for i, config := range configurations {
		isLast := i == len(configurations)-1

		// Render the configuration for this server
		configStr, err := config.renderForClaude()
		if err != nil {
			fmt.Printf("Error generating MCP configuration for %s: %v\n", config.Name, err)
			continue
		}

		yaml.WriteString(configStr)

		if !isLast {
			yaml.WriteString(",\n")
		} else {
			yaml.WriteString("\n")
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

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration using shared logic
// Always uses Docker-based containerized setup in GitHub Actions
func (e *CustomEngine) renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool, networkPermissions *NetworkPermissions) {
	args := generatePlaywrightDockerArgs(playwrightTool, networkPermissions)

	yaml.WriteString("              \"playwright\": {\n")
	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"--shm-size=2gb\",\n")
	yaml.WriteString("                  \"--cap-add=SYS_ADMIN\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"PLAYWRIGHT_ALLOWED_DOMAINS\",\n")
	if len(args.AllowedDomains) == 0 {
		yaml.WriteString("                  \"-e\",\n")
		yaml.WriteString("                  \"PLAYWRIGHT_BLOCK_ALL_DOMAINS\",\n")
	}
	yaml.WriteString("                  \"mcr.microsoft.com/playwright:" + args.ImageVersion + "\"\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"PLAYWRIGHT_ALLOWED_DOMAINS\": \"" + strings.Join(args.AllowedDomains, ",") + "\"")
	if len(args.AllowedDomains) == 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                  \"PLAYWRIGHT_BLOCK_ALL_DOMAINS\": \"true\"")
	}
	yaml.WriteString("\n")
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
// For custom engines, try both Claude and Codex parsing approaches to extract turn information
func (e *CustomEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics

	// First try Claude-style parsing to see if the logs are Claude-format
	registry := GetGlobalEngineRegistry()
	claudeEngine, err := registry.GetEngine("claude")
	if err == nil {
		claudeMetrics := claudeEngine.ParseLogMetrics(logContent, verbose)
		if claudeMetrics.Turns > 0 || claudeMetrics.TokenUsage > 0 || claudeMetrics.EstimatedCost > 0 {
			// Found structured data, use Claude parsing
			if verbose {
				fmt.Println("Custom engine: Using Claude-style parsing for logs")
			}
			return claudeMetrics
		}
	}

	// Try Codex-style parsing if Claude didn't yield results
	codexEngine, err := registry.GetEngine("codex")
	if err == nil {
		codexMetrics := codexEngine.ParseLogMetrics(logContent, verbose)
		if codexMetrics.Turns > 0 || codexMetrics.TokenUsage > 0 {
			// Found some data, use Codex parsing
			if verbose {
				fmt.Println("Custom engine: Using Codex-style parsing for logs")
			}
			return codexMetrics
		}
	}

	// Fall back to basic parsing if neither Claude nor Codex approaches work
	if verbose {
		fmt.Println("Custom engine: Using basic fallback parsing for logs")
	}

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
