package workflow

import (
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var opencodeLog = logger.New("workflow:opencode_engine")

// OpenCodeEngine represents the OpenCode CLI agentic engine.
// OpenCode is a provider-agnostic, open-source AI coding agent that supports
// 75+ models via BYOK (Bring Your Own Key).
type OpenCodeEngine struct {
	BaseEngine
}

func NewOpenCodeEngine() *OpenCodeEngine {
	return &OpenCodeEngine{
		BaseEngine: BaseEngine{
			id:                     "opencode",
			displayName:            "OpenCode",
			description:            "OpenCode CLI with headless mode and multi-provider LLM support",
			experimental:           true,                             // Start as experimental until smoke tests pass consistently
			supportsToolsAllowlist: false,                            // OpenCode manages its own tool permissions via opencode.jsonc
			supportsMaxTurns:       false,                            // No --max-turns flag in opencode run
			supportsWebSearch:      false,                            // Has built-in websearch but not exposed via gh-aw neutral tools yet
			llmGatewayPort:         constants.OpenCodeLLMGatewayPort, // Port 10004
		},
	}
}

// SupportsLLMGateway returns the LLM gateway port for OpenCode engine
func (e *OpenCodeEngine) SupportsLLMGateway() int {
	return constants.OpenCodeLLMGatewayPort
}

// GetModelEnvVarName returns the native environment variable name that the OpenCode CLI uses
// for model selection. Setting OPENCODE_MODEL is equivalent to passing --model to the CLI.
func (e *OpenCodeEngine) GetModelEnvVarName() string {
	return constants.OpenCodeCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the OpenCode engine.
// By default, OpenCode routes through the Copilot API using COPILOT_GITHUB_TOKEN
// (or ${{ github.token }} when copilot-requests feature is enabled).
// Additional provider API keys can be added via engine.env overrides.
func (e *OpenCodeEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	opencodeLog.Print("Collecting required secrets for OpenCode engine")
	var secrets []string

	// Default: Copilot routing via COPILOT_GITHUB_TOKEN.
	// When copilot-requests feature is enabled, no secret is needed (uses github.token).
	if !isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData) {
		secrets = append(secrets, "COPILOT_GITHUB_TOKEN")
	}

	// Allow additional provider API keys from engine.env overrides
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key := range workflowData.EngineConfig.Env {
			if strings.HasSuffix(key, "_API_KEY") || strings.HasSuffix(key, "_KEY") {
				secrets = append(secrets, key)
			}
		}
	}

	// Add common MCP secrets (MCP_GATEWAY_API_KEY if MCP servers present, mcp-scripts secrets)
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)

	// Add GitHub token for GitHub MCP server if present
	if hasGitHubTool(workflowData.ParsedTools) {
		opencodeLog.Print("Adding GITHUB_MCP_SERVER_TOKEN secret")
		secrets = append(secrets, "GITHUB_MCP_SERVER_TOKEN")
	}

	// Add HTTP MCP header secret names
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName := range headerSecrets {
		secrets = append(secrets, varName)
	}
	if len(headerSecrets) > 0 {
		opencodeLog.Printf("Added %d HTTP MCP header secrets", len(headerSecrets))
	}

	return secrets
}

// GetInstallationSteps returns the GitHub Actions steps needed to install OpenCode CLI
func (e *OpenCodeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	opencodeLog.Printf("Generating installation steps for OpenCode engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		opencodeLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	npmSteps := BuildStandardNpmEngineInstallSteps(
		"opencode-ai",
		string(constants.DefaultOpenCodeVersion),
		"Install OpenCode CLI",
		"opencode",
		workflowData,
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetSecretValidationStep returns the secret validation step for the OpenCode engine.
// Returns an empty step if copilot-requests feature is enabled (uses GitHub Actions token).
func (e *OpenCodeEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	if isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData) {
		opencodeLog.Print("Skipping secret validation step: copilot-requests feature enabled, using GitHub Actions token")
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"COPILOT_GITHUB_TOKEN"},
		"OpenCode CLI",
		"https://github.github.com/gh-aw/reference/engines/#opencode",
	)
}

// GetAgentManifestFiles returns OpenCode-specific instruction files that should be
// treated as security-sensitive manifests. Modifying these files can change the
// agent's instructions, permissions, or configuration on the next run.
// opencode.jsonc is the primary OpenCode config file; AGENTS.md is the cross-engine
// convention that OpenCode also reads.
func (e *OpenCodeEngine) GetAgentManifestFiles() []string {
	return []string{"opencode.jsonc", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns OpenCode-specific config directory prefixes
// that must be protected from fork PR injection.
// The .opencode/ directory contains agent configuration, instructions, and other
// settings that could alter agent behaviour.
func (e *OpenCodeEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".opencode/"}
}

// GetDeclaredOutputFiles returns the output files that OpenCode may produce.
func (e *OpenCodeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing OpenCode
func (e *OpenCodeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	opencodeLog.Printf("Generating execution steps for OpenCode engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Step 1: Write opencode.jsonc config (permissions)
	configStep := e.generateOpenCodeConfigStep(workflowData)
	steps = append(steps, configStep)

	// Step 2: Build CLI arguments
	var opencodeArgs []string

	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Enable verbose logging for debugging in CI
	opencodeArgs = append(opencodeArgs, "--print-logs")
	opencodeArgs = append(opencodeArgs, "--log-level", "DEBUG")

	// Prompt from file (positional argument to `opencode run`).
	// Keep this outside shellJoinArgs so command substitution expands at runtime.
	promptArg := "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\""

	// Build command name
	commandName := "opencode"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	opencodeCommand := fmt.Sprintf("%s run %s %s", commandName, shellJoinArgs(opencodeArgs), promptArg)

	// AWF wrapping
	firewallEnabled := isFirewallEnabled(workflowData)
	var command string
	if firewallEnabled {
		// Resolve model for provider-specific domain allowlisting
		model := ""
		if modelConfigured {
			model = workflowData.EngineConfig.Model
		}
		allowedDomains := GetOpenCodeAllowedDomainsWithToolsAndRuntimes(
			model,
			workflowData.NetworkPermissions,
			workflowData.Tools,
			workflowData.Runtimes,
		)

		npmPathSetup := GetNpmBinPathSetup()
		opencodeCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, opencodeCommand)

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "opencode",
			EngineCommand:  opencodeCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			AllowedDomains: allowedDomains,
		})
	} else {
		command = fmt.Sprintf("set -o pipefail\n%s 2>&1 | tee -a %s", opencodeCommand, logFile)
	}

	// Environment variables — default to Copilot routing (OpenAI-compatible API).
	// OPENAI_API_KEY is set from COPILOT_GITHUB_TOKEN (or github.token with copilot-requests).
	// #nosec G101 -- These are NOT hardcoded credentials. They are GitHub Actions expression templates
	// that the runtime replaces with actual values.
	var openaiAPIKey string
	useCopilotRequests := isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData)
	if useCopilotRequests {
		openaiAPIKey = "${{ github.token }}"
		opencodeLog.Print("Using GitHub Actions token as OPENAI_API_KEY (copilot-requests feature enabled)")
	} else {
		openaiAPIKey = "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	}

	env := map[string]string{
		"OPENAI_API_KEY":   openaiAPIKey,
		"GH_AW_PROMPT":     "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
		"NO_PROXY":         "localhost,127.0.0.1",
	}

	// MCP config path
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/opencode.jsonc"
	}

	// LLM gateway base URL override (default Copilot routing via OpenAI-compatible endpoint)
	if firewallEnabled {
		env["OPENAI_BASE_URL"] = fmt.Sprintf("http://host.docker.internal:%d",
			constants.OpenCodeLLMGatewayPort)
	}

	// Safe outputs env
	applySafeOutputEnvToMap(env, workflowData)

	// Model env var (only when explicitly configured)
	if modelConfigured {
		opencodeLog.Printf("Setting %s env var for model: %s",
			constants.OpenCodeCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.OpenCodeCLIModelEnvVar] = workflowData.EngineConfig.Model
	}

	// Custom env from engine config (allows provider override)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Agent config env
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
	}

	// Build execution step
	stepLines := []string{
		"      - name: Execute OpenCode CLI",
		"        id: agentic_execution",
	}
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))
	return steps
}

// generateOpenCodeConfigStep writes opencode.jsonc with all permissions set to allow
// to prevent CI hanging on permission prompts.
func (e *OpenCodeEngine) generateOpenCodeConfigStep(_ *WorkflowData) GitHubActionStep {
	// Build the config JSON with all permissions set to allow
	configJSON := `{"agent":{"build":{"permissions":{"bash":"allow","edit":"allow","read":"allow","glob":"allow","grep":"allow","write":"allow","webfetch":"allow","websearch":"allow"}}}}`

	// Shell command to write or merge the config with restrictive permissions
	command := fmt.Sprintf(`umask 077
mkdir -p "$GITHUB_WORKSPACE"
CONFIG="$GITHUB_WORKSPACE/opencode.jsonc"
BASE_CONFIG='%s'
if [ -f "$CONFIG" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$CONFIG")" '$existing * $base')
  echo "$MERGED" > "$CONFIG"
else
  echo "$BASE_CONFIG" > "$CONFIG"
fi
chmod 600 "$CONFIG"`, configJSON)

	stepLines := []string{"      - name: Write OpenCode configuration"}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, nil)
	return GitHubActionStep(stepLines)
}
