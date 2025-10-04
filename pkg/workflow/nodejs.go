package workflow

import (
	"fmt"
	"strings"
)

// GenerateNodeJsSetupStep creates a GitHub Actions step for setting up Node.js
// Returns a step that installs Node.js v24
func GenerateNodeJsSetupStep() GitHubActionStep {
	return GitHubActionStep{
		"      - name: Setup Node.js",
		"        uses: actions/setup-node@v4",
		"        with:",
		"          node-version: '24'",
	}
}

// addNodeJsSetupIfNeeded adds Node.js setup step if it's not already present in custom steps
// and if the engine requires it (npm-based engines like claude, codex, copilot)
// skipCustomStepsCheck: if true, always add Node.js setup (used for detection job)
func addNodeJsSetupIfNeeded(yaml *strings.Builder, data *WorkflowData, skipCustomStepsCheck bool) {
	// Check if Node.js is already set up in custom steps (unless we're skipping the check)
	nodeJsAlreadySetup := false
	if !skipCustomStepsCheck && data.CustomSteps != "" {
		if strings.Contains(data.CustomSteps, "actions/setup-node") || strings.Contains(data.CustomSteps, "Setup Node.js") {
			nodeJsAlreadySetup = true
		}
	}

	// If Node.js is not already set up and the engine is npm-based (claude, codex, copilot), add it
	if !nodeJsAlreadySetup && (data.AI == "claude" || data.AI == "codex" || data.AI == "copilot" || data.AI == "") {
		yaml.WriteString("      - name: Setup Node.js\n")
		yaml.WriteString("        uses: actions/setup-node@v4\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          node-version: '24'\n")
	}
}

// GenerateNpmInstallSteps creates GitHub Actions steps for installing an npm package globally
// Parameters:
//   - packageName: The npm package name (e.g., "@anthropic-ai/claude-code")
//   - version: The package version to install
//   - stepName: The name to display for the install step (e.g., "Install Claude Code CLI")
//   - cacheKeyPrefix: The prefix for the cache key (unused, kept for API compatibility)
//   - includeNodeSetup: If true, includes Node.js setup step before npm install
//
// Returns steps for installing the npm package (optionally with Node.js setup)
func GenerateNpmInstallSteps(packageName, version, stepName, cacheKeyPrefix string, includeNodeSetup bool) []GitHubActionStep {
	var steps []GitHubActionStep

	// Add Node.js setup if requested
	if includeNodeSetup {
		steps = append(steps, GenerateNodeJsSetupStep())
	}

	// Add npm install step
	installCmd := fmt.Sprintf("npm install -g %s@%s", packageName, version)
	steps = append(steps, GitHubActionStep{
		fmt.Sprintf("      - name: %s", stepName),
		fmt.Sprintf("        run: %s", installCmd),
	})

	return steps
}
