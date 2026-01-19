package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var ralphLog = logger.New("workflow:ralph_engine")

// RalphEngine represents the Ralph autonomous agent loop engine
// Ralph repeatedly runs an AI agent (Copilot) until all PRD items are complete
// Each iteration is a fresh agent instance with clean context
// Memory persists via git history, progress.txt, and prd.json
type RalphEngine struct {
	BaseEngine
	copilotEngine *CopilotEngine // Underlying Copilot engine for AI execution
}

// NewRalphEngine creates a new RalphEngine instance
func NewRalphEngine() *RalphEngine {
	return &RalphEngine{
		BaseEngine: BaseEngine{
			id:                     "ralph",
			displayName:            "Ralph Loop + Copilot",
			description:            "Autonomous agent loop that runs Copilot repeatedly until all PRD items are complete",
			experimental:           true,
			supportsToolsAllowlist: true,  // Inherit from Copilot
			supportsHTTPTransport:  true,  // Inherit from Copilot
			supportsMaxTurns:       false, // Ralph manages iterations, not max-turns
			supportsWebFetch:       true,  // Inherit from Copilot
			supportsWebSearch:      false, // Inherit from Copilot
			supportsFirewall:       true,  // Inherit from Copilot
		},
		copilotEngine: NewCopilotEngine(),
	}
}

// GetRequiredSecretNames returns the list of secrets required by the Ralph engine
// Delegates to Copilot engine since Ralph uses Copilot for execution
func (e *RalphEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	ralphLog.Printf("Getting required secret names for Ralph engine: workflow=%s", workflowData.Name)
	return e.copilotEngine.GetRequiredSecretNames(workflowData)
}

// GetInstallationSteps returns the GitHub Actions steps needed to install Ralph engine
// This includes installing Copilot CLI and Ralph loop scripts
func (e *RalphEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	ralphLog.Printf("Generating installation steps for Ralph engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		ralphLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	var steps []GitHubActionStep

	// Install Copilot CLI first (delegate to Copilot engine)
	copilotSteps := e.copilotEngine.GetInstallationSteps(workflowData)
	steps = append(steps, copilotSteps...)

	// Add Ralph loop script installation
	ralphInstallStep := []string{
		"      - name: Install Ralph loop scripts",
		"        run: |",
		"          # Create ralph scripts directory",
		"          mkdir -p /tmp/gh-aw/ralph",
		"          ",
		"          # Copy Ralph loop scripts from actions/setup/sh",
		"          cp /opt/gh-aw/actions/ralph-loop.sh /tmp/gh-aw/ralph/",
		"          cp /opt/gh-aw/actions/ralph-prompt.md /tmp/gh-aw/ralph/",
		"          chmod +x /tmp/gh-aw/ralph/ralph-loop.sh",
		"          ",
		"          # Initialize prd.json from workspace if it exists, otherwise create empty",
		"          if [ -f prd.json ]; then",
		"            cp prd.json /tmp/gh-aw/ralph/prd.json",
		"          else",
		"            echo '{\"userStories\":[]}' > /tmp/gh-aw/ralph/prd.json",
		"          fi",
		"          ",
		"          # Initialize progress.txt if it doesn't exist",
		"          if [ ! -f /tmp/gh-aw/ralph/progress.txt ]; then",
		"            touch /tmp/gh-aw/ralph/progress.txt",
		"          fi",
		"          ",
		"          echo \"Ralph loop scripts installed successfully\"",
	}
	steps = append(steps, GitHubActionStep(ralphInstallStep))

	return steps
}

// GetExecutionSteps returns the GitHub Actions steps for executing Ralph loop
// Ralph will repeatedly call Copilot until all PRD items are complete
func (e *RalphEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	ralphLog.Printf("Building Ralph engine execution steps: workflow=%s", workflowData.Name)

	var steps []GitHubActionStep

	// Determine max iterations (default 10)
	maxIterations := "10"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		maxIterations = workflowData.EngineConfig.MaxTurns
	}

	// Build Ralph loop execution step
	ralphLoopStep := []string{
		"      - name: Execute Ralph loop with Copilot",
		"        id: ralph-loop",
		"        run: |",
		"          # Set up environment",
		fmt.Sprintf("          export GH_AW_PROMPT=\"%s\"", "/tmp/gh-aw/aw-prompts/prompt.txt"),
		fmt.Sprintf("          export GH_AW_MCP_CONFIG=\"%s\"", "/tmp/gh-aw/mcp-config/mcp-servers.json"),
		fmt.Sprintf("          export RALPH_MAX_ITERATIONS=\"%s\"", maxIterations),
		"          ",
		"          # Run Ralph loop",
		"          cd $GITHUB_WORKSPACE",
		fmt.Sprintf("          /tmp/gh-aw/ralph/ralph-loop.sh 2>&1 | tee %s", logFile),
		"        env:",
		"          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}",
	}

	// Add MCP gateway API key if MCP servers are present
	if HasMCPServers(workflowData) {
		ralphLoopStep = append(ralphLoopStep, "          MCP_GATEWAY_API_KEY: ${{ secrets.MCP_GATEWAY_API_KEY }}")
	}

	// Add GitHub token for GitHub MCP server if present
	if hasGitHubTool(workflowData.ParsedTools) {
		ralphLoopStep = append(ralphLoopStep, "          GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GITHUB_MCP_SERVER_TOKEN }}")
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			ralphLoopStep = append(ralphLoopStep, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	steps = append(steps, GitHubActionStep(ralphLoopStep))

	return steps
}

// RenderMCPConfig renders MCP configuration for Ralph engine
// Delegates to Copilot engine since Ralph uses Copilot's MCP configuration
func (e *RalphEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	ralphLog.Printf("Rendering MCP configuration for Ralph engine: workflow=%s", workflowData.Name)
	e.copilotEngine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}

// ParseLogMetrics extracts metrics from Ralph loop logs
func (e *RalphEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	ralphLog.Printf("Parsing Ralph engine log metrics: log_size=%d bytes", len(logContent))

	var metrics LogMetrics
	lines := strings.Split(logContent, "\n")

	// Track Ralph-specific metrics
	iterationCount := 0
	completedStories := 0
	failedStories := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Count Ralph iterations
		if strings.Contains(trimmed, "Starting iteration") {
			iterationCount++
		}

		// Count completed stories
		if strings.Contains(trimmed, "Story completed") || strings.Contains(trimmed, "passes: true") {
			completedStories++
		}

		// Count failed stories
		if strings.Contains(trimmed, "Story failed") || strings.Contains(trimmed, "passes: false") {
			failedStories++
		}
	}

	// Store Ralph-specific metrics
	metrics.Turns = iterationCount
	// ToolCalls is a slice, create entries for completed and failed stories
	if completedStories > 0 || failedStories > 0 {
		metrics.ToolCalls = make([]ToolCallInfo, 0)
		// We can track these as tool calls to user stories
		if completedStories > 0 {
			metrics.ToolCalls = append(metrics.ToolCalls, ToolCallInfo{
				Name:      "completed_stories",
				CallCount: completedStories,
			})
		}
		if failedStories > 0 {
			metrics.ToolCalls = append(metrics.ToolCalls, ToolCallInfo{
				Name:      "failed_stories",
				CallCount: failedStories,
			})
		}
	}

	// Delegate to Copilot engine for detailed token/cost parsing
	copilotMetrics := e.copilotEngine.ParseLogMetrics(logContent, verbose)
	if copilotMetrics.TokenUsage > 0 {
		metrics.TokenUsage = copilotMetrics.TokenUsage
		metrics.EstimatedCost = copilotMetrics.EstimatedCost
	}

	if verbose {
		fmt.Printf("Ralph metrics: iterations=%d, completed=%d, failed=%d\n",
			iterationCount, completedStories, failedStories)
	}

	return metrics
}

// GetLogParserScriptId returns the JavaScript script name for parsing Ralph logs
func (e *RalphEngine) GetLogParserScriptId() string {
	return "parse_ralph_log"
}

// GetDefaultDetectionModel returns the default model for threat detection
// Delegates to Copilot engine
func (e *RalphEngine) GetDefaultDetectionModel() string {
	return string(constants.DefaultCopilotDetectionModel)
}

// GetDeclaredOutputFiles returns output files that Ralph may produce
// Delegates to Copilot engine and adds Ralph-specific files
func (e *RalphEngine) GetDeclaredOutputFiles() []string {
	files := e.copilotEngine.GetDeclaredOutputFiles()
	// Add Ralph-specific output files (in tmp directory to avoid secret redaction issues)
	files = append(files, "/tmp/gh-aw/ralph/progress.txt", "/tmp/gh-aw/ralph/prd.json")
	return files
}
