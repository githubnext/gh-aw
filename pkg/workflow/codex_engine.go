package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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
			id:                     "codex",
			displayName:            "Codex",
			description:            "Uses OpenAI Codex CLI with MCP server support",
			experimental:           true,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  false, // Codex only supports stdio transport
			supportsMaxTurns:       false, // Codex does not support max-turns feature
		},
	}
}

func (e *CodexEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	// Build the npm install command, optionally with version
	installCmd := "npm install -g @openai/codex"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		installCmd = fmt.Sprintf("npm install -g @openai/codex@%s", workflowData.EngineConfig.Version)
	}

	return []GitHubActionStep{
		{
			"      - name: Setup Node.js",
			"        uses: actions/setup-node@v4",
			"        with:",
			"          node-version: '24'",
		},
		{
			"      - name: Install Codex",
			fmt.Sprintf("        run: %s", installCmd),
		},
	}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Codex
func (e *CodexEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := e.convertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
	}

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
	fullAutoParam := "--dangerously-bypass-approvals-and-sandbox "

	command := fmt.Sprintf(`set -o pipefail
INSTRUCTION=$(cat /tmp/aw-prompts/prompt.txt)
export CODEX_HOME=/tmp/mcp-config

# Create log directory outside git repo
mkdir -p /tmp/aw-logs

# where is Codex
which codex

# Check Codex version
codex --version

# Authenticate with Codex
codex login --api-key "$OPENAI_API_KEY"

# Run codex with log capture - pipefail ensures codex exit code is preserved
codex %s%s--full-auto exec %s"$INSTRUCTION" 2>&1 | tee %s`, modelParam, webSearchParam, fullAutoParam, logFile)

	env := map[string]string{
		"OPENAI_API_KEY":      "${{ secrets.OPENAI_API_KEY }}",
		"GITHUB_STEP_SUMMARY": "${{ env.GITHUB_STEP_SUMMARY }}",
		"GITHUB_AW_PROMPT":    "/tmp/aw-prompts/prompt.txt",
	}

	// Add GITHUB_AW_SAFE_OUTPUTS if output is needed
	hasOutput := workflowData.SafeOutputs != nil
	if hasOutput {
		env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"

		// Add staged flag if specified
		if workflowData.SafeOutputs.Staged != nil && *workflowData.SafeOutputs.Staged {
			env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "true"
		}
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
	stepLines = append(stepLines, "        run: |")

	// Split command into lines and indent them properly
	commandLines := strings.Split(command, "\n")
	for _, line := range commandLines {
		stepLines = append(stepLines, "          "+line)
	}

	// Add environment variables
	if len(env) > 0 {
		stepLines = append(stepLines, "        env:")
		// Sort environment keys for consistent output
		envKeys := make([]string, 0, len(env))
		for key := range env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)

		for _, key := range envKeys {
			value := env[key]
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

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
		// If the original playwright tool has additional configuration (like docker_image_version),
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
	yaml.WriteString("          cat > /tmp/mcp-config/config.toml << EOF\n")

	// Add history configuration to disable persistence
	yaml.WriteString("          [history]\n")
	yaml.WriteString("          persistence = \"none\"\n")

	// Add network configuration based on network permissions
	e.renderNetworkConfig(yaml, workflowData.NetworkPermissions)

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
			e.renderPlaywrightCodexMCPConfig(yaml, playwrightTool, workflowData.NetworkPermissions)
		case "safe-outputs":
			e.renderSafeOutputsCodexMCPConfig(yaml, workflowData)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := expandedTools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderCodexMCPConfig(yaml, toolName, toolConfig); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
			}
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
		if strings.Contains(line, "] thinking") {
			if !inThinkingSection {
				turns++
				inThinkingSection = true
				// Start of a new thinking section, save previous sequence if any
				if len(currentSequence) > 0 {
					metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
					currentSequence = []string{}
				}
			}
		} else if strings.Contains(line, "] tool") || strings.Contains(line, "] exec") || strings.Contains(line, "] codex") {
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

		// Count errors and warnings
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			metrics.ErrorCount++
		}
		if strings.Contains(lowerLine, "warning") {
			metrics.WarningCount++
		}
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

	return metrics
}

// parseCodexToolCallsWithSequence extracts tool call information from Codex log lines and returns tool name
func (e *CodexEngine) parseCodexToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	// Parse tool calls: "] tool provider.method(...)"
	if strings.Contains(line, "] tool ") && strings.Contains(line, "(") {
		if match := regexp.MustCompile(`\] tool ([^(]+)\(`).FindStringSubmatch(line); len(match) > 1 {
			toolName := strings.TrimSpace(match[1])
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
	}

	// Parse exec commands: "] exec command" - treat as bash calls
	if strings.Contains(line, "] exec ") {
		if match := regexp.MustCompile(`\] exec (.+?) in`).FindStringSubmatch(line); len(match) > 1 {
			command := strings.TrimSpace(match[1])
			// Create unique bash entry with command info, avoiding colons
			uniqueBashName := fmt.Sprintf("bash_%s", e.shortenCommand(command))

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
// Always uses Docker MCP as the default
func (e *CodexEngine) renderGitHubCodexMCPConfig(yaml *strings.Builder, githubTool any, workflowData *WorkflowData) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
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

	// Always use Docker-based GitHub MCP server (services mode has been removed)
	yaml.WriteString("          command = \"docker\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"run\",\n")
	yaml.WriteString("            \"-i\",\n")
	yaml.WriteString("            \"--rm\",\n")
	yaml.WriteString("            \"-e\",\n")
	yaml.WriteString("            \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	yaml.WriteString("            \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          env = { \"GITHUB_PERSONAL_ACCESS_TOKEN\" = \"${{ secrets.GITHUB_TOKEN }}\" }\n")
}

// renderPlaywrightCodexMCPConfig generates Playwright MCP server configuration for codex config.toml
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
func (e *CodexEngine) renderPlaywrightCodexMCPConfig(yaml *strings.Builder, playwrightTool any, networkPermissions *NetworkPermissions) {
	args := generatePlaywrightDockerArgs(playwrightTool, networkPermissions)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")
	yaml.WriteString("          command = \"npx\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"@playwright/mcp@latest\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("            \"--allowed-origins\",\n")
		yaml.WriteString("            \"" + strings.Join(args.AllowedDomains, ",") + "\"")
	}
	yaml.WriteString("\n")
	yaml.WriteString("          ]\n")
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
func (e *CodexEngine) renderSafeOutputsCodexMCPConfig(yaml *strings.Builder, workflowData *WorkflowData) {
	// Add safe-outputs MCP server if safe-outputs are configured
	hasSafeOutputs := workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs)
	if hasSafeOutputs {
		yaml.WriteString("          \n")
		yaml.WriteString("          [mcp_servers.safe_outputs]\n")
		yaml.WriteString("          command = \"node\"\n")
		yaml.WriteString("          args = [\n")
		yaml.WriteString("            \"/tmp/safe-outputs/mcp-server.cjs\",\n")
		yaml.WriteString("          ]\n")
		yaml.WriteString("          env = { \"GITHUB_AW_SAFE_OUTPUTS\" = \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\", \"GITHUB_AW_SAFE_OUTPUTS_CONFIG\" = ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }} }\n")
	}
}

// GetLogParserScriptId returns the JavaScript script name for parsing Codex logs
func (e *CodexEngine) GetLogParserScriptId() string {
	return "parse_codex_log"
}

// GetErrorPatterns returns regex patterns for extracting error messages from Codex logs
func (e *CodexEngine) GetErrorPatterns() []ErrorPattern {
	return []ErrorPattern{
		{
			Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+stream\s+(error):\s+(.+)`,
			LevelGroup:   2, // "error" is in the second capture group
			MessageGroup: 3, // error message is in the third capture group
			Description:  "Codex stream errors with timestamp",
		},
		{
			Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
			LevelGroup:   2, // "ERROR" is in the second capture group
			MessageGroup: 3, // error message is in the third capture group
			Description:  "Codex ERROR messages with timestamp",
		},
		{
			Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(WARN|WARNING):\s+(.+)`,
			LevelGroup:   2, // "WARN" or "WARNING" is in the second capture group
			MessageGroup: 3, // warning message is in the third capture group
			Description:  "Codex warning messages with timestamp",
		},
	}
}

// ValidateNetworkPermissions validates network permissions for Codex engine
func (e *CodexEngine) ValidateNetworkPermissions(networkPermissions *NetworkPermissions) error {
	// If no network permissions specified, that's fine for Codex
	if networkPermissions == nil {
		return nil
	}

	// Check for "defaults" mode - not supported by Codex
	if networkPermissions.Mode == "defaults" {
		return fmt.Errorf("network: defaults is not supported by Codex engine. Use network: {} for no network access or network: { allowed: [\"*\"] } for full network access")
	}

	// Check for allowed domains
	if len(networkPermissions.Allowed) == 0 {
		// Empty allowed list {} is valid - means no network access
		return nil
	}

	// Check if it's the wildcard for full network access
	if len(networkPermissions.Allowed) == 1 && networkPermissions.Allowed[0] == "*" {
		// This is valid - means full network access
		return nil
	}

	// Any other specific domains or patterns are not supported by Codex
	return fmt.Errorf("specific network domains are not supported by Codex engine. Use network: {} for no network access or network: { allowed: [\"*\"] } for full network access")
}

// GetDefaultNetworkPermissions returns default network permissions for Codex engine
func (e *CodexEngine) GetDefaultNetworkPermissions() *NetworkPermissions {
	// Codex defaults to no network access (secure by default)
	return &NetworkPermissions{
		Allowed: []string{}, // Empty allowed list means no network access
	}
}

// renderNetworkConfig generates network configuration for codex config.toml
func (e *CodexEngine) renderNetworkConfig(yaml *strings.Builder, networkPermissions *NetworkPermissions) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [sandbox]\n")

	// Default network setting if no permissions specified (equivalent to network: defaults for other engines)
	if networkPermissions == nil {
		yaml.WriteString("          # Network access enabled by default\n")
		yaml.WriteString("          network = true\n")
		return
	}

	// Handle empty allowed list - means no network access
	if len(networkPermissions.Allowed) == 0 {
		yaml.WriteString("          # Network access disabled\n")
		yaml.WriteString("          network = false\n")
		return
	}

	// Handle wildcard - means full network access
	if len(networkPermissions.Allowed) == 1 && networkPermissions.Allowed[0] == "*" {
		yaml.WriteString("          # Network access enabled\n")
		yaml.WriteString("          network = true\n")
		return
	}

	// This should not happen due to validation, but handle gracefully
	yaml.WriteString("          # Network access enabled (fallback)\n")
	yaml.WriteString("          network = true\n")
}
