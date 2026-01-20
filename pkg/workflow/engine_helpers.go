// Package workflow provides shared helper functions for AI engine implementations.
//
// This file contains utilities used across multiple AI engine files (copilot_engine.go,
// claude_engine.go, codex_engine.go, custom_engine.go) to generate common workflow
// steps and configurations.
//
// # Organization Rationale
//
// These helper functions are grouped here because they:
//   - Are used by 3+ engine implementations (shared utilities)
//   - Provide common patterns for agent installation and npm setup
//   - Have a clear domain focus (engine workflow generation)
//   - Are stable and change infrequently
//
// This follows the helper file conventions documented in skills/developer/SKILL.md.
//
// # Key Functions
//
// Agent Installation:
//   - GenerateAgentInstallSteps() - Generate agent installation workflow steps
//
// NPM Installation:
//   - GenerateNpmInstallStep() - Generate npm package installation step
//   - GenerateEngineDependenciesInstallStep() - Generate engine dependencies install step
//
// Configuration:
//   - GetClaudeSystemPrompt() - Get system prompt for Claude engine
//
// These functions encapsulate shared logic that would otherwise be duplicated across
// engine files, maintaining DRY principles while keeping engine-specific code separate.
package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var engineHelpersLog = logger.New("workflow:engine_helpers")

// EngineInstallConfig contains configuration for engine installation steps.
// This struct centralizes the configuration needed to generate the common
// installation steps shared by all engines (secret validation and npm installation).
type EngineInstallConfig struct {
	// Secrets is a list of secret names to validate (at least one must be set)
	Secrets []string
	// DocsURL is the documentation URL shown when secret validation fails
	DocsURL string
	// NpmPackage is the npm package name (e.g., "@github/copilot")
	NpmPackage string
	// Version is the default version of the npm package
	Version string
	// Name is the engine display name for secret validation messages (e.g., "Claude Code")
	Name string
	// CliName is the CLI name used for cache key prefix (e.g., "copilot")
	CliName string
	// InstallStepName is the display name for the npm install step (e.g., "Install Claude Code CLI")
	InstallStepName string
}

// GetBaseInstallationSteps returns the common installation steps for an engine.
// This includes secret validation and npm package installation steps that are
// shared across all engines.
//
// Parameters:
//   - config: Engine-specific configuration for installation
//   - workflowData: The workflow data containing engine configuration
//
// Returns:
//   - []GitHubActionStep: The base installation steps (secret validation + npm install)
func GetBaseInstallationSteps(config EngineInstallConfig, workflowData *WorkflowData) []GitHubActionStep {
	engineHelpersLog.Printf("Generating base installation steps for %s engine: workflow=%s", config.Name, workflowData.Name)

	var steps []GitHubActionStep

	// Add secret validation step
	secretValidation := GenerateMultiSecretValidationStep(
		config.Secrets,
		config.Name,
		config.DocsURL,
	)
	steps = append(steps, secretValidation)

	// Determine step name - use InstallStepName if provided, otherwise default to "Install <Name>"
	stepName := config.InstallStepName
	if stepName == "" {
		stepName = fmt.Sprintf("Install %s", config.Name)
	}

	// Add npm package installation steps
	npmSteps := BuildStandardNpmEngineInstallSteps(
		config.NpmPackage,
		config.Version,
		stepName,
		config.CliName,
		workflowData,
	)
	steps = append(steps, npmSteps...)

	return steps
}

// ExtractAgentIdentifier extracts the agent identifier (filename without extension) from an agent file path.
// This is used by the Copilot CLI which expects agent identifiers, not full paths.
//
// Parameters:
//   - agentFile: The relative path to the agent file (e.g., ".github/agents/test-agent.md" or ".github/agents/test-agent.agent.md")
//
// Returns:
//   - string: The agent identifier (e.g., "test-agent")
//
// Example:
//
//	identifier := ExtractAgentIdentifier(".github/agents/my-agent.md")
//	// Returns: "my-agent"
//
//	identifier := ExtractAgentIdentifier(".github/agents/my-agent.agent.md")
//	// Returns: "my-agent"
func ExtractAgentIdentifier(agentFile string) string {
	engineHelpersLog.Printf("Extracting agent identifier from: %s", agentFile)
	// Extract the base filename from the path
	lastSlash := strings.LastIndex(agentFile, "/")
	filename := agentFile
	if lastSlash >= 0 {
		filename = agentFile[lastSlash+1:]
	}

	// Remove extensions in order: .agent.md, then .md, then .agent
	// This handles all possible agent file naming conventions
	filename = strings.TrimSuffix(filename, ".agent.md")
	filename = strings.TrimSuffix(filename, ".md")
	filename = strings.TrimSuffix(filename, ".agent")

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

// FilterEnvForSecrets filters environment variables to only include allowed secrets
// This is a security measure to ensure that only necessary secrets are passed to the execution step
//
// Parameters:
//   - env: Map of all environment variables
//   - allowedSecrets: List of secret names that are allowed to be passed
//
// Returns:
//   - map[string]string: Filtered environment variables with only allowed secrets
func FilterEnvForSecrets(env map[string]string, allowedSecrets []string) map[string]string {
	engineHelpersLog.Printf("Filtering environment variables: total=%d, allowed_secrets=%d", len(env), len(allowedSecrets))

	// Create a set of allowed secret names for fast lookup
	allowedSet := make(map[string]bool)
	for _, secret := range allowedSecrets {
		allowedSet[secret] = true
	}

	filtered := make(map[string]string)
	secretsRemoved := 0

	for key, value := range env {
		// Check if this env var is a secret reference (starts with "${{ secrets.")
		if strings.Contains(value, "${{ secrets.") {
			// Extract the secret name from the expression
			// Format: ${{ secrets.SECRET_NAME }} or ${{ secrets.SECRET_NAME || ... }}
			secretName := extractSecretName(value)
			if secretName != "" && !allowedSet[secretName] {
				engineHelpersLog.Printf("Removing unauthorized secret from env: %s (secret: %s)", key, secretName)
				secretsRemoved++
				continue
			}
		}
		filtered[key] = value
	}

	engineHelpersLog.Printf("Filtered environment variables: kept=%d, removed=%d", len(filtered), secretsRemoved)
	return filtered
}

// extractSecretName extracts the secret name from a GitHub Actions secret expression
// Example inputs:
//   - "${{ secrets.COPILOT_GITHUB_TOKEN }}" -> "COPILOT_GITHUB_TOKEN"
//   - "${{ secrets.API_KEY || secrets.FALLBACK_KEY }}" -> "API_KEY"
//   - "${{ env.NOT_A_SECRET }}" -> ""
func extractSecretName(expr string) string {
	// Look for "${{ secrets.NAME" pattern
	prefix := "${{ secrets."
	idx := strings.Index(expr, prefix)
	if idx == -1 {
		return ""
	}

	// Extract everything after "${{ secrets."
	rest := expr[idx+len(prefix):]

	// Find the end of the secret name (space, }, or ||)
	endIdx := len(rest)
	for i, ch := range rest {
		if ch == ' ' || ch == '}' || ch == '|' {
			endIdx = i
			break
		}
	}

	return rest[:endIdx]
}

// WrapCommandWithIterationLoop wraps a command with the iteration loop script if iterations > 1
// This implements the ralph-loop pattern where each iteration appends its output to the prompt
// and re-runs the agent with the augmented context.
//
// Parameters:
//   - command: The base command to execute (e.g., "copilot --prompt ...")
//   - logFile: The log file path for capturing output
//   - workflowData: The workflow data containing engine configuration
//
// Returns:
//   - string: The wrapped command (or original if iterations <= 1)
//
// Example:
//   - Input: "copilot --prompt prompt.txt", iterations=3
//   - Output: "/opt/gh-aw/actions/run_agent_loop.sh copilot --prompt prompt.txt"
func WrapCommandWithIterationLoop(command string, logFile string, workflowData *WorkflowData) string {
	// Check if iterations is configured and > 1
	if workflowData.EngineConfig == nil || workflowData.EngineConfig.Iterations == "" {
		return command
	}

	iterations := workflowData.EngineConfig.Iterations
	engineHelpersLog.Printf("Iterations configured: %s", iterations)

	// For iterations = 1, no need to wrap
	if iterations == "1" {
		engineHelpersLog.Print("Iterations = 1, skipping loop wrapper")
		return command
	}

	engineHelpersLog.Printf("Wrapping command with iteration loop (iterations=%s)", iterations)

	// The loop script expects:
	// - GH_AW_ITERATIONS: number of iterations
	// - GH_AW_PROMPT: path to prompt file
	// - GH_AW_LOGS_DIR: directory for logs
	// - GH_AW_LOG_FILE: final log file path
	// - Command to execute as arguments

	// Extract the actual command (remove the tee redirection if present)
	// The loop script will handle its own logging
	actualCommand := command
	if strings.Contains(command, "| tee") {
		// Remove the tee redirection - the loop script will handle logging
		parts := strings.Split(command, "| tee")
		if len(parts) > 0 {
			actualCommand = strings.TrimSpace(parts[0])
		}
	}

	// Wrap with the loop script
	// The loop script will execute the command and handle iteration logic
	wrappedCommand := fmt.Sprintf(`GH_AW_ITERATIONS=%s \
GH_AW_LOGS_DIR=/tmp/gh-aw/logs \
GH_AW_LOG_FILE=%s \
/opt/gh-aw/actions/run_agent_loop.sh \
%s`, iterations, shellEscapeArg(logFile), actualCommand)

	engineHelpersLog.Print("Command wrapped with iteration loop script")
	return wrappedCommand
}
