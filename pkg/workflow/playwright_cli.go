package workflow

// Package workflow provides support for Playwright CLI mode.
//
// # Playwright CLI Mode
//
// When tools.playwright is enabled, the compiler installs the @playwright/cli
// npm package. CLI is the only built-in Playwright integration.
// This is a token-efficient alternative for coding agents that prefer CLI-based
// workflows over MCP: CLI invocations avoid loading large tool schemas and verbose
// accessibility trees into the model context.
//
// See https://github.com/microsoft/playwright-cli for details.
//
// In CLI mode:
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

var playwrightBrowserNames = []string{"chromium", "firefox", "webkit"}

// isPlaywrightCLIMode returns true when the Playwright tool is enabled in the
// supported CLI mode.
func isPlaywrightCLIMode(tools map[string]any) bool {
	playwrightTool, ok := tools["playwright"]
	if !ok || playwrightTool == false {
		return false
	}
	config := parsePlaywrightTool(playwrightTool)
	return config != nil && config.IsCLIMode()
}

// generatePlaywrightCLIInstallSteps returns npm install steps for @playwright/cli
// when playwright is enabled.
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
		NPMInstallOptions{
			IncludeNodeSetup:  needsNodeSetup,
			IsGlobal:          true,
			RunInstallScripts: true,
			CooldownEnabled:   resolveRuntimeCooldown(workflowData, "node"),
		},
	)
	for i := range steps {
		if len(steps[i]) > 0 && steps[i][0] == "      - name: Install Playwright CLI" {
			steps[i] = append(steps[i], "        timeout-minutes: 10")
			break
		}
	}

	// Install the supported browser binaries before the agent starts. Browser downloads
	// are prohibited during agent execution.
	steps = append(steps, generatePlaywrightBrowserInstallSteps()...)

	// Install playwright-cli skills so the coding agent can discover available commands.
	// This step only installs skills; browser binaries are provisioned above.
	installSkillsStep := GitHubActionStep{
		"      - name: Install Playwright CLI skills",
		"        run: playwright-cli install --skills",
		"        env:",
		"          PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: '1'",
		"        timeout-minutes: 10",
	}
	steps = append(steps, installSkillsStep)

	playwrightCLILog.Printf("Generated %d Playwright CLI install steps", len(steps))
	return steps
}

func generatePlaywrightBrowserInstallSteps() []GitHubActionStep {
	steps := make([]GitHubActionStep, 0, len(playwrightBrowserNames))
	for _, browser := range playwrightBrowserNames {
		steps = append(steps, GitHubActionStep{
			"      - name: Install Playwright " + browser + " browser",
			"        run: playwright-cli install-browser " + browser,
			"        timeout-minutes: 10",
		})
	}
	return steps
}
