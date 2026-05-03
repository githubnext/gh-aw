package workflow

import (
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var piLog = logger.New("workflow:pi_engine")

// PiEngine represents the Pi AI coding agent (experimental).
// Pi is a provider-agnostic agentic coding assistant that communicates via stdin/stdout
// and emits a streaming JSONL log for structured event capture.  When engine.model uses
// provider/model format (e.g. "copilot/claude-sonnet-4-20250514"), Pi borrows the
// matching engine's AWF configuration (secrets, gateway port, allowed domains) so the
// firewall can route LLM traffic through the correct sidecar port.  Without a provider
// prefix Pi defaults to the Copilot gateway.
//
// Requirements:
//   - tools.github.mode: gh-proxy must be enabled (pre-authenticated gh CLI).
//   - tools.cli-proxy: true must be enabled (MCP servers mounted as CLI tools).
//
// Both requirements are validated at compile time by validatePiEngineRequirements.
type PiEngine struct {
	BaseEngine
}

// NewPiEngine creates and returns a new PiEngine instance.
func NewPiEngine() *PiEngine {
	return &PiEngine{
		BaseEngine: BaseEngine{
			id:                       "pi",
			displayName:              "Pi",
			description:              "Pi AI coding agent (experimental)",
			experimental:             true,
			supportsToolsAllowlist:   true,
			supportsMaxTurns:         false,
			supportsMaxContinuations: false,
			supportsWebSearch:        false,
			supportsNativeAgentFile:  false,
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Pi CLI uses
// for model selection. Setting PI_MODEL is equivalent to passing --model to the CLI.
func (e *PiEngine) GetModelEnvVarName() string {
	return constants.PiCLIModelEnvVar
}

// resolvePiBackend extracts the provider prefix from the engine model (if any) and maps
// it to the matching UniversalLLMBackend.  A model without a slash (e.g. "claude-sonnet-4")
// defaults to the Copilot backend for backward compatibility.
func resolvePiBackend(workflowData *WorkflowData) UniversalLLMBackend {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Model == "" {
		return UniversalLLMBackendCopilot
	}
	model := workflowData.EngineConfig.Model
	if !strings.Contains(model, "/") {
		// No provider prefix — default to Copilot (backward compatibility).
		return UniversalLLMBackendCopilot
	}
	backend, err := resolveUniversalLLMBackendFromModel(model)
	if err != nil {
		piLog.Printf("Could not resolve backend for Pi model %q, defaulting to copilot: %v", model, err)
		return UniversalLLMBackendCopilot
	}
	return backend
}

// GetRequiredSecretNames returns the list of secrets required by the Pi engine.
// When the model uses provider/model format the provider-specific secret is required
// (e.g. ANTHROPIC_API_KEY for "anthropic/..."); otherwise Pi routes through the
// Copilot LLM gateway and reuses COPILOT_GITHUB_TOKEN.
func (e *PiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	piLog.Print("Collecting required secrets for Pi engine")
	backend := resolvePiBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData))
	secrets := append([]string{}, profile.coreSecretNames...)
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)
	return secrets
}

// GetSecretValidationStep returns the secret validation step for the Pi engine.
// The validated secret depends on the resolved provider backend.
func (e *PiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	backend := resolvePiBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData))
	if len(profile.coreSecretNames) == 0 {
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(
		workflowData,
		profile.coreSecretNames,
		"Pi",
		"https://github.github.com/gh-aw/reference/engines/#pi",
	)
}

// GetInstallationSteps returns the GitHub Actions steps needed to install the Pi CLI.
// If engine.extensions is configured, additional `pi install <extension>` steps are emitted
// after the main CLI install step.
func (e *PiEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	piLog.Printf("Generating installation steps for Pi engine: workflow=%s", workflowData.Name)

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		piLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	version := string(constants.DefaultPiVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	npmSteps := BuildStandardNpmEngineInstallSteps(
		"@pi/cli",
		version,
		"Install Pi CLI",
		"pi",
		workflowData,
	)

	steps := BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)

	// Install extensions declared in engine.extensions: [...]
	// Each extension is installed via `pi install <extension>` before the agent runs.
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Extensions) > 0 {
		commandName := "pi"
		if workflowData.EngineConfig.Command != "" {
			commandName = workflowData.EngineConfig.Command
		}

		for _, ext := range workflowData.EngineConfig.Extensions {
			installCmd := fmt.Sprintf("%s install %s", commandName, shellEscapeArg(ext))
			stepLines := []string{
				"      - name: Install Pi extension " + ext,
			}
			stepLines = FormatStepWithCommandAndEnv(stepLines, installCmd, nil)
			steps = append(steps, GitHubActionStep(stepLines))
		}
		piLog.Printf("Added %d Pi extension install steps", len(workflowData.EngineConfig.Extensions))
	}

	return steps
}

// GetDeclaredOutputFiles returns the output files that Pi may produce.
// The streaming JSONL log is the primary artifact for post-run analysis.
func (e *PiEngine) GetDeclaredOutputFiles() []string {
	return []string{
		PiStreamingLogFile,
	}
}

// GetLogParserScriptId returns the script ID for parsing Pi logs.
func (e *PiEngine) GetLogParserScriptId() string {
	return "parse_pi_log"
}

// GetLogFileForParsing returns the Pi streaming log file path used by the JS log parser.
func (e *PiEngine) GetLogFileForParsing() string {
	return PiStreamingLogFile
}

// GetAgentManifestFiles returns Pi-specific instruction files treated as
// security-sensitive manifests.
func (e *PiEngine) GetAgentManifestFiles() []string {
	return []string{"PI.md", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Pi-specific config directory prefixes.
func (e *PiEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".pi/"}
}

// GetExecutionSteps returns the GitHub Actions steps for executing the Pi CLI.
// The prompt is piped to Pi via stdin; streaming JSON events are written to
// PiStreamingLogFile for post-run analysis and step summary rendering.
func (e *PiEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	piLog.Printf("Generating execution steps for Pi engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	commandName := "pi"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}

	// Build the pi run command. Prompt is piped via stdin.
	piArgs := []string{"run", "--json-log", PiStreamingLogFile}

	// Append any user-supplied extra args from engine.args
	if workflowData.EngineConfig != nil {
		piArgs = append(piArgs, workflowData.EngineConfig.Args...)
	}

	// The prompt is piped from a file via stdin substitution.
	// Two extensions are automatically loaded (in order):
	//   1. pi_provider.cjs  — calls /reflect to discover the open LLM inference paths
	//   2. pi_steering_extension.cjs — injects time-pressure steering messages
	// Pi CLI supports multiple --extension flags; built-in extensions load after any
	// user-specified extensions (via engine.args) so the built-in behaviour wins.
	// ${RUNNER_TEMP} is a Linux shell variable expanded by bash at runtime; gh-aw
	// container environments are Linux-only so this is safe across all runners.
	piCommand := fmt.Sprintf(
		`cat /tmp/gh-aw/aw-prompts/prompt.txt | %s %s --extension "${RUNNER_TEMP}/gh-aw/actions/pi_provider.cjs" --extension "${RUNNER_TEMP}/gh-aw/actions/pi_steering_extension.cjs"`,
		commandName, shellJoinArgs(piArgs))

	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Resolve backend based on the model provider prefix.
	backend := resolvePiBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData))

	var command string
	firewallEnabled := isFirewallEnabled(workflowData)
	if firewallEnabled {
		// Get allowed domains: prefer the pre-warmed cache on WorkflowData to avoid
		// re-running the expensive map+sort operation.
		var allowedDomains string
		if workflowData.CachedAllowedDomainsComputed {
			allowedDomains = workflowData.CachedAllowedDomainsStr
		} else {
			model := ""
			if modelConfigured {
				model = workflowData.EngineConfig.Model
			}
			// The model was validated before reaching here; a malformed model (leading slash)
			// must never occur at this point — panic is the correct invariant guard.
			var err error
			allowedDomains, err = GetPiAllowedDomainsWithModel(model, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
			if err != nil {
				panic(fmt.Sprintf("BUG: invalid Pi model %q reached domain computation (should have been caught by validation): %v", model, err))
			}
		}
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
		}

		npmPathSetup := GetNpmBinPathSetup()
		piCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, piCommand)
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			piCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, piCommandWithPath)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:         "pi",
			EngineCommand:      piCommandWithPath,
			LogFile:            logFile,
			WorkflowData:       workflowData,
			UsesTTY:            false,
			AllowedDomains:     allowedDomains,
			PathSetup:          "touch " + AgentStepSummaryPath,
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, profile.coreSecretNames),
		})
	} else {
		command = fmt.Sprintf(`set -o pipefail
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentStepSummaryPath, logFile, piCommand, logFile)
	}

	// Build the environment map.  Provider-specific credentials and base URL are
	// injected via the backend profile so Pi connects to the correct LLM gateway.
	env := map[string]string{
		"GH_AW_PROMPT":        "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_AW":           "true",
		"GITHUB_WORKSPACE":    "${{ github.workspace }}",
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
	}

	// Inject provider-specific credentials and, when the firewall is enabled,
	// the gateway base URL so Pi routes LLM traffic through the correct sidecar port.
	maps.Copy(env, profile.env)
	if firewallEnabled {
		piLog.Printf("Setting %s to Pi LLM gateway port %d", profile.baseURLEnvName, profile.gatewayPort)
		env[profile.baseURLEnvName] = fmt.Sprintf("http://host.docker.internal:%d", profile.gatewayPort)
	}

	if workflowData.IsDetectionRun {
		env["GH_AW_PHASE"] = "detection"
	} else {
		env["GH_AW_PHASE"] = "agent"
	}
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
	} else {
		env["GH_AW_VERSION"] = "dev"
	}

	// When the AWF firewall is enabled, set git identity environment variables
	// for commit authorship.
	if firewallEnabled {
		maps.Copy(env, getGitIdentityEnvVars())
	}

	// Apply native model env var only when explicitly configured.
	if modelConfigured {
		piLog.Printf("Setting %s env var for model: %s", constants.PiCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.PiCLIModelEnvVar] = workflowData.EngineConfig.Model
	}

	// Apply safe-outputs env
	applySafeOutputEnvToMap(env, workflowData)

	// Apply custom env overrides from engine.env
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Apply custom env from agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		piLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}

	stepLines := []string{
		"      - name: Execute Pi CLI",
		"        id: agentic_execution",
	}

	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}

// PiStreamingLogFile is the path where Pi CLI writes its streaming JSONL event log.
// All Pi tool calls, messages, and metrics are captured here for post-run analysis
// and step summary rendering.
const PiStreamingLogFile = "/tmp/gh-aw/pi-streaming.jsonl"
