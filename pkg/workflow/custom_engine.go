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
			supportsToolsAllowlist: false,
			supportsHTTPTransport:  false,
			supportsMaxTurns:       true,  // Custom engine supports max-turns for consistency
			supportsWebFetch:       false, // Custom engine does not have built-in web-fetch support
			supportsWebSearch:      false, // Custom engine does not have built-in web-search support
			hasDefaultConcurrency:  false, // Custom engine does NOT have default concurrency enabled
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
			envVars["GITHUB_AW_PROMPT"] = "/tmp/gh-aw/aw-prompts/prompt.txt"

			// Add GITHUB_AW_MCP_CONFIG for MCP server configuration
			// Use custom path if specified, otherwise use default
			mcpConfigPath := "/tmp/gh-aw/mcp-config/mcp-servers.json"
			if workflowData.EngineConfig != nil && workflowData.EngineConfig.MCPConfigFile != "" {
				mcpConfigPath = workflowData.EngineConfig.MCPConfigFile
			}
			envVars["GITHUB_AW_MCP_CONFIG"] = mcpConfigPath

			// Add GITHUB_AW_SAFE_OUTPUTS if safe-outputs feature is used
			if workflowData.SafeOutputs != nil {
				envVars["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"

				// Add staged flag if specified
				if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
					envVars["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "true"
				}
				if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
					envVars["GITHUB_AW_TARGET_REPO_SLUG"] = workflowData.TrialTargetRepo
				}

				// Add branch name if upload assets is configured
				if workflowData.SafeOutputs.UploadAssets != nil {
					envVars["GITHUB_AW_ASSETS_BRANCH"] = workflowData.SafeOutputs.UploadAssets.BranchName
					envVars["GITHUB_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB)
					envVars["GITHUB_AW_ASSETS_ALLOWED_EXTS"] = strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")
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
	// Determine MCP config file path and servers field name
	mcpConfigPath := "/tmp/gh-aw/mcp-config/mcp-servers.json"
	mcpServersField := "mcpServers" // Default to mcpServers format
	
	if workflowData != nil && workflowData.EngineConfig != nil {
		if workflowData.EngineConfig.MCPConfigFile != "" {
			mcpConfigPath = workflowData.EngineConfig.MCPConfigFile
		}
		
		// Support both mcp-servers-field and mcp-servers-shape
		// mcp-servers-field takes precedence if both are provided
		if workflowData.EngineConfig.MCPServersField != "" {
			mcpServersField = workflowData.EngineConfig.MCPServersField
		} else if workflowData.EngineConfig.MCPServersShape != "" {
			// Convert shape to field name
			if workflowData.EngineConfig.MCPServersShape == "array" {
				mcpServersField = "servers"
			} else {
				mcpServersField = "mcpServers"
			}
		}
	}

	// Custom engine uses the same MCP configuration generation as Claude
	yaml.WriteString(fmt.Sprintf("          cat > %s << 'EOF'\n", mcpConfigPath))
	
	// Handle different server field formats
	if mcpServersField == "servers" {
		// Extension format: { "servers": [ {...}, {...} ] }
		yaml.WriteString("          {\n")
		yaml.WriteString(fmt.Sprintf("            \"%s\": [\n", mcpServersField))
		
		// Add safe-outputs MCP server if safe-outputs are configured
		totalServers := len(mcpTools)
		serverCount := 0
		
		// Generate configuration for each MCP tool using array format
		for _, toolName := range mcpTools {
			serverCount++
			isLast := serverCount == totalServers
			
			switch toolName {
			case "github":
				githubTool := tools["github"]
				e.renderGitHubMCPConfigArrayFormat(yaml, toolName, githubTool, isLast)
			case "playwright":
				playwrightTool := tools["playwright"]
				e.renderPlaywrightMCPConfigArrayFormat(yaml, toolName, playwrightTool, isLast)
			case "cache-memory":
				e.renderCacheMemoryMCPConfigArrayFormat(yaml, toolName, isLast, workflowData)
			case "safe-outputs":
				e.renderSafeOutputsMCPConfigArrayFormat(yaml, toolName, isLast)
			case "web-fetch":
				e.renderWebFetchMCPConfigArrayFormat(yaml, toolName, isLast)
			default:
				// Handle custom MCP tools (those with MCP-compatible type)
				if toolConfig, ok := tools[toolName].(map[string]any); ok {
					if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
						if err := e.renderCustomMCPConfigArrayFormat(yaml, toolName, toolConfig, isLast); err != nil {
							fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
						}
					}
				}
			}
		}
		
		yaml.WriteString("            ]\n")
		yaml.WriteString("          }\n")
	} else {
		// Default mcpServers format: { "mcpServers": { "name": {...}, ... } }
		yaml.WriteString("          {\n")
		yaml.WriteString(fmt.Sprintf("            \"%s\": {\n", mcpServersField))
		
		// Add safe-outputs MCP server if safe-outputs are configured
		totalServers := len(mcpTools)
		serverCount := 0
		
		// Generate configuration for each MCP tool using shared logic
		for _, toolName := range mcpTools {
			serverCount++
			isLast := serverCount == totalServers
			
			switch toolName {
			case "github":
				githubTool := tools["github"]
				e.renderGitHubMCPConfig(yaml, githubTool, isLast)
			case "playwright":
				playwrightTool := tools["playwright"]
				e.renderPlaywrightMCPConfig(yaml, playwrightTool, isLast)
			case "cache-memory":
				e.renderCacheMemoryMCPConfig(yaml, isLast, workflowData)
			case "safe-outputs":
				e.renderSafeOutputsMCPConfig(yaml, isLast)
			case "web-fetch":
				renderMCPFetchServerConfig(yaml, "json", "              ", isLast, false)
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
	}
	
	yaml.WriteString("          EOF\n")
}

// renderGitHubMCPConfig generates the GitHub MCP server configuration using shared logic
func (e *CustomEngine) renderGitHubMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
	customArgs := getGitHubCustomArgs(githubTool)
	readOnly := getGitHubReadOnly(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Always use Docker-based GitHub MCP server (services mode has been removed)
	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	if readOnly {
		yaml.WriteString("                  \"-e\",\n")
		yaml.WriteString("                  \"GITHUB_READ_ONLY=1\",\n")
	}
	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"")

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "                  ")

	yaml.WriteString("\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration using shared logic
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
func (e *CustomEngine) renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	yaml.WriteString("              \"playwright\": {\n")
	yaml.WriteString("                \"command\": \"npx\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"@playwright/mcp@latest\",\n")
	yaml.WriteString("                  \"--output-dir\",\n")
	yaml.WriteString("                  \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                  \"--allowed-origins\",\n")
		yaml.WriteString("                  \"" + strings.Join(args.AllowedDomains, ";") + "\"")
	}

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "                  ")

	yaml.WriteString("\n")
	yaml.WriteString("                ]\n")

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

// renderCacheMemoryMCPConfig generates the Memory MCP server configuration using shared logic
// Uses Docker-based @modelcontextprotocol/server-memory setup
// renderCacheMemoryMCPConfig handles cache-memory configuration without MCP server mounting
// Cache-memory is now a simple file share, not an MCP server
func (e *CustomEngine) renderCacheMemoryMCPConfig(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
	// Cache-memory no longer uses MCP server mounting
	// The cache folder is available as a simple file share at /tmp/gh-aw/cache-memory/
	// The folder is created by the cache step and is accessible to all tools
	// No MCP configuration is needed for simple file access
}

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
func (e *CustomEngine) renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	yaml.WriteString("              \"safe_outputs\": {\n")
	yaml.WriteString("                \"command\": \"node\",\n")
	yaml.WriteString("                \"args\": [\"/tmp/gh-aw/safe-outputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS\": \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS_CONFIG\": ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }},\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_BRANCH\": \"${{ env.GITHUB_AW_ASSETS_BRANCH }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_MAX_SIZE_KB\": \"${{ env.GITHUB_AW_ASSETS_MAX_SIZE_KB }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_ALLOWED_EXTS\": \"${{ env.GITHUB_AW_ASSETS_ALLOWED_EXTS }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
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

		// Custom engine continues with basic processing
	}

	// Note: Custom engine doesn't collect individual errors - this is handled
	// by the Claude/Codex parsers if their log formats are detected

	return metrics
}

// GetLogParserScriptId returns the JavaScript script name for parsing custom engine logs
func (e *CustomEngine) GetLogParserScriptId() string {
	return "parse_custom_log"
}

// Array format rendering functions for servers array MCP configuration

// renderGitHubMCPConfigArrayFormat generates GitHub MCP server configuration in servers array format
func (e *CustomEngine) renderGitHubMCPConfigArrayFormat(yaml *strings.Builder, toolName string, githubTool any, isLast bool) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
	customArgs := getGitHubCustomArgs(githubTool)
	readOnly := getGitHubReadOnly(githubTool)

	yaml.WriteString("              {\n")
	yaml.WriteString(fmt.Sprintf("                \"name\": \"%s\",\n", toolName))
	yaml.WriteString("                \"transport\": \"stdio\",\n")
	yaml.WriteString("                \"command\": [\n")
	yaml.WriteString("                  \"docker\",\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	if readOnly {
		yaml.WriteString("                  \"-e\",\n")
		yaml.WriteString("                  \"GITHUB_READ_ONLY=1\",\n")
	}
	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"")

	// Append custom args if present
	if len(customArgs) > 0 {
		for _, arg := range customArgs {
			yaml.WriteString(",\n")
			yaml.WriteString(fmt.Sprintf("                  \"%s\"", arg))
		}
	}

	yaml.WriteString("\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightMCPConfigArrayFormat generates Playwright MCP server configuration in servers array format
func (e *CustomEngine) renderPlaywrightMCPConfigArrayFormat(yaml *strings.Builder, toolName string, playwrightTool any, isLast bool) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	yaml.WriteString("              {\n")
	yaml.WriteString(fmt.Sprintf("                \"name\": \"%s\",\n", toolName))
	yaml.WriteString("                \"transport\": \"stdio\",\n")
	yaml.WriteString("                \"command\": [\n")
	yaml.WriteString("                  \"npx\",\n")
	yaml.WriteString("                  \"@playwright/mcp@latest\",\n")
	yaml.WriteString("                  \"--output-dir\",\n")
	yaml.WriteString("                  \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                  \"--allowed-origins\",\n")
		yaml.WriteString("                  \"" + strings.Join(args.AllowedDomains, ";") + "\"")
	}

	// Append custom args if present
	if len(customArgs) > 0 {
		for _, arg := range customArgs {
			yaml.WriteString(",\n")
			yaml.WriteString(fmt.Sprintf("                  \"%s\"", arg))
		}
	}

	yaml.WriteString("\n")
	yaml.WriteString("                ]\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderCacheMemoryMCPConfigArrayFormat generates cache-memory MCP configuration in servers array format
// (currently no-op as cache-memory is a simple file share)
func (e *CustomEngine) renderCacheMemoryMCPConfigArrayFormat(yaml *strings.Builder, toolName string, isLast bool, workflowData *WorkflowData) {
	// Cache-memory no longer uses MCP server mounting
}

// renderSafeOutputsMCPConfigArrayFormat generates Safe Outputs MCP server configuration in servers array format
func (e *CustomEngine) renderSafeOutputsMCPConfigArrayFormat(yaml *strings.Builder, toolName string, isLast bool) {
	yaml.WriteString("              {\n")
	yaml.WriteString(fmt.Sprintf("                \"name\": \"%s\",\n", toolName))
	yaml.WriteString("                \"transport\": \"stdio\",\n")
	yaml.WriteString("                \"command\": [\n")
	yaml.WriteString("                  \"node\",\n")
	yaml.WriteString("                  \"/tmp/gh-aw/safe-outputs/mcp-server.cjs\"\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS\": \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS_CONFIG\": ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }},\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_BRANCH\": \"${{ env.GITHUB_AW_ASSETS_BRANCH }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_MAX_SIZE_KB\": \"${{ env.GITHUB_AW_ASSETS_MAX_SIZE_KB }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_ALLOWED_EXTS\": \"${{ env.GITHUB_AW_ASSETS_ALLOWED_EXTS }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderWebFetchMCPConfigArrayFormat generates web-fetch MCP server configuration in servers array format
func (e *CustomEngine) renderWebFetchMCPConfigArrayFormat(yaml *strings.Builder, toolName string, isLast bool) {
	yaml.WriteString("              {\n")
	yaml.WriteString(fmt.Sprintf("                \"name\": \"%s\",\n", toolName))
	yaml.WriteString("                \"transport\": \"stdio\",\n")
	yaml.WriteString("                \"command\": [\n")
	yaml.WriteString("                  \"npx\",\n")
	yaml.WriteString("                  \"-y\",\n")
	yaml.WriteString("                  \"@modelcontextprotocol/server-fetch\"\n")
	yaml.WriteString("                ]\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderCustomMCPConfigArrayFormat generates custom MCP server configuration in servers array format
func (e *CustomEngine) renderCustomMCPConfigArrayFormat(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	// Get MCP configuration
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}

	yaml.WriteString("              {\n")
	yaml.WriteString(fmt.Sprintf("                \"name\": \"%s\",\n", toolName))
	yaml.WriteString("                \"transport\": \"stdio\",\n")

	// Build command array from command + args
	if mcpConfig.Command != "" {
		yaml.WriteString("                \"command\": [\n")
		yaml.WriteString(fmt.Sprintf("                  \"%s\"", mcpConfig.Command))
		if len(mcpConfig.Args) > 0 {
			for _, arg := range mcpConfig.Args {
				yaml.WriteString(",\n")
				yaml.WriteString(fmt.Sprintf("                  \"%s\"", arg))
			}
		}
		yaml.WriteString("\n")
		yaml.WriteString("                ]")
	}

	// Add environment variables if present
	if len(mcpConfig.Env) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                \"env\": {\n")
		
		envKeys := make([]string, 0, len(mcpConfig.Env))
		for key := range mcpConfig.Env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)
		
		for i, key := range envKeys {
			comma := ","
			if i == len(envKeys)-1 {
				comma = ""
			}
			yaml.WriteString(fmt.Sprintf("                  \"%s\": \"%s\"%s\n", key, mcpConfig.Env[key], comma))
		}
		yaml.WriteString("                }")
	}

	// Add allowed tools if specified
	if len(mcpConfig.Allowed) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                \"tools\": [\n")
		for i, tool := range mcpConfig.Allowed {
			comma := ","
			if i == len(mcpConfig.Allowed)-1 {
				comma = ""
			}
			yaml.WriteString(fmt.Sprintf("                  \"%s\"%s\n", tool, comma))
		}
		yaml.WriteString("                ]")
	}

	yaml.WriteString("\n")
	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}
