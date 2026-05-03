// This file provides playwright tool validation for agentic workflows.
//
// # Playwright Mode Validation
//
// validatePlaywrightMode warns when playwright is configured in MCP mode
// (the default when no mode is specified, or when mode: mcp is set explicitly).
// MCP mode is deprecated in favor of CLI mode (mode: cli), which is more
// token-efficient and does not require a separate Docker container.
//
// # Migration
//
// To migrate from MCP mode to CLI mode:
//
//  1. Add `mode: cli` to your playwright tool configuration:
//
//     tools:
//       playwright:
//         mode: cli
//
//  2. Update prompts to use `playwright-cli <command>` via bash instead of
//     MCP browser tool calls. For example:
//     - Old: use browser_navigate MCP tool
//     - New: run `playwright-cli browser_navigate --url <url>` in bash
//
//  3. Use `localhost` directly when accessing local servers — playwright-cli
//     runs on the runner host, not in a separate Docker container.

package workflow

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var playwrightValidationLog = logger.New("workflow:playwright_validation")

// validatePlaywrightMode warns (non-strict) or errors (strict) when the
// playwright tool is configured in MCP mode. MCP mode is deprecated; use
// mode: cli instead for token-efficient, container-free browser automation.
func (c *Compiler) validatePlaywrightMode(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.Tools == nil {
		return nil
	}

	playwrightTool, ok := workflowData.Tools["playwright"]
	if !ok || playwrightTool == false {
		return nil
	}

	if isPlaywrightCLIMode(workflowData.Tools) {
		playwrightValidationLog.Print("playwright mode: cli — no deprecation warning")
		return nil
	}

	playwrightValidationLog.Print("playwright mode: mcp detected — emitting deprecation warning")

	warningMsg := "tools.playwright: MCP mode is deprecated. " +
		"Migrate to CLI mode by adding `mode: cli` to your playwright configuration. " +
		"CLI mode runs playwright-cli directly on the runner (no Docker container required), " +
		"is more token-efficient, and lets you use `localhost` to reach local dev servers. " +
		"Update your prompts to run `playwright-cli <command>` in bash instead of using MCP browser tools. " +
		"See: https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/playwright.md"

	if c.strictMode {
		return fmt.Errorf("strict mode: %s", warningMsg)
	}

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
	c.IncrementWarningCount()
	return nil
}
