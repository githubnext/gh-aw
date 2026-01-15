package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var promptsLog = logger.New("workflow:prompts")

// prompts.go consolidates all prompt-related functions for agentic workflows.
// This file contains functions that generate workflow steps to append various
// contextual instructions to the agent's prompt file during execution.
//
// Prompts are organized by feature area:
// - Safe outputs: Instructions for using the safeoutputs MCP server
// - Cache memory: Instructions for persistent cache folder access
// - Tool prompts: Instructions for specific tools (edit, playwright)
// - PR context: Instructions for pull request branch context

// ============================================================================
// Safe Outputs Prompts
// ============================================================================

// generateSafeOutputsPromptStep generates a separate step for safe outputs instructions
// This tells agents to use the safeoutputs MCP server instead of gh CLI
func (c *Compiler) generateSafeOutputsPromptStep(yaml *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if !HasSafeOutputsEnabled(safeOutputs) {
		return
	}

	// Get the list of enabled tool names
	enabledTools := GetEnabledSafeOutputToolNames(safeOutputs)
	if len(enabledTools) == 0 {
		return
	}

	promptsLog.Printf("Generating safe outputs prompt step with %d enabled tools", len(enabledTools))

	// Create a comma-separated list of tool names for the prompt
	toolsList := strings.Join(enabledTools, ", ")

	// Create the prompt text with the actual tool names injected
	promptText := fmt.Sprintf(`<safe-outputs>
<description>GitHub API Access Instructions</description>
<important>
The gh CLI is NOT authenticated. Do NOT use gh commands for GitHub operations.
</important>
<instructions>
To create or modify GitHub resources (issues, discussions, pull requests, etc.), you MUST call the appropriate safe output tool. Simply writing content will NOT work - the workflow requires actual tool calls.

**Available tools**: %s

**Critical**: Tool calls write structured data that downstream jobs process. Without tool calls, follow-up actions will be skipped.
</instructions>
</safe-outputs>`, toolsList)

	generateStaticPromptStep(yaml,
		"Append safe outputs instructions to prompt",
		promptText,
		true)
}

// ============================================================================
// Cache Memory Prompts
// ============================================================================

// generateCacheMemoryPromptStep generates a separate step for cache memory instructions
// when cache-memory is enabled, informing the agent about persistent storage capabilities
func (c *Compiler) generateCacheMemoryPromptStep(yaml *strings.Builder, config *CacheMemoryConfig) {
	if config == nil || len(config.Caches) == 0 {
		return
	}

	promptsLog.Printf("Generating cache memory prompt step with %d caches", len(config.Caches))

	appendPromptStepWithHeredoc(yaml,
		"Append cache-memory instructions to prompt",
		func(y *strings.Builder) {
			generateCacheMemoryPromptSection(y, config)
		})
}

// ============================================================================
// Tool Prompts - Playwright
// ============================================================================

// hasPlaywrightTool checks if the playwright tool is enabled in the tools configuration
func hasPlaywrightTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.Playwright != nil
}

// generatePlaywrightPromptStep generates a separate step for playwright output directory instructions
// Only generates the step if playwright tool is enabled in the workflow
func (c *Compiler) generatePlaywrightPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStepFromFile(yaml,
		"Append playwright output directory instructions to prompt",
		playwrightPromptFile,
		hasPlaywrightTool(data.ParsedTools))
}

// ============================================================================
// PR Context Prompts
// ============================================================================

// generatePRContextPromptStep generates a separate step for PR context instructions
func (c *Compiler) generatePRContextPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if any of the workflow's event triggers are comment-related events
	hasCommentTriggers := c.hasCommentRelatedTriggers(data)

	if !hasCommentTriggers {
		promptsLog.Print("Skipping PR context prompt: no comment-related triggers")
		return // No comment-related triggers, skip PR context instructions
	}

	// Also check if checkout step will be added - only show prompt if checkout happens
	needsCheckout := c.shouldAddCheckoutStep(data)
	if !needsCheckout {
		promptsLog.Print("Skipping PR context prompt: no checkout step needed")
		return // No checkout, so no PR branch checkout will happen
	}

	promptsLog.Print("Generating PR context prompt step for comment-triggered workflow")

	// Check that permissions allow contents read access
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		return // No contents read access, cannot checkout
	}

	// Build the condition string
	condition := BuildPRCommentCondition()

	// Use shared helper but we need to render condition manually since it requires RenderConditionAsIf
	// which is more complex than a simple if: string
	yaml.WriteString("      - name: Append PR context instructions to prompt\n")
	RenderConditionAsIf(yaml, condition, "          ")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptFileToYAML(yaml, prContextPromptFile, "          ")
}

// hasCommentRelatedTriggers checks if the workflow has any comment-related event triggers
func (c *Compiler) hasCommentRelatedTriggers(data *WorkflowData) bool {
	// Check for command trigger (which expands to comment events)
	if len(data.Command) > 0 {
		return true
	}

	if data.On == "" {
		return false
	}

	// Check for comment-related event types in the "on" configuration
	commentEvents := []string{"issue_comment", "pull_request_review_comment", "pull_request_review"}
	for _, event := range commentEvents {
		if strings.Contains(data.On, event) {
			return true
		}
	}

	return false
}

// ============================================================================
// Infrastructure Prompts - Temporary Folder
// ============================================================================

// generateTempFolderPromptStep generates a separate step for temporary folder usage instructions
func (c *Compiler) generateTempFolderPromptStep(yaml *strings.Builder) {
	generateStaticPromptStepFromFile(yaml,
		"Append temporary folder instructions to prompt",
		tempFolderPromptFile,
		true) // Always include temp folder instructions
}

// ============================================================================
// GitHub Context Prompts
// ============================================================================

// generateGitHubContextPromptStep generates a separate step for GitHub context information
// when the github tool is enabled. This injects repository, issue, discussion, pull request,
// comment, and run ID information into the prompt.
//
// The function uses generateStaticPromptStepWithExpressions to securely handle the GitHub
// Actions expressions in the context prompt. This extracts ${{ ... }} expressions into
// environment variables and uses shell variable expansion in the heredoc, preventing
// template injection vulnerabilities.
func (c *Compiler) generateGitHubContextPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStepWithExpressions(yaml,
		"Append GitHub context to prompt",
		githubContextPromptText,
		hasGitHubTool(data.ParsedTools))
}
