package workflow

import (
	"strings"
)

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
// Tool Prompts - Playwright
// ============================================================================

// hasPlaywrightTool checks if the playwright tool is enabled in the tools configuration
func hasPlaywrightTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.Playwright != nil
}

// ============================================================================
// PR Context Prompts
// ============================================================================

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
