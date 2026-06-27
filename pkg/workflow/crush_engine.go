package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var crushLog = logger.New("workflow:crush_engine")

// CrushEngine represents the Crush CLI agentic engine.
// Crush is a provider-agnostic, open-source AI coding agent with broader BYOK
// (Bring Your Own Key) support, but gh-aw currently supports a subset of
// providers for engine.model validation: copilot, anthropic, openai, and codex.
type CrushEngine struct {
	UniversalLLMConsumerEngine
}

func NewCrushEngine() *CrushEngine {
	return &CrushEngine{
		UniversalLLMConsumerEngine: UniversalLLMConsumerEngine{
			BaseEngine: BaseEngine{
				id:           "crush",
				displayName:  "Crush",
				description:  "Crush CLI with headless mode and multi-provider LLM support",
				experimental: true, // Start as experimental until smoke tests pass consistently
				capabilities: EngineCapabilities{
					ToolsAllowlist: false, // Crush manages its own tool permissions via .crush.json
					MaxTurns:       true,  // AWF max-turns is supported for Crush runs
					WebSearch:      false, // Has built-in websearch but not exposed via gh-aw neutral tools yet
				},
			},
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Crush CLI uses
// for model selection. Setting CRUSH_MODEL is equivalent to passing --model to the CLI.
func (e *CrushEngine) GetModelEnvVarName() string {
	return constants.CrushCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the Crush engine.
// By default, Crush routes through the Copilot API using COPILOT_GITHUB_TOKEN
// (or ${{ github.token }} when permissions.copilot-requests is set to write).
// Additional provider API keys can be added via engine.env overrides.
func (e *CrushEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	crushLog.Print("Collecting required secrets for Crush engine")
	return e.GetUniversalRequiredSecretNames(workflowData)
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the Crush engine
// supports as defined in the AWF specification. Crush is a multi-provider engine so all
// provider API keys are valid engine.env overrides.
func (e *CrushEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.CopilotGitHubToken,
		constants.AnthropicAPIKey,
		constants.CodexAPIKey,
		constants.OpenAIAPIKey,
	}
}

// GetInstallationSteps returns the GitHub Actions steps needed to install Crush CLI
func (e *CrushEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	crushLog.Printf("Generating installation steps for Crush engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		crushLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Use version from engine config if provided, otherwise default to pinned version
	version := string(constants.DefaultCrushVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	// Crush requires post-install scripts (native binaries) so --ignore-scripts must
	// NOT be passed. This is intentionally different from other engine installs.
	npmSteps := GenerateNpmInstallSteps(
		"@charmland/crush",
		version,
		"Install Crush CLI",
		"crush",
		true, // Include Node.js setup
		true, // Crush requires post-install scripts for native binaries
		resolveRuntimeCooldown(workflowData, "node"),
	)

	// Run crush --version to verify the installation and force any deferred binary downloads
	commandName := "crush"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	versionStep := GitHubActionStep{
		"      - name: Verify Crush CLI installation",
		"        run: " + commandName + " --version",
	}
	npmSteps = append(npmSteps, versionStep)

	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetSecretValidationStep returns the secret validation step for the Crush engine.
// Returns an empty step if permissions.copilot-requests is write (uses GitHub Actions token).
func (e *CrushEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return e.GetUniversalSecretValidationStep(
		workflowData,
		"Crush CLI",
		"https://github.github.com/gh-aw/reference/engines/#crush",
	)
}

// GetAgentManifestFiles returns Crush-specific instruction files that should be
// treated as security-sensitive manifests. Modifying these files can change the
// agent's instructions, permissions, or configuration on the next run.
// .crush.json is the primary Crush config file; AGENTS.md is the cross-engine
// convention that Crush also reads.
func (e *CrushEngine) GetAgentManifestFiles() []string {
	return []string{".crush.json", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Crush-specific config directory prefixes
// that must be protected from fork PR injection.
// The .crush/ directory contains agent configuration, instructions, and other
// settings that could alter agent behaviour.
func (e *CrushEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".crush/"}
}

// GetDeclaredOutputFiles returns the output files that Crush may produce.
func (e *CrushEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Crush
func (e *CrushEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	crushLog.Printf("Generating execution steps for Crush engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	return e.BuildCLIEngineExecutionSteps(workflowData, logFile, UniversalCLIEngineExecutionConfig{
		EngineConstant:     constants.CrushEngine,
		DefaultCommandName: "crush",
		ExtraCLIArgs:       []string{"--verbose"},
		MCPConfigFile:      ".crush.json",
		StepName:           "Execute Crush CLI",
		ConfigStep:         e.generateCrushConfigStep(workflowData),
		ModelEnvVarName:    constants.CrushCLIModelEnvVar,
		WriteTimestamp:     true,
	})
}

// generateCrushConfigStep writes .crush.json with all permissions set to allow
// to prevent CI hanging on permission prompts.
func (e *CrushEngine) generateCrushConfigStep(_ *WorkflowData) GitHubActionStep {
	// Build the config JSON with all permissions set to allow
	// OpenCode/Crush uses "permission" (singular) — "permissions" (plural) is silently ignored.
	// "external_directory" must be "allow" in non-interactive CI mode (defaults to "ask" → implicit deny).
	configJSON := `{"agent":{"build":{"permission":{"bash":"allow","edit":"allow","read":"allow","glob":"allow","grep":"allow","write":"allow","webfetch":"allow","websearch":"allow","external_directory":"allow"}}}}`

	// Shell command to write or merge the config with restrictive permissions
	command := fmt.Sprintf(`umask 077
mkdir -p "$GITHUB_WORKSPACE"
CONFIG="$GITHUB_WORKSPACE/.crush.json"
BASE_CONFIG='%s'
if [ -f "$CONFIG" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$CONFIG")" '$existing * $base')
  echo "$MERGED" > "$CONFIG"
else
  echo "$BASE_CONFIG" > "$CONFIG"
fi
chmod 600 "$CONFIG"`, configJSON)

	stepLines := []string{"      - name: Write Crush Config"}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, nil)
	return GitHubActionStep(stepLines)
}
