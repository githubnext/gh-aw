// This file provides agent file and feature support validation.
//
// # Agent Validation
//
// This file validates agent-specific configuration and feature compatibility
// for agentic workflows. It ensures that:
//   - Custom agent files exist when specified
//   - Engine features are supported (HTTP transport, max-turns, web-search, bare mode)
//   - Workflow triggers have appropriate security constraints
//
// # Validation Functions
//
//   - validateAgentFile() - Validates custom agent file exists
//   - validateMaxTurnsSupport() - Validates max-turns feature support
//   - validateMaxContinuationsSupport() - Validates max-continuations feature support
//   - validateWebSearchSupport() - Validates web-search feature support (warning)
//   - validateBareModeSupport() - Validates bare mode feature support (warning)
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
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/goccy/go-yaml"
)

var agentValidationLog = newValidationLogger("agent")

// validateAgentFile validates that the custom agent file specified in imports exists
func (c *Compiler) validateAgentFile(workflowData *WorkflowData, markdownPath string) error {
	// Check if agent file is specified in imports
	if workflowData.AgentFile == "" {
		return nil // No agent file specified, no validation needed
	}

	agentPath := workflowData.AgentFile
	agentValidationLog.Printf("Validating agent file exists: %s", agentPath)

	// Validate path characters to prevent shell injection via crafted filenames.
	// Only alphanumeric characters, dots, underscores, hyphens, forward slashes,
	// and spaces are permitted. Shell metacharacters are rejected.
	if !agentFilePathRegex.MatchString(agentPath) {
		return formatCompilerError(markdownPath, "error",
			fmt.Sprintf("agent file path '%s' contains invalid characters. Only alphanumeric characters, dots, underscores, hyphens, forward slashes, and spaces are allowed.", agentPath), nil)
	}

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
			return formatCompilerError(markdownPath, "error",
				fmt.Sprintf("agent file '%s' does not exist. Ensure the file exists in the repository and is properly imported.", agentPath), nil)
		}
		// Other error (permissions, etc.)
		return formatCompilerError(markdownPath, "error",
			fmt.Sprintf("failed to access agent file '%s': %v", agentPath, err), err)
	}

	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			"✓ Agent file exists: "+agentPath))
	}

	return nil
}

// validateMaxTurnsSupport validates that max-turns is only used with engines that support this feature
func (c *Compiler) validateMaxTurnsSupport(frontmatter map[string]any, engine CodingAgentEngine) error {
	// Check if max-turns is specified in the engine config
	_, engineConfig := c.ExtractEngineConfig(frontmatter)

	hasMaxTurns := engineConfig != nil && engineConfig.MaxTurns != ""

	if !hasMaxTurns {
		// No max-turns specified, no validation needed
		return nil
	}

	agentValidationLog.Printf("Validating max-turns support: engine=%s, maxTurns=%s", engine.GetID(), engineConfig.MaxTurns)

	// max-turns is specified, check if the engine supports it
	if !engine.GetCapabilities().MaxTurns {
		agentValidationLog.Printf("Engine %s does not support max-turns feature", engine.GetID())
		return fmt.Errorf("max-turns not supported: engine '%s' does not support the max-turns feature", engine.GetID())
	}

	// Engine supports max-turns - additional validation could be added here if needed
	// For now, we rely on JSON schema validation for format checking

	return nil
}

// validateMaxContinuationsSupport validates that max-continuations is only used with engines that support this feature
func (c *Compiler) validateMaxContinuationsSupport(frontmatter map[string]any, engine CodingAgentEngine) error {
	// Check if max-continuations is specified in the engine config
	_, engineConfig := c.ExtractEngineConfig(frontmatter)

	if engineConfig == nil || engineConfig.MaxContinuations == 0 {
		// No max-continuations specified, no validation needed
		return nil
	}

	agentValidationLog.Printf("Validating max-continuations support: engine=%s, maxContinuations=%d", engine.GetID(), engineConfig.MaxContinuations)

	// max-continuations is specified, check if the engine supports it
	if !engine.GetCapabilities().MaxContinuations {
		agentValidationLog.Printf("Engine %s does not support max-continuations feature", engine.GetID())
		return fmt.Errorf("max-continuations not supported: engine '%s' does not support the max-continuations feature", engine.GetID())
	}

	return nil
}

// validateUniversalLLMConsumerModel validates that universal consumer engines
// (OpenCode/Crush) declare a provider-qualified engine.model.
func (c *Compiler) validateUniversalLLMConsumerModel(frontmatter map[string]any, engine CodingAgentEngine) error {
	if engine.GetID() != "opencode" && engine.GetID() != "crush" {
		return nil
	}

	_, engineConfig := c.ExtractEngineConfig(frontmatter)
	if engineConfig == nil || strings.TrimSpace(engineConfig.Model) == "" {
		return fmt.Errorf("engine.model is required for engine '%s' and must use provider/model format (for example: copilot/gpt-5, anthropic/claude-sonnet-4, openai/gpt-4.1)", engine.GetID())
	}

	if _, err := resolveUniversalLLMBackendFromModel(engineConfig.Model); err != nil {
		return fmt.Errorf("invalid engine.model for engine '%s': %w", engine.GetID(), err)
	}

	return nil
}

// validatePiEngineRequirements validates Pi's required tool configuration.
func (c *Compiler) validatePiEngineRequirements(tools *ToolsConfig, engine CodingAgentEngine) error {
	if engine.GetID() != "pi" {
		return nil
	}

	if tools == nil || tools.GitHub == nil || tools.GitHub.Mode != "gh-proxy" {
		return errors.New("engine 'pi' requires tools.github.mode: gh-proxy")
	}

	if !tools.CLIProxy {
		return errors.New("engine 'pi' requires tools.cli-proxy: true")
	}

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

	agentValidationLog.Printf("Validating web-search support for engine: %s", engine.GetID())

	// web-search is specified, check if the engine supports it
	if !engine.GetCapabilities().WebSearch {
		agentValidationLog.Printf("Engine %s does not natively support web-search tool, emitting warning", engine.GetID())
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Engine '%s' does not support the web-search tool. See https://github.github.com/gh-aw/guides/web-search/ for alternatives.", engine.GetID())))
		c.IncrementWarningCount()
	}
}

// validateBareModeSupport validates that bare mode is only used with engines that support this feature.
// Emits a warning and has no effect on engines that do not support bare mode.
func (c *Compiler) validateBareModeSupport(frontmatter map[string]any, engine CodingAgentEngine) {
	_, engineConfig := c.ExtractEngineConfig(frontmatter)

	if engineConfig == nil || !engineConfig.Bare {
		// bare mode not requested, no validation needed
		return
	}

	agentValidationLog.Printf("Validating bare mode support for engine: %s", engine.GetID())

	if !engine.GetCapabilities().BareMode {
		agentValidationLog.Printf("Engine %s does not support bare mode, emitting warning", engine.GetID())
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Engine '%s' does not support bare mode (engine.bare: true). Bare mode is only supported for the 'copilot' and 'claude' engines. The setting will be ignored.", engine.GetID())))
		c.IncrementWarningCount()
	}
}

// validateWorkflowRunBranches validates workflow_run trigger requirements.
// It enforces required workflows and branch restrictions guidance.
func (c *Compiler) validateWorkflowRunBranches(workflowData *WorkflowData, markdownPath string) error {
	if !strings.Contains(workflowData.On, "workflow_run") {
		return nil
	}
	agentValidationLog.Print("Validating workflow_run trigger requirements")
	workflowRunMap, ok := parseWorkflowRunTrigger(workflowData.On)
	if !ok {
		return nil
	}
	if err := validateWorkflowRunHasWorkflows(workflowRunMap, markdownPath); err != nil {
		return err
	}
	if _, hasBranches := workflowRunMap["branches"]; hasBranches {
		if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("✓ workflow_run trigger has branch restrictions"))
		}
		return nil
	}
	return c.emitWorkflowRunMissingBranches(markdownPath)
}

func parseWorkflowRunTrigger(onYAML string) (map[string]any, bool) {
	var parsedData map[string]any
	if err := yaml.Unmarshal([]byte(onYAML), &parsedData); err != nil {
		agentValidationLog.Printf("Could not parse On field as YAML: %v", err)
		return nil, false
	}
	onData, hasOn := parsedData["on"]
	if !hasOn {
		return nil, false
	}
	onMap, isMap := onData.(map[string]any)
	if !isMap {
		return nil, false
	}
	workflowRunVal, hasWorkflowRun := onMap["workflow_run"]
	if !hasWorkflowRun {
		return nil, false
	}
	workflowRunMap, ok := workflowRunVal.(map[string]any)
	return workflowRunMap, ok
}

func validateWorkflowRunHasWorkflows(workflowRunMap map[string]any, markdownPath string) error {
	workflowsVal, hasWorkflows := workflowRunMap["workflows"]
	if hasWorkflows && hasNonEmptyWorkflowRunWorkflows(workflowsVal) {
		return nil
	}
	message := `workflow_run trigger must include a non-empty workflows field.

GitHub Actions requires on.workflow_run.workflows to reference at least one workflow.
Without it, the compiled workflow is invalid and will be rejected.

Suggested fix:
on:
  workflow_run:
    workflows: ["your-workflow"]
    types: [completed]`
	return formatCompilerError(markdownPath, "error", message, nil)
}

func (c *Compiler) emitWorkflowRunMissingBranches(markdownPath string) error {
	message := "workflow_run trigger should include branch restrictions for security and performance.\n\n" +
		"Without branch restrictions, the workflow will run for workflow runs on ALL branches,\n" +
		"which can cause unexpected behavior and security issues.\n\n" +
		"Suggested fix: Add branch restrictions to your workflow_run trigger:\n" +
		"on:\n" +
		"  workflow_run:\n" +
		"    workflows: [\"your-workflow\"]\n" +
		"    types: [completed]\n" +
		"    branches:\n" +
		"      - main\n" +
		"      - develop"

	if c.strictMode {
		return formatCompilerError(markdownPath, "error", message, nil)
	}
	formattedWarning := formatCompilerMessage(markdownPath, "warning", message)
	fmt.Fprintln(os.Stderr, formattedWarning)
	c.IncrementWarningCount()
	return nil
}

// hasNonEmptyWorkflowRunWorkflows returns true when workflow_run.workflows
// includes at least one non-empty workflow name.
//
// Supported types:
//   - string: valid when non-empty after trimming whitespace
//   - []string: valid when any item is non-empty after trimming whitespace
//   - []any: valid when any string item is non-empty after trimming whitespace
//
// For all other types, it returns false.
func hasNonEmptyWorkflowRunWorkflows(v any) bool {
	switch workflows := v.(type) {
	case string:
		return strings.TrimSpace(workflows) != ""
	case []string:
		for _, workflow := range workflows {
			if strings.TrimSpace(workflow) != "" {
				return true
			}
		}
		return false
	case []any:
		for _, workflow := range workflows {
			s, ok := workflow.(string)
			if ok && strings.TrimSpace(s) != "" {
				return true
			}
		}
		return false
	default:
		return false
	}
}
