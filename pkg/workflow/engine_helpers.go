package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var engineHelpersLog = logger.New("workflow:engine_helpers")

// ExtractAgentIdentifier extracts the agent identifier (filename without extension) from an agent file path.
// This is used by the Copilot CLI which expects agent identifiers, not full paths.
//
// Parameters:
//   - agentFile: The relative path to the agent file (e.g., ".github/agents/test-agent.md")
//
// Returns:
//   - string: The agent identifier (e.g., "test-agent")
//
// Example:
//
//	identifier := ExtractAgentIdentifier(".github/agents/my-agent.md")
//	// Returns: "my-agent"
func ExtractAgentIdentifier(agentFile string) string {
	engineHelpersLog.Printf("Extracting agent identifier from: %s", agentFile)
	// Extract the base filename from the path
	lastSlash := strings.LastIndex(agentFile, "/")
	filename := agentFile
	if lastSlash >= 0 {
		filename = agentFile[lastSlash+1:]
	}

	// Remove the .md extension using TrimSuffix (unconditionally safe)
	filename = strings.TrimSuffix(filename, ".md")

	return filename
}

// ResolveAgentFilePath returns the properly quoted agent file path with GITHUB_WORKSPACE prefix.
// This helper extracts the common pattern shared by Copilot, Codex, and Claude engines.
//
// The agent file path is relative to the repository root, so we prefix it with ${GITHUB_WORKSPACE}
// and wrap the entire expression in double quotes to handle paths with spaces while allowing
// shell variable expansion.
//
// Parameters:
//   - agentFile: The relative path to the agent file (e.g., ".github/agents/test-agent.md")
//
// Returns:
//   - string: The double-quoted path with GITHUB_WORKSPACE prefix (e.g., "${GITHUB_WORKSPACE}/.github/agents/test-agent.md")
//
// Example:
//
//	agentPath := ResolveAgentFilePath(".github/agents/my-agent.md")
//	// Returns: "${GITHUB_WORKSPACE}/.github/agents/my-agent.md"
//
// Note: The entire path is wrapped in double quotes (not just the variable) to ensure:
//  1. The shellEscapeArg function recognizes it as already-quoted and doesn't add single quotes
//  2. Shell variable expansion works (${GITHUB_WORKSPACE} gets expanded inside double quotes)
//  3. Paths with spaces are properly handled
func ResolveAgentFilePath(agentFile string) string {
	return fmt.Sprintf("\"${GITHUB_WORKSPACE}/%s\"", agentFile)
}

// BuildStandardNpmEngineInstallSteps creates standard npm installation steps for engines
// This helper extracts the common pattern shared by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - packageName: The npm package name (e.g., "@github/copilot")
//   - defaultVersion: The default version constant (e.g., constants.DefaultCopilotVersion)
//   - stepName: The display name for the install step (e.g., "Install GitHub Copilot CLI")
//   - cacheKeyPrefix: The cache key prefix (e.g., "copilot")
//   - workflowData: The workflow data containing engine configuration
//
// Returns:
//   - []GitHubActionStep: The installation steps including Node.js setup
func BuildStandardNpmEngineInstallSteps(
	packageName string,
	defaultVersion string,
	stepName string,
	cacheKeyPrefix string,
	workflowData *WorkflowData,
) []GitHubActionStep {
	engineHelpersLog.Printf("Building npm engine install steps: package=%s, version=%s", packageName, defaultVersion)

	// Use version from engine config if provided, otherwise default to pinned version
	version := defaultVersion
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
		engineHelpersLog.Printf("Using engine config version: %s", version)
	}

	// Add npm package installation steps (includes Node.js setup)
	return GenerateNpmInstallSteps(
		packageName,
		version,
		stepName,
		cacheKeyPrefix,
		true, // Include Node.js setup
	)
}

// InjectCustomEngineSteps processes custom steps from engine config and converts them to GitHubActionSteps.
// This shared function extracts the common pattern used by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - workflowData: The workflow data containing engine configuration
//   - convertStepFunc: A function that converts a step map to YAML string (engine-specific)
//
// Returns:
//   - []GitHubActionStep: Array of custom steps ready to be included in the execution pipeline
func InjectCustomEngineSteps(
	workflowData *WorkflowData,
	convertStepFunc func(map[string]any) (string, error),
) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		engineHelpersLog.Printf("Injecting %d custom engine steps", len(workflowData.EngineConfig.Steps))
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := convertStepFunc(step)
			if err != nil {
				engineHelpersLog.Printf("Failed to convert custom step: %v", err)
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
		engineHelpersLog.Printf("Successfully injected %d custom engine steps", len(steps))
	}

	return steps
}

// RenderCustomMCPToolConfigHandler is a function type that engines must provide to render their specific MCP config
// FormatStepWithCommandAndEnv formats a GitHub Actions step with command and environment variables.
// This shared function extracts the common pattern used by Copilot and Codex engines.
//
// Parameters:
//   - stepLines: Existing step lines to append to (e.g., name, id, comments, timeout)
//   - command: The command to execute (may contain multiple lines)
//   - env: Map of environment variables to include in the step
//
// Returns:
//   - []string: Complete step lines including run command and env section
func FormatStepWithCommandAndEnv(stepLines []string, command string, env map[string]string) []string {
	engineHelpersLog.Printf("Formatting step with command and %d environment variables", len(env))
	// Add the run section
	stepLines = append(stepLines, "        run: |")

	// Split command into lines and indent them properly
	commandLines := strings.Split(command, "\n")
	for _, line := range commandLines {
		// Don't add indentation to empty lines
		if line == "" {
			stepLines = append(stepLines, "")
		} else {
			stepLines = append(stepLines, "          "+line)
		}
	}

	// Add environment variables
	if len(env) > 0 {
		stepLines = append(stepLines, "        env:")
		// Sort environment keys for consistent output
		envKeys := make([]string, 0, len(env))
		for key := range env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)

		for _, key := range envKeys {
			value := env[key]
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	return stepLines
}
