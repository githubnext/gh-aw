// This file provides shared helper functions for AI engine implementations.
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
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineHelpersLog = logger.New("workflow:engine_helpers")

// agentFilePathRegex is an allowlist of safe characters for agent file paths.
// Only alphanumeric characters, dots, underscores, hyphens, forward slashes, and
// spaces are permitted. This prevents shell metacharacters (such as ", $, `, ;, |)
// from being embedded in paths that are later interpolated into shell commands.
var agentFilePathRegex = regexp.MustCompile(`^[a-zA-Z0-9._/\- ]+$`)

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

// getEngineEnvOverrides returns the engine.env map from workflowData, or nil if not set.
// This is used to pass user-provided env overrides to steps such as secret validation,
// so that overridden token expressions are used instead of the default "${{ secrets.KEY }}".
func getEngineEnvOverrides(workflowData *WorkflowData) map[string]string {
	if workflowData == nil || workflowData.EngineConfig == nil {
		return nil
	}
	return workflowData.EngineConfig.Env
}

// GetBaseInstallationSteps returns the common installation steps for an engine.
// This includes npm package installation steps shared across all engines.
// Secret validation is now handled in the activation job via GetSecretValidationStep.
//
// Parameters:
//   - config: Engine-specific configuration for installation
//   - workflowData: The workflow data containing engine configuration
//
// Returns:
//   - []GitHubActionStep: The base installation steps (npm install)
func GetBaseInstallationSteps(config EngineInstallConfig, workflowData *WorkflowData) []GitHubActionStep {
	engineHelpersLog.Printf("Generating base installation steps for %s engine: workflow=%s", config.Name, workflowData.Name)

	var steps []GitHubActionStep

	// Secret validation step is now generated in the activation job (GetSecretValidationStep).

	// Determine step name - use InstallStepName if provided, otherwise default to "Install <Name>"
	stepName := config.InstallStepName
	if stepName == "" {
		stepName = "Install " + config.Name
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
	// Always pass false for runInstallScripts: engine CLI installs must never run
	// pre/post install scripts regardless of the workflow's run-install-scripts setting.
	// This is a supply chain security requirement for the engine binary itself.
	return GenerateNpmInstallSteps(
		packageName,
		version,
		stepName,
		cacheKeyPrefix,
		true,  // Include Node.js setup
		false, // Always disable scripts for engine CLI installs
	)
}

// BuildNpmEngineInstallStepsWithAWF injects an AWF installation step between the Node.js
// setup step and the CLI install steps when the firewall is enabled. This eliminates the
// duplicated AWF-injection pattern shared by Claude, Gemini, and Copilot engines.
//
// The expected layout of npmSteps is:
//   - npmSteps[0]  – Node.js setup step
//   - npmSteps[1:] – CLI installation step(s)
//
// Parameters:
//   - npmSteps: Pre-computed npm installation steps (from BuildStandardNpmEngineInstallSteps
//     or GenerateCopilotInstallerSteps)
//   - workflowData: The workflow data (used to determine firewall configuration)
//
// Returns:
//   - []GitHubActionStep: Steps in order: Node.js setup, AWF (if enabled), CLI install
func BuildNpmEngineInstallStepsWithAWF(npmSteps []GitHubActionStep, workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Node.js setup step
	}

	// Inject AWF installation after Node.js setup but before the CLI install steps
	if isFirewallEnabled(workflowData) {
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
	}

	if len(npmSteps) > 1 {
		steps = append(steps, npmSteps[1:]...) // CLI installation and subsequent steps
	}

	return steps
}

// GenerateMultiSecretValidationStep creates a GitHub Actions step that validates at least one
// of multiple secrets is available.
// secretNames: slice of secret names to validate (e.g., []string{"CODEX_API_KEY", "OPENAI_API_KEY"})
// engineName: the display name of the engine (e.g., "Codex")
// docsURL: URL to the documentation page for setting up the secret
// envOverrides: optional map of env var key to expression override (from engine.env); when set,
// the overridden expression is used instead of the default "${{ secrets.KEY }}" so the
// validation step checks the user-provided secret reference rather than the default one.
func GenerateMultiSecretValidationStep(secretNames []string, engineName, docsURL string, envOverrides map[string]string) GitHubActionStep {
	if len(secretNames) == 0 {
		// This is a programming error - engine configurations should always provide secrets
		// Log the error and return empty step to avoid breaking compilation
		engineHelpersLog.Printf("ERROR: GenerateMultiSecretValidationStep called with empty secretNames for engine %s", engineName)
		return GitHubActionStep{}
	}

	// Build the step name
	stepName := fmt.Sprintf("      - name: Validate %s secret", strings.Join(secretNames, " or "))

	// Build the command to call the validation script
	// The script expects: SECRET_NAME1 [SECRET_NAME2 ...] ENGINE_NAME DOCS_URL
	// Use shellJoinArgs to properly escape multi-word engine names and special characters
	scriptArgs := append(secretNames, engineName, docsURL)
	scriptArgsStr := shellJoinArgs(scriptArgs)

	stepLines := []string{
		stepName,
		"        id: validate-secret",
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/validate_multi_secret.sh\" " + scriptArgsStr,
		"        env:",
	}

	// Add env section with all secrets. When engine.env provides an override for a key,
	// use that expression (e.g. "${{ secrets.MY_ORG_TOKEN }}") so the validation step
	// validates the user-supplied secret instead of the default one.
	for _, secretName := range secretNames {
		expr := fmt.Sprintf("${{ secrets.%s }}", secretName)
		if envOverrides != nil {
			if override, ok := envOverrides[secretName]; ok {
				expr = override
			}
		}
		stepLines = append(stepLines, fmt.Sprintf("          %s: %s", secretName, expr))
	}

	return GitHubActionStep(stepLines)
}

// BuildDefaultSecretValidationStep returns a secret validation step for the given engine
// configuration, or an empty step when a custom command is specified. This consolidates
// the common guard+delegate pattern shared across all engine GetSecretValidationStep
// implementations.
//
// Parameters:
//   - workflowData: The workflow data (checked for custom command)
//   - secrets: The secret names to validate (e.g., []string{"ANTHROPIC_API_KEY"})
//   - name: The engine display name used in the step (e.g., "Claude Code")
//   - docsURL: The documentation URL shown when validation fails
//
// Returns:
//   - GitHubActionStep: The validation step, or an empty step if a custom command is set
func BuildDefaultSecretValidationStep(workflowData *WorkflowData, secrets []string, name, docsURL string) GitHubActionStep {
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		engineHelpersLog.Printf("Skipping secret validation step: custom command specified (%s)", workflowData.EngineConfig.Command)
		return GitHubActionStep{}
	}
	if workflowData != nil && strings.TrimSpace(workflowData.Environment) != "" {
		engineHelpersLog.Print("Skipping secret validation step: top-level environment is configured")
		return GitHubActionStep{}
	}
	return GenerateMultiSecretValidationStep(secrets, name, docsURL, getEngineEnvOverrides(workflowData))
}

// collectCommonMCPSecrets returns the MCP-related secret names shared across all engines:
//   - MCP_GATEWAY_API_KEY (when MCP servers are present)
//   - mcp-scripts secrets (when mcp-scripts feature is enabled)
//
// Parameters:
//   - workflowData: The workflow data used to check MCP server and mcp-scripts configuration
//
// Returns:
//   - []string: Common MCP secret names (may be empty)
func collectCommonMCPSecrets(workflowData *WorkflowData) []string {
	var secrets []string

	if HasMCPServers(workflowData) {
		secrets = append(secrets, "MCP_GATEWAY_API_KEY")
	}

	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpScriptsSecrets := collectMCPScriptsSecrets(workflowData.MCPScripts)
		for varName := range mcpScriptsSecrets {
			secrets = append(secrets, varName)
		}
	}

	return secrets
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
	commandLines := strings.SplitSeq(command, "\n")
	for line := range commandLines {
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
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, yamlStringValue(value)))
		}
	}

	return stepLines
}

// yamlStringValue returns a YAML-safe representation of a string value.
// If the value starts with a YAML flow indicator ('{' or '[') or other characters
// that would cause it to be misinterpreted by YAML parsers, it wraps the value
// in single quotes. Any embedded single quotes are escaped by doubling them (' becomes ”).
func yamlStringValue(value string) string {
	if len(value) == 0 {
		return value
	}
	// Values starting with YAML flow indicators need quoting to be treated as strings.
	// '{' would be parsed as a YAML flow mapping, '[' as a YAML flow sequence.
	first := value[0]
	if first != '{' && first != '[' {
		return value
	}
	// Single-quote the value, escaping any embedded single quotes by doubling them.
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// FilterEnvForSecrets filters environment variables to only include allowed secrets.
// This is a security measure to ensure that only necessary secrets are passed to the execution step.
//
// An env var carrying a secret reference is kept when either:
//   - The referenced secret name (e.g. "COPILOT_GITHUB_TOKEN") is in allowedNamesAndKeys, OR
//   - The env var key itself (e.g. "COPILOT_GITHUB_TOKEN") is in allowedNamesAndKeys.
//
// The second rule allows users to override an engine's required env var with a
// differently-named secret, e.g. COPILOT_GITHUB_TOKEN: ${{ secrets.MY_ORG_TOKEN }}.
//
// Parameters:
//   - env: Map of all environment variables
//   - allowedNamesAndKeys: List of secret names and/or env var keys that are permitted
//
// Returns:
//   - map[string]string: Filtered environment variables with only allowed secrets
func FilterEnvForSecrets(env map[string]string, allowedNamesAndKeys []string) map[string]string {
	engineHelpersLog.Printf("Filtering environment variables: total=%d, allowed=%d", len(env), len(allowedNamesAndKeys))

	// Create a set for fast lookup — entries may be secret names or env var keys.
	allowedSet := make(map[string]bool)
	for _, entry := range allowedNamesAndKeys {
		allowedSet[entry] = true
	}

	filtered := make(map[string]string)
	secretsRemoved := 0

	for key, value := range env {
		// Check if this env var is a secret reference (starts with "${{ secrets.")
		if strings.Contains(value, "${{ secrets.") {
			// Extract the secret name from the expression
			// Format: ${{ secrets.SECRET_NAME }} or ${{ secrets.SECRET_NAME || ... }}
			secretName := ExtractSecretName(value)
			// Allow the secret if the secret name OR the env var key is in the allowed set.
			if secretName != "" && !allowedSet[secretName] && !allowedSet[key] {
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

// GetNpmBinPathSetup returns a simple shell command that adds hostedtoolcache bin directories
// to PATH. This is specifically for npm-installed CLIs (like Claude and Codex) that need
// to find their binaries installed via `npm install -g`.
//
// Unlike GetHostedToolcachePathSetup(), this does NOT use GH_AW_TOOL_BINS because AWF's
// native chroot mode already handles tool-specific paths (GOROOT, JAVA_HOME, etc.) via
// AWF_HOST_PATH and the entrypoint.sh script. This function only adds the generic
// hostedtoolcache bin directories for npm packages.
//
// Returns:
//   - string: A shell command that exports PATH with hostedtoolcache bin directories prepended
func GetNpmBinPathSetup() string {
	// Find all bin directories in hostedtoolcache (Node.js, Python, etc.)
	// This finds paths like /opt/hostedtoolcache/node/22.13.0/x64/bin
	//
	// After the find, re-prepend GOROOT/bin if set. The find returns directories
	// alphabetically, so go/1.23.12 shadows go/1.25.0. Re-prepending GOROOT/bin
	// ensures the Go version set by actions/setup-go takes precedence.
	// AWF's entrypoint.sh exports GOROOT before the user command runs.
	return `export PATH="$(find /opt/hostedtoolcache -maxdepth 4 -type d -name bin 2>/dev/null | tr '\n' ':')$PATH"; [ -n "$GOROOT" ] && export PATH="$GOROOT/bin:$PATH" || true`
}

// EngineHasValidateSecretStep checks if the engine provides a validate-secret step.
// This is used to determine whether the secret_verification_result job output should be added.
//
// The validate-secret step is provided by engines that override GetSecretValidationStep():
//   - Copilot engine: Adds step unless copilot-requests feature is enabled or custom command is set
//   - Claude engine: Adds step unless custom command is set
//   - Codex engine: Adds step unless custom command is set
//   - Gemini engine: Adds step unless custom command is set
//   - Custom engine: Never adds this step (uses BaseEngine default which returns empty)
//
// Parameters:
//   - engine: The agentic engine to check
//   - data: The workflow data (needed for GetSecretValidationStep)
//
// Returns:
//   - bool: true if the engine provides a validate-secret step, false otherwise
func EngineHasValidateSecretStep(engine CodingAgentEngine, data *WorkflowData) bool {
	step := engine.GetSecretValidationStep(data)
	return len(step) > 0
}
