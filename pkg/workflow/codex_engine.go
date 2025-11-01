package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// Pre-compiled regexes for Codex log parsing (performance optimization)
var (
	codexToolCallOldFormat    = regexp.MustCompile(`\] tool ([^(]+)\(`)
	codexToolCallNewFormat    = regexp.MustCompile(`^tool ([^(]+)\(`)
	codexExecCommandOldFormat = regexp.MustCompile(`\] exec (.+?) in`)
	codexExecCommandNewFormat = regexp.MustCompile(`^exec (.+?) in`)
	codexDurationPattern      = regexp.MustCompile(`in\s+(\d+(?:\.\d+)?)\s*s`)
	codexTokenUsagePattern    = regexp.MustCompile(`(?i)tokens\s+used[:\s]+(\d+)`)
)

// CodexEngine represents the Codex agentic engine (experimental)
type CodexEngine struct {
	BaseEngine
}

func NewCodexEngine() *CodexEngine {
	return &CodexEngine{
		BaseEngine: BaseEngine{
			id:                     "codex",
			displayName:            "Codex",
			description:            "Uses OpenAI Codex CLI with MCP server support",
			experimental:           true,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true,  // Codex now supports HTTP transport for remote MCP servers
			supportsMaxTurns:       false, // Codex does not support max-turns feature
			supportsWebFetch:       false, // Codex does not have built-in web-fetch support
			supportsWebSearch:      true,  // Codex has built-in web-search support
		},
	}
}

func (e *CodexEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Add secret validation step - Codex supports both CODEX_API_KEY and OPENAI_API_KEY as fallback
	secretValidation := GenerateMultiSecretValidationStep(
		[]string{"CODEX_API_KEY", "OPENAI_API_KEY"},
		"Codex",
		"https://githubnext.github.io/gh-aw/reference/engines/#openai-codex",
	)
	steps = append(steps, secretValidation)

	npmSteps := BuildStandardNpmEngineInstallSteps(
		"@openai/codex",
		constants.DefaultCodexVersion,
		"Install Codex",
		"codex",
		workflowData,
	)
	steps = append(steps, npmSteps...)
	return steps
}

// GetDeclaredOutputFiles returns the output files that Codex may produce
// Codex (written in Rust) writes logs to ~/.codex/log/codex-tui.log
func (e *CodexEngine) GetDeclaredOutputFiles() []string {
	// Return the Codex log directory for artifact collection
	// Using mcp-config folder structure for consistency with other engines
	return []string{
		"/tmp/gh-aw/mcp-config/logs/",
	}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Codex
func (e *CodexEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	// Handle custom steps if they exist in engine config
	steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)

	// Build model parameter only if specified in engineConfig
	var modelParam string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		modelParam = fmt.Sprintf("-c model=%s ", workflowData.EngineConfig.Model)
	}

	// Build search parameter if web-search tool is present
	webSearchParam := ""
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebSearch != nil {
		webSearchParam = "--search "
	}

	// See https://github.com/githubnext/gh-aw/issues/892
	fullAutoParam := " --full-auto --skip-git-repo-check " //"--dangerously-bypass-approvals-and-sandbox "

	// Build custom args parameter if specified in engineConfig
	var customArgsParam string
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		for _, arg := range workflowData.EngineConfig.Args {
			customArgsParam += arg + " "
		}
	}

	// Build the command with custom agent file prepending if specified (via imports)
	var instructionCommand string
	if workflowData.AgentFile != "" {
		// Extract markdown body from custom agent file (skip frontmatter) and prepend to prompt
		instructionCommand = fmt.Sprintf(`set -o pipefail
AGENT_CONTENT=$(awk 'BEGIN{skip=1} /^---$/{if(skip){skip=0;next}else{skip=1;next}} !skip' %s)
INSTRUCTION=$(printf "%%s\n\n%%s" "$AGENT_CONTENT" "$(cat $GH_AW_PROMPT)")
mkdir -p $CODEX_HOME/logs
codex %sexec%s%s%s"$INSTRUCTION" 2>&1 | tee %s`, workflowData.AgentFile, modelParam, webSearchParam, fullAutoParam, customArgsParam, logFile)
	} else {
		instructionCommand = fmt.Sprintf(`set -o pipefail
INSTRUCTION=$(cat $GH_AW_PROMPT)
mkdir -p $CODEX_HOME/logs
codex %sexec%s%s%s"$INSTRUCTION" 2>&1 | tee %s`, modelParam, webSearchParam, fullAutoParam, customArgsParam, logFile)
	}

	command := instructionCommand

	// Get effective GitHub token based on precedence: top-level github-token > default
	effectiveGitHubToken := getEffectiveGitHubToken("", workflowData.GitHubToken)

	env := map[string]string{
		"CODEX_API_KEY":       "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
		"GITHUB_STEP_SUMMARY": "${{ env.GITHUB_STEP_SUMMARY }}",
		"GH_AW_PROMPT":        "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GH_AW_MCP_CONFIG":    "/tmp/gh-aw/mcp-config/config.toml",
		"CODEX_HOME":          "/tmp/gh-aw/mcp-config",
		"RUST_LOG":            "trace,hyper_util=info,mio=info,reqwest=info,os_info=info,codex_otel=warn,codex_core=debug,ocodex_exec=debug",
		"GH_AW_GITHUB_TOKEN":  effectiveGitHubToken,
	}

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Add GH_AW_STARTUP_TIMEOUT environment variable (in seconds) if startup-timeout is specified
	if workflowData.ToolsStartupTimeout > 0 {
		env["GH_AW_STARTUP_TIMEOUT"] = fmt.Sprintf("%d", workflowData.ToolsStartupTimeout)
	}

	// Add GH_AW_TOOL_TIMEOUT environment variable (in seconds) if timeout is specified
	if workflowData.ToolsTimeout > 0 {
		env["GH_AW_TOOL_TIMEOUT"] = fmt.Sprintf("%d", workflowData.ToolsTimeout)
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}

	// Generate the step for Codex execution
	stepName := "Run Codex"
	var stepLines []string

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))

	// Format step with command and environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// expandNeutralToolsToCodexTools converts neutral tools to Codex-specific tools format
// This ensures that playwright tools get the same allowlist as the copilot agent
func (e *CodexEngine) expandNeutralToolsToCodexTools(tools map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy all existing tools
	for key, value := range tools {
		result[key] = value
	}

	// Handle playwright tool by converting it to an MCP tool configuration with copilot agent tools
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		// Create playwright as an MCP tool with the same tools available as copilot agent
		playwrightMCP := map[string]any{
			"allowed": GetCopilotAgentPlaywrightTools(),
		}
		// If the original playwright tool has additional configuration (like version),
		// preserve it while adding the allowed tools
		if playwrightConfig, ok := tools["playwright"].(map[string]any); ok {
			for key, value := range playwrightConfig {
				playwrightMCP[key] = value
			}
		}
		// Always set the allowed tools to match copilot agent
		playwrightMCP["allowed"] = GetCopilotAgentPlaywrightTools()
		result["playwright"] = playwrightMCP
	}

	return result
}

func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Build MCP configuration as JSON structure
	mcpConfig := make(map[string]any)

	// Add history configuration to disable persistence
	mcpConfig["history"] = map[string]string{
		"persistence": "none",
	}

	// Expand neutral tools (like playwright: null) to include the copilot agent tools
	expandedTools := e.expandNeutralToolsToCodexTools(tools)

	// Build mcp_servers section
	mcpServers := make(map[string]any)

	// Generate each MCP server configuration
	for _, toolName := range mcpTools {
		var serverConfig map[string]any
		switch toolName {
		case "github":
			githubTool := expandedTools["github"]
			serverConfig = e.buildGitHubCodexMCPConfig(githubTool, workflowData)
		case "playwright":
			playwrightTool := expandedTools["playwright"]
			serverConfig = e.buildPlaywrightCodexMCPConfig(playwrightTool)
		case "agentic-workflows":
			serverConfig = e.buildAgenticWorkflowsCodexMCPConfig()
		case "safe-outputs":
			serverConfig = e.buildSafeOutputsCodexMCPConfig(workflowData)
		case "web-fetch":
			serverConfig = e.buildWebFetchCodexMCPConfig()
		default:
			// Handle custom MCP tools
			if toolConfig, ok := expandedTools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					var err error
					serverConfig, err = e.buildCustomCodexMCPConfig(toolName, toolConfig)
					if err != nil {
						// Skip this server if there's an error
						continue
					}
				}
			}
		}

		// Only add non-nil configurations
		if serverConfig != nil {
			mcpServers[toolName] = serverConfig
		}
	}

	mcpConfig["mcp_servers"] = mcpServers

	// Append custom config if provided
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Config != "" {
		mcpConfig["custom_config"] = workflowData.EngineConfig.Config
	}

	// Serialize to JSON
	mcpConfigJSON, err := json.Marshal(mcpConfig)
	if err != nil {
		// Fallback to empty config if marshaling fails
		mcpConfigJSON = []byte("{}")
	}

	// Use JavaScript script to generate TOML config file
	yaml.WriteString("      - name: Generate Codex MCP configuration\n")
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	yaml.WriteString("        env:\n")
	// Use YAML block scalar for JSON to handle special characters
	yaml.WriteString("          GH_AW_MCP_CONFIG_JSON: |\n")
	yaml.WriteString("            " + string(mcpConfigJSON) + "\n")
	yaml.WriteString("          GH_AW_MCP_CONFIG: /tmp/gh-aw/mcp-config/config.toml\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	WriteJavaScriptToYAML(yaml, generateCodexConfigScript)
}

// ParseLogMetrics implements engine-specific log parsing for Codex
func (e *CodexEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")
	turns := 0
	inThinkingSection := false
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls
	var currentSequence []string                  // Track tool sequence

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect thinking sections as indicators of turns
		// Support both old format: "] thinking" and new Rust format: "thinking" (standalone line)
		trimmedLine := strings.TrimSpace(line)
		if strings.Contains(line, "] thinking") || trimmedLine == "thinking" {
			if !inThinkingSection {
				turns++
				inThinkingSection = true
				// Start of a new thinking section, save previous sequence if any
				if len(currentSequence) > 0 {
					metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
					currentSequence = []string{}
				}
			}
		} else if strings.Contains(line, "] tool") || strings.Contains(line, "] exec") || strings.Contains(line, "] codex") ||
			strings.HasPrefix(trimmedLine, "tool ") || strings.HasPrefix(trimmedLine, "exec ") {
			inThinkingSection = false
		}

		// Extract tool calls from Codex logs and add to sequence
		if toolName := e.parseCodexToolCallsWithSequence(line, toolCallMap); toolName != "" {
			currentSequence = append(currentSequence, toolName)
		}

		// Extract Codex-specific token usage (always sum for Codex)
		if tokenUsage := e.extractCodexTokenUsage(line); tokenUsage > 0 {
			totalTokenUsage += tokenUsage
		}

		// Basic processing - error/warning counting moved to end of function
	}

	// Add final sequence if any
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}

	metrics.TokenUsage = totalTokenUsage
	metrics.Turns = turns

	// Convert tool call map to slice
	for _, toolInfo := range toolCallMap {
		metrics.ToolCalls = append(metrics.ToolCalls, *toolInfo)
	}

	// Sort tool calls by name for consistent output
	sort.Slice(metrics.ToolCalls, func(i, j int) bool {
		return metrics.ToolCalls[i].Name < metrics.ToolCalls[j].Name
	})

	// Count errors and warnings using pattern matching for better accuracy
	errorPatterns := e.GetErrorPatterns()
	if len(errorPatterns) > 0 {
		metrics.Errors = CountErrorsAndWarningsWithPatterns(logContent, errorPatterns)
	}

	return metrics
}

// parseCodexToolCallsWithSequence extracts tool call information from Codex log lines and returns tool name
func (e *CodexEngine) parseCodexToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	trimmedLine := strings.TrimSpace(line)

	// Parse tool calls: "] tool provider.method(...)" (old format)
	// or "tool provider.method(...)" (new Rust format)
	var toolName string

	// Try old format first: "] tool provider.method(...)"
	if strings.Contains(line, "] tool ") && strings.Contains(line, "(") {
		if match := codexToolCallOldFormat.FindStringSubmatch(line); len(match) > 1 {
			toolName = strings.TrimSpace(match[1])
		}
	}

	// Try new Rust format: "tool provider.method(...)"
	if toolName == "" && strings.HasPrefix(trimmedLine, "tool ") && strings.Contains(trimmedLine, "(") {
		if match := codexToolCallNewFormat.FindStringSubmatch(trimmedLine); len(match) > 1 {
			toolName = strings.TrimSpace(match[1])
		}
	}

	if toolName != "" {
		prettifiedName := PrettifyToolName(toolName)

		// For Codex, format provider.method as provider_method (avoiding colons)
		if strings.Contains(toolName, ".") {
			parts := strings.Split(toolName, ".")
			if len(parts) >= 2 {
				provider := parts[0]
				method := strings.Join(parts[1:], "_")
				prettifiedName = fmt.Sprintf("%s_%s", provider, method)
			}
		}

		// Initialize or update tool call info
		if toolInfo, exists := toolCallMap[prettifiedName]; exists {
			toolInfo.CallCount++
		} else {
			toolCallMap[prettifiedName] = &ToolCallInfo{
				Name:          prettifiedName,
				CallCount:     1,
				MaxOutputSize: 0, // TODO: Extract output size from results if available
				MaxDuration:   0, // Will be updated when duration is found
			}
		}

		return prettifiedName
	}

	// Parse exec commands: "] exec command" (old format)
	// or "exec command in" (new Rust format) - treat as bash calls
	var execCommand string

	// Try old format: "] exec command in"
	if strings.Contains(line, "] exec ") {
		if match := codexExecCommandOldFormat.FindStringSubmatch(line); len(match) > 1 {
			execCommand = strings.TrimSpace(match[1])
		}
	}

	// Try new Rust format: "exec command in"
	if execCommand == "" && strings.HasPrefix(trimmedLine, "exec ") {
		if match := codexExecCommandNewFormat.FindStringSubmatch(trimmedLine); len(match) > 1 {
			execCommand = strings.TrimSpace(match[1])
		}
	}

	if execCommand != "" {
		// Create unique bash entry with command info, avoiding colons
		uniqueBashName := fmt.Sprintf("bash_%s", ShortenCommand(execCommand))

		// Initialize or update tool call info
		if toolInfo, exists := toolCallMap[uniqueBashName]; exists {
			toolInfo.CallCount++
		} else {
			toolCallMap[uniqueBashName] = &ToolCallInfo{
				Name:          uniqueBashName,
				CallCount:     1,
				MaxOutputSize: 0,
				MaxDuration:   0, // Will be updated when duration is found
			}
		}

		return uniqueBashName
	}

	// Parse duration from success/failure lines: "] success in 0.2s" or "] failure in 1.5s"
	if strings.Contains(line, "success in") || strings.Contains(line, "failure in") || strings.Contains(line, "failed in") {
		// Extract duration pattern like "in 0.2s", "in 1.5s"
		if match := codexDurationPattern.FindStringSubmatch(line); len(match) > 1 {
			if durationSeconds, err := strconv.ParseFloat(match[1], 64); err == nil {
				duration := time.Duration(durationSeconds * float64(time.Second))

				// Find the most recent tool call to associate with this duration
				// Since we don't have direct association, we'll update the most recent entry
				// This is a limitation of the log format, but it's the best we can do
				e.updateMostRecentToolWithDuration(toolCallMap, duration)
			}
		}
	}

	return "" // No tool call found
}

// updateMostRecentToolWithDuration updates the tool with maximum duration
// Since we can't perfectly correlate duration lines with specific tool calls in Codex logs,
// we approximate by updating any tool that doesn't have a duration yet, or updating the max
func (e *CodexEngine) updateMostRecentToolWithDuration(toolCallMap map[string]*ToolCallInfo, duration time.Duration) {
	// Find a tool that either has no duration yet or can be updated with a larger duration
	for _, toolInfo := range toolCallMap {
		if toolInfo.MaxDuration == 0 || duration > toolInfo.MaxDuration {
			toolInfo.MaxDuration = duration
			// Only update one tool per duration line to avoid over-attribution
			break
		}
	}
}

// extractCodexTokenUsage extracts token usage from Codex-specific log lines
func (e *CodexEngine) extractCodexTokenUsage(line string) int {
	// Codex format: "tokens used: 13934"
	// Use pre-compiled pattern for performance
	if match := codexTokenUsagePattern.FindStringSubmatch(line); len(match) > 1 {
		if count, err := strconv.Atoi(match[1]); err == nil {
			return count
		}
	}
	return 0
}

// renderGitHubCodexMCPConfig generates GitHub MCP server configuration for codex config.toml
// Supports both local (Docker) and remote (hosted) modes
func (e *CodexEngine) renderGitHubCodexMCPConfig(yaml *strings.Builder, githubTool any, workflowData *WorkflowData) {
	githubType := getGitHubType(githubTool)
	customGitHubToken := getGitHubToken(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	toolsets := getGitHubToolsets(githubTool)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.github]\n")

	// Add user_agent field defaulting to workflow identifier
	userAgent := "github-agentic-workflow"
	if workflowData != nil {
		// Check if user_agent is configured in engine config first
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.UserAgent != "" {
			userAgent = workflowData.EngineConfig.UserAgent
		} else if workflowData.Name != "" {
			// Fall back to sanitizing workflow name to identifier
			userAgent = SanitizeIdentifier(workflowData.Name)
		}
	}
	yaml.WriteString("          user_agent = \"" + userAgent + "\"\n")

	// Use tools.startup-timeout if specified, otherwise default to DefaultMCPStartupTimeoutSeconds
	startupTimeout := constants.DefaultMCPStartupTimeoutSeconds
	if workflowData.ToolsStartupTimeout > 0 {
		startupTimeout = workflowData.ToolsStartupTimeout
	}
	yaml.WriteString(fmt.Sprintf("          startup_timeout_sec = %d\n", startupTimeout))

	// Use tools.timeout if specified, otherwise default to DefaultToolTimeoutSeconds
	toolTimeout := constants.DefaultToolTimeoutSeconds
	if workflowData.ToolsTimeout > 0 {
		toolTimeout = workflowData.ToolsTimeout
	}
	yaml.WriteString(fmt.Sprintf("          tool_timeout_sec = %d\n", toolTimeout))

	// https://developers.openai.com/codex/mcp
	// Check if remote mode is enabled
	if githubType == "remote" {
		// Remote mode - use hosted GitHub MCP server with streamable HTTP
		// Use readonly endpoint if read-only mode is enabled
		if readOnly {
			yaml.WriteString("          url = \"https://api.githubcopilot.com/mcp-readonly/\"\n")
		} else {
			yaml.WriteString("          url = \"https://api.githubcopilot.com/mcp/\"\n")
		}

		// Use bearer_token_env_var for authentication
		yaml.WriteString("          bearer_token_env_var = \"GH_AW_GITHUB_TOKEN\"\n")
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)

		yaml.WriteString("          command = \"docker\"\n")
		yaml.WriteString("          args = [\n")
		yaml.WriteString("            \"run\",\n")
		yaml.WriteString("            \"-i\",\n")
		yaml.WriteString("            \"--rm\",\n")
		yaml.WriteString("            \"-e\",\n")
		yaml.WriteString("            \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
		if readOnly {
			yaml.WriteString("            \"-e\",\n")
			yaml.WriteString("            \"GITHUB_READ_ONLY=1\",\n")
		}

		// Add GITHUB_TOOLSETS environment variable (always configured, defaults to "default")
		yaml.WriteString("            \"-e\",\n")
		yaml.WriteString("            \"GITHUB_TOOLSETS=" + toolsets + "\",\n")

		yaml.WriteString("            \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"")

		// Append custom args if present
		writeArgsToYAML(yaml, customArgs, "            ")

		yaml.WriteString("\n")
		yaml.WriteString("          ]\n")

		// Use TOML section syntax for environment variables
		yaml.WriteString("          \n")
		yaml.WriteString("          [mcp_servers.github.env]\n")

		// Use effective token with precedence: custom > top-level > default
		effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
		yaml.WriteString("          GITHUB_PERSONAL_ACCESS_TOKEN = \"" + effectiveToken + "\"\n")
	}
}

// renderPlaywrightCodexMCPConfig generates Playwright MCP server configuration for codex config.toml
// Uses the shared helper for TOML format
func (e *CodexEngine) renderPlaywrightCodexMCPConfig(yaml *strings.Builder, playwrightTool any) {
	renderPlaywrightMCPConfigTOML(yaml, playwrightTool)
}

// renderCodexMCPConfig generates custom MCP server configuration for a single tool in codex workflow config.toml
func (e *CodexEngine) renderCodexMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any) error {
	yaml.WriteString("          \n")
	fmt.Fprintf(yaml, "          [mcp_servers.%s]\n", toolName)

	// Use the shared MCP config renderer with TOML format
	renderer := MCPConfigRenderer{
		IndentLevel: "          ",
		Format:      "toml",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		return err
	}

	return nil
}

// renderSafeOutputsCodexMCPConfig generates the Safe Outputs MCP server configuration for codex config.toml
// Uses the shared helper for TOML format
func (e *CodexEngine) renderSafeOutputsCodexMCPConfig(yaml *strings.Builder, workflowData *WorkflowData) {
	// Add safe-outputs MCP server if safe-outputs are configured
	hasSafeOutputs := workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs)
	if hasSafeOutputs {
		renderSafeOutputsMCPConfigTOML(yaml)
	}
}

// renderAgenticWorkflowsCodexMCPConfig generates the Agentic Workflows MCP server configuration for codex config.toml
// Uses the shared helper for TOML format
func (e *CodexEngine) renderAgenticWorkflowsCodexMCPConfig(yaml *strings.Builder) {
	renderAgenticWorkflowsMCPConfigTOML(yaml)
}

// buildGitHubCodexMCPConfig builds GitHub MCP server configuration as a map
func (e *CodexEngine) buildGitHubCodexMCPConfig(githubTool any, workflowData *WorkflowData) map[string]any {
	githubType := getGitHubType(githubTool)
	customGitHubToken := getGitHubToken(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	toolsets := getGitHubToolsets(githubTool)

	config := make(map[string]any)

	// Add user_agent field defaulting to workflow identifier
	userAgent := "github-agentic-workflow"
	if workflowData != nil {
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.UserAgent != "" {
			userAgent = workflowData.EngineConfig.UserAgent
		} else if workflowData.Name != "" {
			userAgent = SanitizeIdentifier(workflowData.Name)
		}
	}
	config["user_agent"] = userAgent

	// Use tools.startup-timeout if specified, otherwise default to DefaultMCPStartupTimeoutSeconds
	startupTimeout := constants.DefaultMCPStartupTimeoutSeconds
	if workflowData.ToolsStartupTimeout > 0 {
		startupTimeout = workflowData.ToolsStartupTimeout
	}
	config["startup_timeout_sec"] = startupTimeout

	// Use tools.timeout if specified, otherwise default to DefaultToolTimeoutSeconds
	toolTimeout := constants.DefaultToolTimeoutSeconds
	if workflowData.ToolsTimeout > 0 {
		toolTimeout = workflowData.ToolsTimeout
	}
	config["tool_timeout_sec"] = toolTimeout

	// Check if remote mode is enabled
	if githubType == "remote" {
		config["type"] = "http"
		if readOnly {
			config["url"] = "https://api.githubcopilot.com/mcp-readonly/"
		} else {
			config["url"] = "https://api.githubcopilot.com/mcp/"
		}
		config["bearer_token_env_var"] = "GH_AW_GITHUB_TOKEN"
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)

		config["command"] = "docker"
		args := []string{"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN"}
		if readOnly {
			args = append(args, "-e", "GITHUB_READ_ONLY=1")
		}
		args = append(args, "-e", "GITHUB_TOOLSETS="+toolsets)
		args = append(args, "ghcr.io/github/github-mcp-server:"+githubDockerImageVersion)
		// Append custom args if present
		if len(customArgs) > 0 {
			args = append(args, customArgs...)
		}
		config["args"] = args

		// Add environment variables
		effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
		config["env"] = map[string]string{
			"GITHUB_PERSONAL_ACCESS_TOKEN": effectiveToken,
		}
	}

	return config
}

// buildPlaywrightCodexMCPConfig builds Playwright MCP server configuration as a map
func (e *CodexEngine) buildPlaywrightCodexMCPConfig(playwrightTool any) map[string]any {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	config := make(map[string]any)
	config["command"] = "npx"

	argsList := []string{"@playwright/mcp@latest", "--output-dir", "/tmp/gh-aw/mcp-logs/playwright"}
	if len(args.AllowedDomains) > 0 {
		argsList = append(argsList, "--allowed-origins", strings.Join(args.AllowedDomains, ";"))
	}
	// Append custom args if present
	if len(customArgs) > 0 {
		argsList = append(argsList, customArgs...)
	}
	config["args"] = argsList

	return config
}

// buildSafeOutputsCodexMCPConfig builds Safe Outputs MCP server configuration as a map
func (e *CodexEngine) buildSafeOutputsCodexMCPConfig(workflowData *WorkflowData) map[string]any {
	// Only build config if safe-outputs are configured
	hasSafeOutputs := workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs)
	if !hasSafeOutputs {
		return nil
	}

	config := make(map[string]any)
	config["command"] = "node"
	config["args"] = []string{"/tmp/gh-aw/safeoutputs/mcp-server.cjs"}
	config["env"] = map[string]string{
		"GH_AW_SAFE_OUTPUTS":        "${{ env.GH_AW_SAFE_OUTPUTS }}",
		"GH_AW_SAFE_OUTPUTS_CONFIG": "${{ toJSON(env.GH_AW_SAFE_OUTPUTS_CONFIG) }}",
		"GH_AW_ASSETS_BRANCH":       "${{ env.GH_AW_ASSETS_BRANCH }}",
		"GH_AW_ASSETS_MAX_SIZE_KB":  "${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}",
		"GH_AW_ASSETS_ALLOWED_EXTS": "${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}",
		"GITHUB_REPOSITORY":         "${{ github.repository }}",
		"GITHUB_SERVER_URL":         "${{ github.server_url }}",
	}

	return config
}

// buildAgenticWorkflowsCodexMCPConfig builds Agentic Workflows MCP server configuration as a map
func (e *CodexEngine) buildAgenticWorkflowsCodexMCPConfig() map[string]any {
	config := make(map[string]any)
	config["command"] = "gh"
	config["args"] = []string{"aw", "mcp-server"}
	config["env"] = map[string]string{
		"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
	}

	return config
}

// buildWebFetchCodexMCPConfig builds Web Fetch MCP server configuration as a map
func (e *CodexEngine) buildWebFetchCodexMCPConfig() map[string]any {
	config := make(map[string]any)
	config["command"] = "npx"
	config["args"] = []string{"-y", "@modelcontextprotocol/server-fetch"}

	return config
}

// buildCustomCodexMCPConfig builds custom MCP server configuration as a map
func (e *CodexEngine) buildCustomCodexMCPConfig(toolName string, toolConfig map[string]any) (map[string]any, error) {
	// Get MCP configuration in the new format
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return nil, err
	}

	config := make(map[string]any)

	// Add fields based on type
	switch mcpConfig.Type {
	case "stdio", "local":
		if mcpConfig.Command != "" {
			config["command"] = mcpConfig.Command
		}
		if len(mcpConfig.Args) > 0 {
			config["args"] = mcpConfig.Args
		}
		if len(mcpConfig.Env) > 0 {
			config["env"] = mcpConfig.Env
		}
	case "http":
		config["type"] = "http"
		if mcpConfig.URL != "" {
			config["url"] = mcpConfig.URL
		}
		if len(mcpConfig.Headers) > 0 {
			config["headers"] = mcpConfig.Headers
		}
	default:
		return nil, fmt.Errorf("unsupported MCP type: %s", mcpConfig.Type)
	}

	return config, nil
}

// GetLogParserScriptId returns the JavaScript script name for parsing Codex logs
func (e *CodexEngine) GetLogParserScriptId() string {
	return "parse_codex_log"
}

// GetErrorPatterns returns regex patterns for extracting error messages from Codex logs
func (e *CodexEngine) GetErrorPatterns() []ErrorPattern {
	patterns := GetCommonErrorPatterns()

	// Add Codex-specific error patterns for Rust log format
	patterns = append(patterns, []ErrorPattern{
		// Rust format patterns (without brackets, with milliseconds and Z timezone)
		{
			ID:           "codex-rust-error",
			Pattern:      `(\d{4}-\d{2}-\d{2}T[\d:.]+Z)\s+(ERROR)\s+(.+)`,
			LevelGroup:   2, // "ERROR" is in the second capture group
			MessageGroup: 3, // error message is in the third capture group
			Description:  "Codex ERROR messages with timestamp",
		},
		{
			ID:           "codex-rust-warning",
			Pattern:      `(\d{4}-\d{2}-\d{2}T[\d:.]+Z)\s+(WARN|WARNING)\s+(.+)`,
			LevelGroup:   2, // "WARN" or "WARNING" is in the second capture group
			MessageGroup: 3, // warning message is in the third capture group
			Description:  "Codex warning messages with timestamp",
		},
	}...)

	return patterns
}
