// Package workflow provides agent file and feature support validation.
//
// # Agent Validation
//
// This file validates agent-specific configuration and feature compatibility
// for agentic workflows. It ensures that:
//   - Custom agent files exist when specified
//   - Engine features are supported (HTTP transport, max-turns, web-search)
//   - Workflow triggers have appropriate security constraints
//
// # Validation Functions
//
//   - validateAgentFile() - Validates custom agent file exists
//   - validateHTTPTransportSupport() - Validates HTTP MCP compatibility with engine
//   - validateMaxTurnsSupport() - Validates max-turns feature support
//   - validateWebSearchSupport() - Validates web-search feature support (warning)
//   - validateWorkflowRunBranches() - Validates workflow_run has branch restrictions
//
// # Validation Patterns
//
// This file uses several patterns:
//   - File existence validation: Agent files
//   - Feature compatibility checks: Engine capabilities
//   - Security validation: workflow_run branch restrictions
//   - Warning vs error: Some validations warn instead of fail
//
// # Security Considerations
//
// The validateWorkflowRunBranches function enforces security best practices:
//   - In strict mode: Errors when workflow_run lacks branch restrictions
//   - In normal mode: Warns when workflow_run lacks branch restrictions
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates custom agent file configuration
//   - It checks engine feature compatibility
//   - It validates agent-specific requirements
//   - It enforces security constraints on triggers
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var agentValidationLog = logger.New("workflow:agent_validation")

// validateAgentFile validates that the custom agent file specified in imports exists
func (c *Compiler) validateAgentFile(workflowData *WorkflowData, markdownPath string) error {
	// Check if agent file is specified in imports
	if workflowData.AgentFile == "" {
		return nil // No agent file specified, no validation needed
	}

	agentPath := workflowData.AgentFile
	agentValidationLog.Printf("Validating agent file exists: %s", agentPath)

	var fullAgentPath string

	// Check if agentPath is already absolute
	if filepath.IsAbs(agentPath) {
		// Use the path as-is (for backward compatibility with tests)
		fullAgentPath = agentPath
	} else {
		// Agent file path is relative to repository root (e.g., ".github/agents/file.md")
		// Need to resolve it relative to the markdown file's directory
		markdownDir := filepath.Dir(markdownPath)
		// Navigate up from .github/workflows to repository root
		repoRoot := filepath.Join(markdownDir, "..", "..")
		fullAgentPath = filepath.Join(repoRoot, agentPath)
	}

	// Check if the file exists
	if _, err := os.Stat(fullAgentPath); err != nil {
		if os.IsNotExist(err) {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("🔍 Can't find the agent file '%s'.\n\nPlease check:\n  • The file path is correct\n  • The file exists in your repository\n  • The path is relative to repository root\n\nExample:\n  agent:\n    file: \".github/agents/developer.md\"", agentPath),
			})
			return errors.New(formattedErr)
		}
		// Other error (permissions, etc.)
		formattedErr := console.FormatError(console.CompilerError{
			Position: console.ErrorPosition{
				File:   markdownPath,
				Line:   1,
				Column: 1,
			},
			Type:    "error",
			Message: fmt.Sprintf("failed to access agent file '%s': %v", agentPath, err),
		})
		return errors.New(formattedErr)
	}

	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			fmt.Sprintf("✓ Agent file exists: %s", agentPath)))
	}

	return nil
}

// validateHTTPTransportSupport validates that HTTP MCP servers are only used with engines that support HTTP transport
func (c *Compiler) validateHTTPTransportSupport(tools map[string]any, engine CodingAgentEngine) error {
	if engine.SupportsHTTPTransport() {
		// Engine supports HTTP transport, no validation needed
		return nil
	}

	// Engine doesn't support HTTP transport, check for HTTP MCP servers
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(config); hasMcp && mcpType == "http" {
				return fmt.Errorf("⚠️  The '%s' tool uses HTTP transport, which isn't supported by the '%s' engine.\n\nWhy? The %s engine only supports stdio (local process) communication.\n\nYour options:\n  1. Switch to an engine that supports HTTP (recommended):\n     engine: copilot\n\n  2. Change the tool to use stdio transport:\n     tools:\n       %s:\n         type: stdio\n         command: \"node server.js\"\n\nLearn more: https://githubnext.github.io/gh-aw/reference/engines/", toolName, engine.GetID(), engine.GetID(), toolName)
			}
		}
	}

	return nil
}

// validateMaxTurnsSupport validates that max-turns is only used with engines that support this feature
func (c *Compiler) validateMaxTurnsSupport(frontmatter map[string]any, engine CodingAgentEngine) error {
	// Check if max-turns is specified in the engine config
	engineSetting, engineConfig := c.ExtractEngineConfig(frontmatter)
	_ = engineSetting // Suppress unused variable warning

	hasMaxTurns := engineConfig != nil && engineConfig.MaxTurns != ""

	if !hasMaxTurns {
		// No max-turns specified, no validation needed
		return nil
	}

	// max-turns is specified, check if the engine supports it
	if !engine.SupportsMaxTurns() {
		return fmt.Errorf("⚠️  The max-turns feature isn't supported by the '%s' engine.\n\nWhy? This feature requires specific engine capabilities for conversation management.\n\nYour options:\n  1. Switch to Copilot (recommended):\n     engine:\n       id: copilot\n       max-turns: 5\n\n  2. Remove max-turns from your configuration\n\nLearn more: https://githubnext.github.io/gh-aw/reference/engines/#max-turns", engine.GetID())
	}

	// Engine supports max-turns - additional validation could be added here if needed
	// For now, we rely on JSON schema validation for format checking

	return nil
}

// validateWebSearchSupport validates that web-search tool is only used with engines that support this feature
func (c *Compiler) validateWebSearchSupport(tools map[string]any, engine CodingAgentEngine) {
	// Check if web-search tool is requested
	_, hasWebSearch := tools["web-search"]

	if !hasWebSearch {
		// No web-search specified, no validation needed
		return
	}

	// web-search is specified, check if the engine supports it
	if !engine.SupportsWebSearch() {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("💡 The '%s' engine doesn't support the web-search tool yet.\n\nThe workflow will compile, but web-search won't be available at runtime.\n\nSee alternatives: https://githubnext.github.io/gh-aw/guides/web-search/", engine.GetID())))
		c.IncrementWarningCount()
	}
}

// validateWorkflowRunBranches validates that workflow_run triggers include branch restrictions
// This is a security best practice to avoid running on all branches
func (c *Compiler) validateWorkflowRunBranches(workflowData *WorkflowData, markdownPath string) error {
	if workflowData.On == "" {
		return nil
	}

	agentValidationLog.Print("Validating workflow_run triggers for branch restrictions")

	// Parse the On field as YAML to check for workflow_run
	// The On field is a YAML string that starts with "on:" key
	var parsedData map[string]any
	if err := yaml.Unmarshal([]byte(workflowData.On), &parsedData); err != nil {
		// If we can't parse the YAML, skip this validation
		agentValidationLog.Printf("Could not parse On field as YAML: %v", err)
		return nil
	}

	// Extract the actual "on" section from the parsed data
	onData, hasOn := parsedData["on"]
	if !hasOn {
		// No "on" key found, skip validation
		return nil
	}

	onMap, isMap := onData.(map[string]any)
	if !isMap {
		// "on" is not a map, skip validation
		return nil
	}

	// Check if workflow_run is present
	workflowRunVal, hasWorkflowRun := onMap["workflow_run"]
	if !hasWorkflowRun {
		// No workflow_run trigger, no validation needed
		return nil
	}

	// Check if workflow_run has branches field
	workflowRunMap, isMap := workflowRunVal.(map[string]any)
	if !isMap {
		// workflow_run is not a map (unusual), skip validation
		return nil
	}

	_, hasBranches := workflowRunMap["branches"]
	if hasBranches {
		// Has branch restrictions, validation passed
		if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("✓ workflow_run trigger has branch restrictions"))
		}
		return nil
	}

	// workflow_run without branches - this is a warning or error depending on mode
	message := "⚠️  Your workflow_run trigger is missing branch restrictions.\n\n" +
		"Why this matters: Without branch restrictions, the workflow will run for ALL branches,\n" +
		"which can cause:\n" +
		"  • Unexpected behavior on feature branches\n" +
		"  • Wasted CI resources\n" +
		"  • Potential security issues\n\n" +
		"Recommended fix - Add branch restrictions:\n" +
		"on:\n" +
		"  workflow_run:\n" +
		"    workflows: [\"your-workflow\"]\n" +
		"    types: [completed]\n" +
		"    branches:\n" +
		"      - main\n" +
		"      - develop\n\n" +
		"Learn more: https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#workflow_run"

	if c.strictMode {
		// In strict mode, this is an error
		formattedErr := console.FormatError(console.CompilerError{
			Position: console.ErrorPosition{
				File:   markdownPath,
				Line:   1,
				Column: 1,
			},
			Type:    "error",
			Message: message,
		})
		return errors.New(formattedErr)
	}

	// In normal mode, this is a warning
	formattedWarning := console.FormatError(console.CompilerError{
		Position: console.ErrorPosition{
			File:   markdownPath,
			Line:   1,
			Column: 1,
		},
		Type:    "warning",
		Message: message,
	})
	fmt.Fprintln(os.Stderr, formattedWarning)
	c.IncrementWarningCount()

	return nil
}
