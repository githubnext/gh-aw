package workflow

import (
	"fmt"
	"strings"
)

// GitHubModelsEngine represents the GitHub Models agentic engine
// Uses actions/ai-inference to execute prompts with GitHub Models
type GitHubModelsEngine struct {
	BaseEngine
}

// NewGitHubModelsEngine creates a new GitHubModelsEngine instance
func NewGitHubModelsEngine() *GitHubModelsEngine {
	return &GitHubModelsEngine{
		BaseEngine: BaseEngine{
			id:                     "github-models",
			displayName:            "GitHub Models",
			description:            "Uses GitHub Models via actions/ai-inference action",
			experimental:           true,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true,  // Supports HTTP transport via MCP
			supportsMaxTurns:       false, // GitHub Models does not support max-turns yet
		},
	}
}

// GetInstallationSteps returns empty installation steps since GitHub Models doesn't need installation
// The action is pulled directly from GitHub Marketplace
func (e *GitHubModelsEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{}
}

// GetDeclaredOutputFiles returns the output files that GitHub Models may produce
func (e *GitHubModelsEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetVersionCommand returns empty string since GitHub Models is an action, not a CLI tool
func (e *GitHubModelsEngine) GetVersionCommand() string {
	return ""
}

// GetExecutionSteps returns the GitHub Actions steps for executing GitHub Models
func (e *GitHubModelsEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
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

	// Determine the action version
	version := "v1"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	// Build the ai-inference step
	inferenceStepLines := []string{
		"      - name: Run AI Inference",
		"        id: inference",
		fmt.Sprintf("        uses: actions/ai-inference@%s", version),
		"        with:",
		"          prompt-file: '/tmp/aw-prompts/prompt.txt'",
	}

	// Add model if specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		inferenceStepLines = append(inferenceStepLines, fmt.Sprintf("          model: '%s'", workflowData.EngineConfig.Model))
	}

	// Token is always required for actions/ai-inference
	inferenceStepLines = append(inferenceStepLines, "          token: ${{ secrets.GITHUB_TOKEN }}")

	// Add GitHub MCP support if github tool is enabled
	hasGitHubTool := false
	if workflowData.Tools != nil {
		if _, ok := workflowData.Tools["github"]; ok {
			hasGitHubTool = true
		}
	}

	if hasGitHubTool {
		inferenceStepLines = append(inferenceStepLines,
			"          enable-github-mcp: true",
			"          github-mcp-token: ${{ secrets.GITHUB_MCP_TOKEN }}",
		)
	}

	steps = append(steps, GitHubActionStep(inferenceStepLines))

	// Add step to read and log the response
	readResponseStepLines := []string{
		"      - name: Process AI Response",
		"        if: always()",
		"        run: |",
		"          # Capture the response to log file",
		"          if [ -f \"${{ steps.inference.outputs.response-file }}\" ]; then",
		"            cat \"${{ steps.inference.outputs.response-file }}\" | tee -a " + logFile,
		"          else",
		"            echo \"No response file found\" | tee -a " + logFile,
		"          fi",
		"          # Ensure log file exists",
		"          touch " + logFile,
	}
	steps = append(steps, GitHubActionStep(readResponseStepLines))

	return steps
}

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *GitHubModelsEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// RenderMCPConfig renders MCP configuration for GitHub Models engine
// GitHub Models uses the ai-inference action's built-in MCP support, so this is minimal
func (e *GitHubModelsEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// GitHub Models engine handles MCP via the actions/ai-inference action parameters
	// No separate MCP config file is needed - it's configured via action inputs
	// The action handles GitHub MCP server internally when enable-github-mcp is true
}

// ParseLogMetrics extracts metrics from GitHub Models log content
func (e *GitHubModelsEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics

	// GitHub Models doesn't produce structured metrics like Claude or Codex
	// Parse basic information from the response

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

	if verbose {
		fmt.Println("GitHub Models engine: Basic log parsing complete")
	}

	return metrics
}

// GetLogParserScriptId returns the JavaScript script name for parsing GitHub Models logs
func (e *GitHubModelsEngine) GetLogParserScriptId() string {
	return "parse_github_models_log"
}

// GetErrorPatterns returns error patterns for GitHub Models logs
func (e *GitHubModelsEngine) GetErrorPatterns() []ErrorPattern {
	return []ErrorPattern{
		{
			Pattern:      `(?i)error:\s*(.+)`,
			MessageGroup: 1,
			Description:  "Generic error messages",
		},
		{
			Pattern:      `(?i)failed:\s*(.+)`,
			MessageGroup: 1,
			Description:  "Failed operation messages",
		},
	}
}
