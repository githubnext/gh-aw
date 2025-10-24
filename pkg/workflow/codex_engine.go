package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// convertToIdentifier converts a workflow name to a valid identifier format
// by converting to lowercase and replacing spaces with hyphens
func convertToIdentifier(name string) string {
	// Convert to lowercase
	identifier := strings.ToLower(name)
	// Replace spaces and other common separators with hyphens
	identifier = strings.ReplaceAll(identifier, " ", "-")
	identifier = strings.ReplaceAll(identifier, "_", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	identifier = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(identifier, "")
	// Remove any double hyphens that might have been created
	identifier = regexp.MustCompile(`-+`).ReplaceAllString(identifier, "-")
	// Remove leading/trailing hyphens
	identifier = strings.Trim(identifier, "-")

	// If the result is empty, return a default identifier
	if identifier == "" {
		identifier = "github-agentic-workflow"
	}

	return identifier
}

// CodexEngine represents the Codex agentic engine (experimental)
type CodexEngine struct {
	BaseEngine
}

func NewCodexEngine() *CodexEngine {
	return &CodexEngine{
		BaseEngine: BaseEngine{
			id:                      "codex",
			displayName:             "Codex",
			description:             "Uses OpenAI Codex CLI with MCP server support",
			experimental:            true,
			supportsToolsAllowlist:  true,
			supportsHTTPTransport:   true,   // Codex now supports HTTP transport for remote MCP servers
			supportsMaxTurns:        false,  // Codex does not support max-turns feature
			supportsWebFetch:        false,  // Codex does not have built-in web-fetch support
			supportsWebSearch:       true,   // Codex has built-in web-search support
			supportsMCPConfigCLIArg: false,  // Codex uses TOML config file, not CLI argument
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

// GetVersionCommand returns the command to get Codex's version
func (e *CodexEngine) GetVersionCommand() string {
	return "codex --version"
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
	if _, hasWebSearch := workflowData.Tools["web-search"]; hasWebSearch {
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

	command := fmt.Sprintf(`set -o pipefail
INSTRUCTION=$(cat $GH_AW_PROMPT)
mkdir -p $CODEX_HOME/logs
codex %sexec%s%s%s"$INSTRUCTION" 2>&1 | tee %s`, modelParam, webSearchParam, fullAutoParam, customArgsParam, logFile)

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

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *CodexEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
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
	yaml.WriteString("          cat > /tmp/gh-aw/mcp-config/config.toml << EOF\n")

	// Add history configuration to disable persistence
	yaml.WriteString("          [history]\n")
	yaml.WriteString("          persistence = \"none\"\n")

	// Expand neutral tools (like playwright: null) to include the copilot agent tools
	expandedTools := e.expandNeutralToolsToCodexTools(tools)

	// Generate [mcp_servers] section
	for _, toolName := range mcpTools {
		switch toolName {
		case "github":
			githubTool := expandedTools["github"]
			e.renderGitHubCodexMCPConfig(yaml, githubTool, workflowData)
		case "playwright":
			playwrightTool := expandedTools["playwright"]
			e.renderPlaywrightCodexMCPConfig(yaml, playwrightTool)
		case "agentic-workflows":
			e.renderAgenticWorkflowsCodexMCPConfig(yaml)
		case "safe-outputs":
			e.renderSafeOutputsCodexMCPConfig(yaml, workflowData)
		case "web-fetch":
			renderMCPFetchServerConfig(yaml, "toml", "          ", false, false)
		default:
			// Handle custom MCP tools using shared helper (with adapter for isLast parameter)
			HandleCustomMCPToolInSwitch(yaml, toolName, expandedTools, false, func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
				return e.renderCodexMCPConfig(yaml, toolName, toolConfig)
			})
		}
	}

	// Append custom config if provided
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Config != "" {
		yaml.WriteString("          \n")
		yaml.WriteString("          # Custom configuration\n")
		// Write the custom config line by line with proper indentation
		configLines := strings.Split(workflowData.EngineConfig.Config, "\n")
		for _, line := range configLines {
			if strings.TrimSpace(line) != "" {
				yaml.WriteString("          " + line + "\n")
			} else {
				yaml.WriteString("          \n")
			}
		}
	}

	yaml.WriteString("          EOF\n")
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
		if match := regexp.MustCompile(`\] tool ([^(]+)\(`).FindStringSubmatch(line); len(match) > 1 {
			toolName = strings.TrimSpace(match[1])
		}
	}

	// Try new Rust format: "tool provider.method(...)"
	if toolName == "" && strings.HasPrefix(trimmedLine, "tool ") && strings.Contains(trimmedLine, "(") {
		if match := regexp.MustCompile(`^tool ([^(]+)\(`).FindStringSubmatch(trimmedLine); len(match) > 1 {
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
		if match := regexp.MustCompile(`\] exec (.+?) in`).FindStringSubmatch(line); len(match) > 1 {
			execCommand = strings.TrimSpace(match[1])
		}
	}

	// Try new Rust format: "exec command in"
	if execCommand == "" && strings.HasPrefix(trimmedLine, "exec ") {
		if match := regexp.MustCompile(`^exec (.+?) in`).FindStringSubmatch(trimmedLine); len(match) > 1 {
			execCommand = strings.TrimSpace(match[1])
		}
	}

	if execCommand != "" {
		// Create unique bash entry with command info, avoiding colons
		uniqueBashName := fmt.Sprintf("bash_%s", e.shortenCommand(execCommand))

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
		if match := regexp.MustCompile(`in\s+(\d+(?:\.\d+)?)\s*s`).FindStringSubmatch(line); len(match) > 1 {
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

// shortenCommand creates a short identifier for bash commands
func (e *CodexEngine) shortenCommand(command string) string {
	// Take first 20 characters and remove newlines
	shortened := strings.ReplaceAll(command, "\n", " ")
	if len(shortened) > 20 {
		shortened = shortened[:20] + "..."
	}
	return shortened
}

// extractCodexTokenUsage extracts token usage from Codex-specific log lines
func (e *CodexEngine) extractCodexTokenUsage(line string) int {
	// Codex format: "tokens used: 13934"
	codexPattern := `tokens\s+used[:\s]+(\d+)`
	if match := ExtractFirstMatch(line, codexPattern); match != "" {
		if count, err := strconv.Atoi(match); err == nil {
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
			// Fall back to converting workflow name to identifier
			userAgent = convertToIdentifier(workflowData.Name)
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
