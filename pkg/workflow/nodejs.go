package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var nodejsLog = logger.New("workflow:nodejs")

// GenerateNodeJsSetupStep creates a GitHub Actions step for setting up Node.js
// Returns a step that installs Node.js v24
// Caching is disabled by default to prevent cache poisoning vulnerabilities in release workflows
func GenerateNodeJsSetupStep() GitHubActionStep {
	return GitHubActionStep{
		"      - name: Setup Node.js",
		fmt.Sprintf("        uses: %s", GetActionPin("actions/setup-node")),
		"        with:",
		"          node-version: '24'",
		"          package-manager-cache: false",
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
	return GenerateNpmInstallStepsWithScope(packageName, version, stepName, cacheKeyPrefix, includeNodeSetup, true)
}

// GenerateNpmInstallStepsWithScope generates npm installation steps with control over global vs local installation
func GenerateNpmInstallStepsWithScope(packageName, version, stepName, cacheKeyPrefix string, includeNodeSetup bool, isGlobal bool) []GitHubActionStep {
	nodejsLog.Printf("Generating npm install steps: package=%s, version=%s, includeNodeSetup=%v, isGlobal=%v", packageName, version, includeNodeSetup, isGlobal)

	var steps []GitHubActionStep

	// Add Node.js setup if requested
	if includeNodeSetup {
		nodejsLog.Print("Including Node.js setup step")
		steps = append(steps, GenerateNodeJsSetupStep())
	}

	// Add npm install step
	globalFlag := ""
	if isGlobal {
		globalFlag = "-g "
	}
	installCmd := fmt.Sprintf("npm install %s%s@%s", globalFlag, packageName, version)
	steps = append(steps, GitHubActionStep{
		fmt.Sprintf("      - name: %s", stepName),
		fmt.Sprintf("        run: %s", installCmd),
	})

	return steps
}

// GenerateNpmInstallStepsWithVerification generates npm installation steps with CLI verification commands
// This is used for CLI packages like @github/copilot that support help commands
func GenerateNpmInstallStepsWithVerification(packageName, version, stepName, cliName string, includeNodeSetup bool, isGlobal bool) []GitHubActionStep {
	nodejsLog.Printf("Generating npm install steps with verification: package=%s, version=%s, cli=%s", packageName, version, cliName)

	var steps []GitHubActionStep

	// Add Node.js setup if requested
	if includeNodeSetup {
		nodejsLog.Print("Including Node.js setup step")
		steps = append(steps, GenerateNodeJsSetupStep())
	}

	// Build npm install command
	globalFlag := ""
	if isGlobal {
		globalFlag = "-g "
	}
	installCmd := fmt.Sprintf("npm install %s%s@%s", globalFlag, packageName, version)

	// Build verification commands for Copilot CLI
	// Run help on top-level commands to check for flag changes
	var stepLines []string
	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        run: |")
	stepLines = append(stepLines, fmt.Sprintf("          %s", installCmd))
	stepLines = append(stepLines, fmt.Sprintf("          %s --version", cliName))
	stepLines = append(stepLines, "          # Check help on key commands to detect flag changes")
	stepLines = append(stepLines, fmt.Sprintf("          %s help config || true", cliName))
	stepLines = append(stepLines, fmt.Sprintf("          %s help environment || true", cliName))

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}
