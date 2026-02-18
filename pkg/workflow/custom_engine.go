// This file implements the Custom agentic engine.
//
// The Custom engine allows users to bring their own AI provider by defining
// custom installation and execution steps through the engine configuration.
// This engine provides maximum flexibility for integrating third-party AI tools
// or custom implementations that aren't natively supported.
//
// Configuration Example:
//
//	engine:
//	  id: custom
//	  command: /path/to/custom/agent  # Optional: Skip installation if command is provided
//	  steps:                          # Custom installation/setup steps
//	    - name: Install Custom Agent
//	      run: |
//	        curl -sSL https://example.com/install.sh | bash
//	  env:                            # Custom environment variables
//	    CUSTOM_API_KEY: ${{ secrets.CUSTOM_API_KEY }}
//	  args:                           # Custom command-line arguments
//	    - --model
//	    - gpt-4

package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var customLog = logger.New("workflow:custom_engine")

// CustomEngine represents a custom agentic engine that allows users to
// bring their own AI provider with full control over installation and execution.
type CustomEngine struct {
	BaseEngine
}

func NewCustomEngine() *CustomEngine {
	customLog.Print("Creating new Custom engine instance")
	return &CustomEngine{
		BaseEngine: BaseEngine{
			id:                     "custom",
			displayName:            "Custom Engine",
			description:            "Bring your own AI provider with custom installation and execution steps",
			experimental:           false,
			supportsToolsAllowlist: false, // Custom engines don't have built-in tool support
			supportsMaxTurns:       false, // Custom engines must implement max-turns themselves
			supportsWebFetch:       false, // Custom engines must implement web-fetch themselves
			supportsWebSearch:      false, // Custom engines must implement web-search themselves
			supportsFirewall:       false, // Custom engines can implement firewall support if needed
			supportsPlugins:        false, // Custom engines must implement plugin support themselves
			supportsLLMGateway:     false, // Custom engines don't have LLM gateway support by default
		},
	}
}

// SupportsLLMGateway returns -1 indicating no LLM gateway support
func (e *CustomEngine) SupportsLLMGateway() int {
	return -1
}

// GetRequiredSecretNames returns an empty list since custom engines define their own secrets
// Users should declare required secrets through the engine.env configuration
func (e *CustomEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	customLog.Print("Custom engine does not require predefined secrets")
	return []string{}
}

// GetInstallationSteps returns custom installation steps defined in engine configuration
// If engine.command is specified, installation steps are skipped
func (e *CustomEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	customLog.Print("Generating installation steps for custom engine")

	// If command is specified, skip installation (user provides their own executable)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		customLog.Printf("Skipping installation - custom command specified: %s", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Return custom steps from engine configuration using the shared helper
	steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)

	customLog.Printf("Generated %d installation steps from engine configuration", len(steps))
	return steps
}

// GetExecutionSteps returns execution steps for the custom engine
// This is typically defined by users through the engine configuration
func (e *CustomEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	customLog.Print("Generating execution steps for custom engine")

	steps := []GitHubActionStep{}

	// Custom engines must provide their own execution logic through:
	// 1. engine.command - path to the executable
	// 2. engine.args - command-line arguments
	// 3. Custom environment variables via engine.env
	//
	// Users are responsible for:
	// - Installing their AI provider (via engine.steps)
	// - Configuring authentication (via engine.env with secrets)
	// - Executing the agent with the prompt
	// - Handling tool integration if needed

	if workflowData.EngineConfig != nil {
		if workflowData.EngineConfig.Command != "" {
			customLog.Printf("Custom command specified: %s", workflowData.EngineConfig.Command)
		}
		if len(workflowData.EngineConfig.Args) > 0 {
			customLog.Printf("Custom args: %v", workflowData.EngineConfig.Args)
		}
		if len(workflowData.EngineConfig.Env) > 0 {
			customLog.Printf("Custom env vars: %d", len(workflowData.EngineConfig.Env))
		}
	}

	// Custom engines should document their execution pattern in comments
	steps = append(steps, GitHubActionStep{
		"      # Custom Engine Execution",
		"      # This engine expects users to define their own execution steps through:",
		"      #   - engine.command: Path to the AI agent executable",
		"      #   - engine.args: Command-line arguments to pass to the agent",
		"      #   - engine.env: Environment variables for authentication and configuration",
		"      #",
		"      # Example configuration:",
		"      # engine:",
		"      #   id: custom",
		"      #   command: /usr/local/bin/my-agent",
		"      #   args: [\"--model\", \"gpt-4\", \"--prompt\", \"$PROMPT\"]",
		"      #   env:",
		"      #     API_KEY: ${{ secrets.MY_API_KEY }}",
	})

	customLog.Printf("Generated %d execution steps (placeholder)", len(steps))
	return steps
}

// GetDeclaredOutputFiles returns empty list - custom engines define their own outputs
func (e *CustomEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// RenderMCPConfig provides no MCP configuration for custom engines
// Custom engines must implement MCP support themselves if needed
func (e *CustomEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	customLog.Print("Custom engine does not provide built-in MCP configuration")
	// No-op - custom engines don't provide MCP configuration by default
}

// convertStepToYAML converts a step map to YAML format for GitHub Actions
func (e *CustomEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}
