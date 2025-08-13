package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
		},
	}
}

func (e *CodexEngine) GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep {
	// Build the npm install command, optionally with version
	installCmd := "npm install -g @openai/codex"
	if engineConfig != nil && engineConfig.Version != "" {
		installCmd = fmt.Sprintf("npm install -g @openai/codex@%s", engineConfig.Version)
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

func (e *CodexEngine) GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig) ExecutionConfig {
	// Use model from engineConfig if available, otherwise default to o4-mini
	model := "o4-mini"
	if engineConfig != nil && engineConfig.Model != "" {
		model = engineConfig.Model
	}

	command := fmt.Sprintf(`INSTRUCTION=$(cat /tmp/aw-prompts/prompt.txt)
export CODEX_HOME=/tmp/mcp-config

# Create log directory outside git repo
mkdir -p /tmp/aw-logs

# Run codex with log capture
codex exec \
  -c model=%s \
  --full-auto "$INSTRUCTION" 2>&1 | tee %s`, model, logFile)

	return ExecutionConfig{
		StepName: "Run Codex",
		Command:  command,
		Environment: map[string]string{
			"OPENAI_API_KEY":      "${{ secrets.OPENAI_API_KEY }}",
			"GITHUB_STEP_SUMMARY": "${{ env.GITHUB_STEP_SUMMARY }}",
		},
	}
}

func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
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
	var startTime, endTime time.Time
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract timestamps for duration calculation
		timestamp := ExtractTimestamp(line)
		if !timestamp.IsZero() {
			if startTime.IsZero() || timestamp.Before(startTime) {
				startTime = timestamp
			}
			if endTime.IsZero() || timestamp.After(endTime) {
				endTime = timestamp
			}
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

	metrics.TokenUsage = totalTokenUsage

	// Calculate duration
	if !startTime.IsZero() && !endTime.IsZero() {
		metrics.Duration = endTime.Sub(startTime)
	}

	return metrics
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

// GetFilenamePatterns returns patterns that can be used to detect Codex from filenames
func (e *CodexEngine) GetFilenamePatterns() []string {
	return []string{"codex"}
}

// DetectFromContent analyzes log content and returns a confidence score for Codex engine
func (e *CodexEngine) DetectFromContent(logContent string) int {
	confidence := 0
	lines := strings.Split(logContent, "\n")
	maxLinesToCheck := 20
	if len(lines) < maxLinesToCheck {
		maxLinesToCheck = len(lines)
	}

	for i := 0; i < maxLinesToCheck; i++ {
		line := lines[i]

		// Look for Codex-specific patterns - must be exact format
		// Codex format: "tokens used: 13934" (not preceded by other words like "Total")
		if strings.Contains(line, "tokens used:") {
			// Check that it's not preceded by words like "Total", "Input", "Output"
			lowerLine := strings.ToLower(line)
			if !strings.Contains(lowerLine, "total tokens used") &&
				!strings.Contains(lowerLine, "input tokens used") &&
				!strings.Contains(lowerLine, "output tokens used") {
				// Check if it matches the exact Codex pattern
				if match := ExtractFirstMatch(line, `tokens\s+used[:\s]+(\d+)`); match != "" {
					confidence += 20 // Strong indicator
				}
			}
		}
		if strings.Contains(strings.ToLower(line), "codex") {
			confidence += 10
		}
	}

	return confidence
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

// renderCodexMCPConfig generates custom MCP server configuration for a single tool in codex workflow config.toml
func (e *CodexEngine) renderCodexMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any) error {
	yaml.WriteString("          \n")
	yaml.WriteString(fmt.Sprintf("          [mcp_servers.%s]\n", toolName))

	// Use the shared MCP config renderer with TOML format
	renderer := MCPConfigRenderer{
		IndentLevel: "          ",
		Format:      "toml",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, false, renderer)
	if err != nil {
		return err
	}

	return nil
}
