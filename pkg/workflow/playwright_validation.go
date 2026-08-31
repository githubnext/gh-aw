// This file provides playwright tool validation for agentic workflows.
//
// # Playwright Mode Validation
//
// validatePlaywrightMode rejects the removed MCP mode. Playwright uses CLI mode
// when mode is omitted or set to cli.
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

import "strings"

// validatePlaywrightMode validates that Playwright mode is static and rejects
// the removed built-in MCP integration.
func (c *Compiler) validatePlaywrightMode(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.Tools == nil {
		return nil
	}

	playwrightTool, ok := workflowData.Tools["playwright"]
	if !ok || playwrightTool == false {
		return nil
	}
	if config, ok := playwrightTool.(map[string]any); ok {
		if mode, ok := config["mode"].(string); ok && hasExpressionMarker(mode) {
			return NewValidationError(
				"tools.playwright.mode",
				mode,
				"mode must be a literal value; expressions are not allowed",
				"Set mode to cli, or omit mode because CLI is the default",
			)
		}
		if mode, ok := config["mode"].(string); ok && strings.EqualFold(mode, "mcp") {
			return NewValidationError(
				"tools.playwright.mode",
				mode,
				"built-in Playwright MCP support has been removed",
				"Remove `mode: mcp` or change it to `mode: cli`, then update prompts to run `playwright-cli <command>` from bash. If MCP is still required, configure Playwright explicitly under `mcp-servers`. See https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/playwright.md",
			)
		}
	}
	return nil
}
