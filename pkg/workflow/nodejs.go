package workflow

import (
	"fmt"
)

// GenerateNodeJsSetupStep creates a GitHub Actions step for setting up Node.js
// Returns a step that installs Node.js v24
func GenerateNodeJsSetupStep() GitHubActionStep {
	return GitHubActionStep{
		"      - name: Setup Node.js",
		fmt.Sprintf("        uses: %s", GetActionPinWithComment("actions/setup-node")),
		"        with:",
		"          node-version: '24'",
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
