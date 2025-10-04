package workflow

import (
	"fmt"
	"strings"
)

// addNodeJsSetupIfNeeded adds Node.js setup step if it's not already present in custom steps
// and if the engine requires it (npm-based engines like claude, codex, copilot)
func addNodeJsSetupIfNeeded(yaml *strings.Builder, data *WorkflowData) {
	// Check if Node.js is already set up in custom steps
	nodeJsAlreadySetup := false
	if data.CustomSteps != "" {
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
//
// Returns steps for installing the npm package (does NOT include Node.js setup or caching)
func GenerateNpmInstallSteps(packageName, version, stepName, cacheKeyPrefix string) []GitHubActionStep {
	installCmd := fmt.Sprintf("npm install -g %s@%s", packageName, version)

	return []GitHubActionStep{
		{
			fmt.Sprintf("      - name: %s", stepName),
			fmt.Sprintf("        run: %s", installCmd),
		},
	}
}
