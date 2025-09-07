package workflow

import (
	"fmt"
	"regexp"
	"sort"
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
	metrics := NewLogMetrics()
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")

	for i, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract Codex-specific token usage (always sum for Codex)
		if tokenUsage := e.extractCodexTokenUsage(line); tokenUsage > 0 {
			totalTokenUsage += tokenUsage
		}

		// Extract tool invocation statistics
		e.extractCodexToolInvocations(lines, i, &metrics, verbose)

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

// extractCodexToolInvocations extracts tool invocation statistics from Codex log lines
func (e *CodexEngine) extractCodexToolInvocations(lines []string, currentIndex int, metrics *LogMetrics, verbose bool) {
	if currentIndex >= len(lines) {
		return
	}

	line := lines[currentIndex]

	// Detect tool usage: "[2025-08-31T12:37:11] tool get_current_time(...)"
	toolPattern := `\] tool ([^(]+)\(`
	if toolMatch := regexp.MustCompile(toolPattern).FindStringSubmatch(line); toolMatch != nil {
		toolName := strings.TrimSpace(toolMatch[1])

		// Look ahead to find the result status and duration
		success := false
		duration := time.Millisecond * 100 // Default placeholder
		outputSize := int64(0)

		for j := currentIndex + 1; j < len(lines) && j < currentIndex+10; j++ {
			nextLine := lines[j]

			// Check for success/failure status
			if strings.Contains(nextLine, "success in") || strings.Contains(nextLine, "succeeded in") {
				success = true
				// Try to extract duration from "success in 0.1s" format
				if durationMatch := regexp.MustCompile(`(?:success|succeeded) in ([\d.]+)s`).FindStringSubmatch(nextLine); durationMatch != nil {
					if seconds, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
						duration = time.Duration(seconds * float64(time.Second))
					}
				}
				break
			} else if strings.Contains(nextLine, "failure in") || strings.Contains(nextLine, "failed in") || strings.Contains(nextLine, "error in") {
				success = false
				// Try to extract duration from "failed in 0.1s" format
				if durationMatch := regexp.MustCompile(`(?:failure|failed|error) in ([\d.]+)s`).FindStringSubmatch(nextLine); durationMatch != nil {
					if seconds, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
						duration = time.Duration(seconds * float64(time.Second))
					}
				}
				break
			}

			// Estimate output size from lines that might contain tool output
			if !strings.Contains(nextLine, "[2025-") && strings.TrimSpace(nextLine) != "" {
				outputSize += int64(len(nextLine))
			}
		}

		// Normalize tool name format (e.g., "github.search_issues" -> "github::search_issues")
		if strings.Contains(toolName, ".") {
			parts := strings.Split(toolName, ".")
			if len(parts) >= 2 {
				provider := parts[0]
				method := strings.Join(parts[1:], "_")
				toolName = fmt.Sprintf("%s::%s", provider, method)
			}
		}

		metrics.AddToolInvocation(toolName, outputSize, duration, success)
	}

	// Detect exec commands: "[2025-08-31T12:37:11] exec echo 'Hello World' in working directory"
	execPattern := `\] exec (.+?) in`
	if execMatch := regexp.MustCompile(execPattern).FindStringSubmatch(line); execMatch != nil {
		// Look ahead to find the result status and duration
		success := false
		duration := time.Millisecond * 100 // Default placeholder
		outputSize := int64(0)

		for j := currentIndex + 1; j < len(lines) && j < currentIndex+10; j++ {
			nextLine := lines[j]

			// Check for success/failure status
			if strings.Contains(nextLine, "succeeded in") {
				success = true
				// Try to extract duration
				if durationMatch := regexp.MustCompile(`succeeded in ([\d.]+)s`).FindStringSubmatch(nextLine); durationMatch != nil {
					if seconds, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
						duration = time.Duration(seconds * float64(time.Second))
					}
				}
				break
			} else if strings.Contains(nextLine, "failed in") || strings.Contains(nextLine, "error") {
				success = false
				// Try to extract duration
				if durationMatch := regexp.MustCompile(`failed in ([\d.]+)s`).FindStringSubmatch(nextLine); durationMatch != nil {
					if seconds, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
						duration = time.Duration(seconds * float64(time.Second))
					}
				}
				break
			}

			// Estimate output size from command output
			if !strings.Contains(nextLine, "[2025-") && strings.TrimSpace(nextLine) != "" {
				outputSize += int64(len(nextLine))
			}
		}

		// Use "Bash" as the tool name for exec commands, consistent with Claude
		bashCommand := execMatch[1] // Extract the command from the regex match
		metrics.AddToolInvocationWithCommand("Bash", outputSize, duration, success, bashCommand)
	}
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
