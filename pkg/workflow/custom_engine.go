package workflow

import (
	"encoding/json"
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

func (e *CustomEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Custom engine uses the same MCP configuration generation as Claude (JSON format)
	// Prepare configuration data for JavaScript script
	mcpConfigData := e.prepareMCPConfigData(tools, mcpTools, workflowData)
	
	// Set environment variables for the JavaScript script
	yaml.WriteString("          export MCP_CONFIG_FORMAT=json\n")
	
	// Add safe-outputs configuration if enabled
	if mcpConfigData.SafeOutputsConfig != nil {
		configJSON, _ := json.Marshal(mcpConfigData.SafeOutputsConfig)
		yaml.WriteString(fmt.Sprintf("          export MCP_SAFE_OUTPUTS_CONFIG='%s'\n", string(configJSON)))
	}
	
	// Add GitHub configuration if present
	if mcpConfigData.GitHubConfig != nil {
		configJSON, _ := json.Marshal(mcpConfigData.GitHubConfig)
		yaml.WriteString(fmt.Sprintf("          export MCP_GITHUB_CONFIG='%s'\n", string(configJSON)))
	}
	
	// Add Playwright configuration if present
	if mcpConfigData.PlaywrightConfig != nil {
		configJSON, _ := json.Marshal(mcpConfigData.PlaywrightConfig)
		yaml.WriteString(fmt.Sprintf("          export MCP_PLAYWRIGHT_CONFIG='%s'\n", string(configJSON)))
	}
	
	// Add custom tools configuration if present
	if len(mcpConfigData.CustomToolsConfig) > 0 {
		configJSON, _ := json.Marshal(mcpConfigData.CustomToolsConfig)
		yaml.WriteString(fmt.Sprintf("          export MCP_CUSTOM_TOOLS_CONFIG='%s'\n", string(configJSON)))
	}
	
	// Create temporary file with the JavaScript script
	yaml.WriteString("          cat > /tmp/generate-mcp-config.cjs << 'EOF'\n")
	
	// Write the JavaScript script
	scriptLines := strings.Split(GetGenerateMCPConfigScript(), "\n")
	for _, line := range scriptLines {
		if strings.TrimSpace(line) != "" {
			yaml.WriteString(fmt.Sprintf("          %s\n", line))
		}
	}
	yaml.WriteString("          EOF\n")
	
	// Execute the JavaScript script
	yaml.WriteString("          node /tmp/generate-mcp-config.cjs\n")
}

// prepareMCPConfigData prepares configuration data for the JavaScript MCP config generator
func (e *CustomEngine) prepareMCPConfigData(tools map[string]any, mcpTools []string, workflowData *WorkflowData) MCPConfigData {
	data := MCPConfigData{
		CustomToolsConfig: make(map[string]map[string]any),
	}

	// Add safe-outputs configuration if enabled
	hasSafeOutputs := workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs)
	if hasSafeOutputs {
		data.SafeOutputsConfig = map[string]any{"enabled": true}
	}

	// Process each MCP tool
	for _, toolName := range mcpTools {
		switch toolName {
		case "github":
			if githubTool, ok := tools["github"]; ok {
				data.GitHubConfig = e.prepareGitHubConfig(githubTool)
			}
		case "playwright":
			if playwrightTool, ok := tools["playwright"]; ok {
				data.PlaywrightConfig = e.preparePlaywrightConfig(playwrightTool, workflowData.NetworkPermissions)
			}
		default:
			// Handle custom MCP tools
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					data.CustomToolsConfig[toolName] = e.prepareCustomToolConfig(toolConfig)
				}
			}
		}
	}

	return data
}

// prepareGitHubConfig prepares GitHub MCP configuration data
func (e *CustomEngine) prepareGitHubConfig(githubTool any) map[string]any {
	dockerImageVersion := getGitHubDockerImageVersion(githubTool)
	return map[string]any{
		"dockerImageVersion": dockerImageVersion,
	}
}

// preparePlaywrightConfig prepares Playwright MCP configuration data
func (e *CustomEngine) preparePlaywrightConfig(playwrightTool any, networkPermissions *NetworkPermissions) map[string]any {
	config := map[string]any{}
	
	// Get docker image version
	if toolMap, ok := playwrightTool.(map[string]any); ok {
		if version, exists := toolMap["docker_image_version"]; exists {
			if versionStr, ok := version.(string); ok {
				config["dockerImageVersion"] = versionStr
			}
		}
	}

	// Add allowed domains from network permissions
	if networkPermissions != nil && len(networkPermissions.Allowed) > 0 {
		config["allowedDomains"] = networkPermissions.Allowed
	}

	return config
}

// prepareCustomToolConfig prepares custom MCP tool configuration data
func (e *CustomEngine) prepareCustomToolConfig(toolConfig map[string]any) map[string]any {
	mcpConfig, err := getMCPConfig(toolConfig, "")
	if err != nil {
		return map[string]any{}
	}

	config := map[string]any{}
	
	// Copy relevant MCP properties
	if command, exists := mcpConfig["command"]; exists {
		config["command"] = command
	}
	if args, exists := mcpConfig["args"]; exists {
		config["args"] = args
	}
	if env, exists := mcpConfig["env"]; exists {
		config["env"] = env
	}
	if url, exists := mcpConfig["url"]; exists {
		config["url"] = url
	}
	if headers, exists := mcpConfig["headers"]; exists {
		config["headers"] = headers
	}

	return config
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
