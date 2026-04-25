package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var piLog = logger.New("workflow:pi_engine")

// PiEngine represents the Pi coding agent agentic engine.
// Pi is a minimal, provider-agnostic terminal coding harness with broad BYOK
// (Bring Your Own Key) support. It uses the same provider/model format as
// OpenCode and Crush (e.g., "anthropic/claude-sonnet-4") to determine the
// provider and route requests through the appropriate API.
type PiEngine struct {
	UniversalLLMConsumerEngine
}

func NewPiEngine() *PiEngine {
	return &PiEngine{
		UniversalLLMConsumerEngine: UniversalLLMConsumerEngine{
			BaseEngine: BaseEngine{
				id:                     "pi",
				displayName:            "Pi",
				description:            "Pi coding agent with print mode and multi-provider LLM support",
				experimental:           true,  // Start as experimental until smoke tests pass consistently
				supportsToolsAllowlist: false, // Pi manages its own tool permissions
				supportsMaxTurns:       false, // No --max-turns flag in pi
				supportsWebSearch:      false, // Has built-in tools but not exposed via gh-aw neutral tools yet
				llmGatewayPort:         constants.PiLLMGatewayPort,
			},
		},
	}
}

// SupportsLLMGateway returns the LLM gateway port for Pi engine
func (e *PiEngine) SupportsLLMGateway() int {
	return constants.PiLLMGatewayPort
}

// GetModelEnvVarName returns empty string because pi uses the --model CLI flag
// rather than a native environment variable for model selection.
func (e *PiEngine) GetModelEnvVarName() string {
	return ""
}

// GetRequiredSecretNames returns the list of secrets required by the Pi engine.
// By default, Pi routes through the Copilot API using COPILOT_GITHUB_TOKEN
// (or ${{ github.token }} when copilot-requests feature is enabled).
// Additional provider API keys can be added via engine.env overrides.
func (e *PiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	piLog.Print("Collecting required secrets for Pi engine")
	return e.GetUniversalRequiredSecretNames(workflowData)
}

// GetInstallationSteps returns the GitHub Actions steps needed to install Pi CLI
func (e *PiEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	piLog.Printf("Generating installation steps for Pi engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		piLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	npmSteps := BuildStandardNpmEngineInstallSteps(
		"@mariozechner/pi-coding-agent",
		string(constants.DefaultPiVersion),
		"Install Pi coding agent",
		"pi",
		workflowData,
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetSecretValidationStep returns the secret validation step for the Pi engine.
// Returns an empty step if copilot-requests feature is enabled (uses GitHub Actions token).
func (e *PiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return e.GetUniversalSecretValidationStep(
		workflowData,
		"Pi coding agent",
		"https://github.github.com/gh-aw/reference/engines/#pi",
	)
}

// GetAgentManifestFiles returns Pi-specific instruction files that should be
// treated as security-sensitive manifests. Pi reads AGENTS.md (and CLAUDE.md)
// at startup from the current directory and parent directories.
func (e *PiEngine) GetAgentManifestFiles() []string {
	return []string{"AGENTS.md", "CLAUDE.md"}
}

// GetAgentManifestPathPrefixes returns Pi-specific config directory prefixes
// that must be protected from fork PR injection.
// The .pi/ directory contains agent configuration, settings, skills, and
// extensions that could alter agent behaviour.
func (e *PiEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".pi/"}
}

// GetDeclaredOutputFiles returns the output files that Pi may produce.
func (e *PiEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Pi
func (e *PiEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	piLog.Printf("Generating execution steps for Pi engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Build CLI arguments
	var piArgs []string

	// Use print mode (non-interactive, print response and exit)
	piArgs = append(piArgs, "--print")

	// Disable session saving in CI to avoid accumulating session files
	piArgs = append(piArgs, "--no-session")

	// Pass model via CLI flag if configured (Pi uses provider/model format)
	if modelConfigured {
		piArgs = append(piArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Prompt from file (positional argument).
	// Keep this outside shellJoinArgs so command substitution expands at runtime.
	promptArg := "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\""

	// Build command name
	commandName := "pi"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	piCommand := fmt.Sprintf("%s %s %s", commandName, shellJoinArgs(piArgs), promptArg)

	// AWF wrapping
	firewallEnabled := isFirewallEnabled(workflowData)
	var command string
	if firewallEnabled {
		// Resolve model for provider-specific domain allowlisting
		model := ""
		if modelConfigured {
			model = workflowData.EngineConfig.Model
		}
		allowedDomains := GetPiAllowedDomainsWithToolsAndRuntimes(
			model,
			workflowData.NetworkPermissions,
			workflowData.Tools,
			workflowData.Runtimes,
		)

		npmPathSetup := GetNpmBinPathSetup()
		piCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, piCommand)
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			piCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, piCommandWithPath)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "pi",
			EngineCommand:  piCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			AllowedDomains: allowedDomains,
		})
	} else {
		command = fmt.Sprintf("set -o pipefail\n%s 2>&1 | tee -a %s", piCommand, logFile)
	}

	env := map[string]string{
		"GH_AW_PROMPT":     "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
		"NO_PROXY":         "localhost,127.0.0.1",
	}
	e.ApplyUniversalProviderEnv(env, workflowData, firewallEnabled)

	// Safe outputs env
	applySafeOutputEnvToMap(env, workflowData)

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
		"      - name: Execute Pi coding agent",
		"        id: agentic_execution",
	}
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}
