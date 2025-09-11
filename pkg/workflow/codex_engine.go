package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
			supportsToolsWhitelist: true,
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

	// Use model from engineConfig if available, otherwise default to o4-mini
	model := "o4-mini"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		model = workflowData.EngineConfig.Model
	}

	command := fmt.Sprintf(`set -o pipefail
INSTRUCTION=$(cat /tmp/aw-prompts/prompt.txt)
export CODEX_HOME=/tmp/mcp-config

# Create log directory outside git repo
mkdir -p /tmp/aw-logs

# Run codex with log capture - pipefail ensures codex exit code is preserved
codex exec \
  -c model=%s \
  --full-auto "$INSTRUCTION" 2>&1 | tee %s`, model, logFile)

	env := map[string]string{
		"OPENAI_API_KEY":      "${{ secrets.OPENAI_API_KEY }}",
		"GITHUB_STEP_SUMMARY": "${{ env.GITHUB_STEP_SUMMARY }}",
		"GITHUB_AW_PROMPT":    "/tmp/aw-prompts/prompt.txt",
	}

	// Add GITHUB_AW_SAFE_OUTPUTS if output is needed
	hasOutput := workflowData.SafeOutputs != nil
	if hasOutput {
		env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"
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

func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, networkPermissions *NetworkPermissions) {
	yaml.WriteString("          cat > /tmp/mcp-config/config.toml << EOF\n")

	// Add history configuration to disable persistence
	yaml.WriteString("          [history]\n")
	yaml.WriteString("          persistence = \"none\"\n")

	// Generate [mcp_servers] section
	for _, toolName := range mcpTools {
		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubCodexMCPConfig(yaml, githubTool)
		case "playwright":
			playwrightTool := tools["playwright"]
			e.renderPlaywrightCodexMCPConfig(yaml, playwrightTool, networkPermissions)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderCodexMCPConfig(yaml, toolName, toolConfig); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
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
			}
		} else if strings.Contains(line, "] tool") || strings.Contains(line, "] exec") || strings.Contains(line, "] codex") {
			inThinkingSection = false
		}

		// Extract tool calls from Codex logs
		e.parseCodexToolCalls(line, toolCallMap)

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

// parseCodexToolCalls extracts tool call information from Codex log lines
func (e *CodexEngine) parseCodexToolCalls(line string, toolCallMap map[string]*ToolCallInfo) {
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
				}
			}
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
				}
			}
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
func (e *CodexEngine) renderGitHubCodexMCPConfig(yaml *strings.Builder, githubTool any) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.github]\n")

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
// Always uses Docker-based containerized setup in GitHub Actions
func (e *CodexEngine) renderPlaywrightCodexMCPConfig(yaml *strings.Builder, playwrightTool any, networkPermissions *NetworkPermissions) {
	playwrightDockerImageVersion := getPlaywrightDockerImageVersion(playwrightTool)
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")

	// Always use Docker-based Playwright MCP server for consistent containerized execution
	yaml.WriteString("          command = \"docker\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"run\",\n")
	yaml.WriteString("            \"-i\",\n")
	yaml.WriteString("            \"--rm\",\n")
	yaml.WriteString("            \"--shm-size=2gb\",\n")
	yaml.WriteString("            \"--cap-add=SYS_ADMIN\",\n")

	// Generate domain restriction arguments from Playwright tool configuration with bundle resolution
	// Always add domain arguments (defaults to localhost if not configured)
	domainArgs := generatePlaywrightDomainArgs(playwrightTool, networkPermissions)
	for _, arg := range domainArgs {
		yaml.WriteString("            \"" + arg + "\",\n")
	}

	yaml.WriteString("            \"mcr.microsoft.com/playwright:" + playwrightDockerImageVersion + "\"\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          env = {}\n")
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

// GetLogParserScript returns the JavaScript script name for parsing Codex logs
func (e *CodexEngine) GetLogParserScript() string {
	return "parse_codex_log"
}
