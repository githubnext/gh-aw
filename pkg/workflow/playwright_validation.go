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
//     - New: run `playwright-cli goto <url>` in bash
//
//  3. Use `localhost` directly when accessing local servers — playwright-cli
//     runs on the runner host, not in a separate Docker container.

package workflow

import "strings"

func normalizePlaywrightBrowser(browser string) string {
	switch strings.ToLower(strings.TrimSpace(browser)) {
	case "chrome", "chromium":
		return "chromium"
	case "firefox":
		return "firefox"
	case "webkit":
		return "webkit"
	default:
		return ""
	}
}

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
		if browsers, ok := config["browsers"].([]any); ok {
			for _, browser := range browsers {
				name, ok := browser.(string)
				if !ok || normalizePlaywrightBrowser(name) == "" {
					return NewValidationError(
						"tools.playwright.browsers",
						name,
						"unsupported browser; choose chrome, chromium, firefox, or webkit",
						"Set browsers to a list containing supported Playwright browser names",
					)
				}
			}
			if len(browsers) == 0 {
				return NewValidationError(
					"tools.playwright.browsers",
					"[]",
					"at least one browser is required",
					"Omit browsers to use the default Chromium browser",
				)
			}
		}
	}
	return nil
}
