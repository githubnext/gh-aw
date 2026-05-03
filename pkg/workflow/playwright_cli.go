package workflow

// Package workflow provides support for Playwright CLI mode.
//
// # Playwright CLI Mode
//
// When tools.playwright.mode is set to "cli", the compiler installs the
// @playwright/cli npm package instead of launching the Docker-based MCP server.
// This is a token-efficient alternative for coding agents that prefer CLI-based
// workflows over MCP: CLI invocations avoid loading large tool schemas and verbose
// accessibility trees into the model context.
//
// See https://github.com/microsoft/playwright-cli for details.
//
// In CLI mode:
//   - The mcr.microsoft.com/playwright/mcp Docker image is NOT pulled.
//   - playwright is NOT registered as an MCP server in the gateway config.
//   - @playwright/cli is installed via npm (global) before the agent runs.
//   - playwright-cli install --skills installs agent skill files so the coding
//     agent can discover and use the available playwright-cli commands.
//   - The agent uses `playwright-cli <command>` directly via bash.
//
// Example workflow frontmatter:
//
//	tools:
//	  playwright:
//	    mode: cli

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var playwrightCLILog = logger.New("workflow:playwright_cli")

// isPlaywrightCLIMode returns true when the playwright tool in the given tools map
// is configured with mode: cli.
func isPlaywrightCLIMode(tools map[string]any) bool {
	playwrightTool, ok := tools["playwright"]
	if !ok || playwrightTool == false {
		return false
	}
	config := parsePlaywrightTool(playwrightTool)
	return config != nil && config.IsCLIMode()
}

// generatePlaywrightCLIInstallSteps returns npm install steps for @playwright/cli
// when playwright is configured in CLI mode. Returns nil if playwright is in MCP mode.
//
// Node.js setup is intentionally omitted here because all supported engines
// (copilot, claude, codex, gemini) include a Node.js setup step in their own
// installation steps, which run before this function is called.
func generatePlaywrightCLIInstallSteps(workflowData *WorkflowData) []GitHubActionStep {
	if !isPlaywrightCLIMode(workflowData.Tools) {
		return nil
	}

	playwrightCLILog.Print("Generating @playwright/cli install steps (CLI mode)")

	version := string(constants.DefaultPlaywrightCLIVersion)
	// Use version override from playwright config if provided
	if playwrightTool, ok := workflowData.Tools["playwright"]; ok {
		config := parsePlaywrightTool(playwrightTool)
		if config != nil && config.Version != "" {
			version = config.Version
			playwrightCLILog.Printf("Using playwright CLI version from config: %s", version)
		}
	}

	// Install @playwright/cli globally.
	// Node.js setup is needed only when a custom engine.command is specified because
	// in that case the engine's own install steps (which normally set up Node) are skipped.
	// When EngineConfig is nil or Command is empty (standard engine configuration), Node.js
	// is already set up by the engine install steps that run before this function is called.
	needsNodeSetup := workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != ""
	steps := GenerateNpmInstallStepsWithScope(
		"@playwright/cli",
		version,
		"Install Playwright CLI",
		"playwright-cli",
		needsNodeSetup, // true only when engine.command skips standard engine install steps
		true,           // Global install so playwright-cli is on PATH
		true,           // Allow install scripts for browser setup
	)

	// Install playwright-cli skills so the coding agent can discover available commands.
	installSkillsStep := GitHubActionStep{
		"      - name: Install Playwright CLI skills",
		"        run: playwright-cli install --skills",
	}
	steps = append(steps, installSkillsStep)

	playwrightCLILog.Printf("Generated %d Playwright CLI install steps", len(steps))
	return steps
}
