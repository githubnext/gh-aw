package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
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
			id:           "pi",
			displayName:  "Pi",
			description:  "Pi AI coding agent (experimental)",
			experimental: true,
			capabilities: EngineCapabilities{
				ToolsAllowlist:   true,
				MaxTurns:         true,
				MaxContinuations: false,
				WebSearch:        false,
				NativeAgentFile:  false,
			},
		},
	}
}

// GetModelEnvVarName returns the legacy Pi model env-var name exposed by gh-aw.
// gh-aw passes the model to the Pi CLI via --model and separately exports the
// original workflow model for extensions.
func (e *PiEngine) GetModelEnvVarName() string {
	return constants.PiCLIModelEnvVar
}

// resolvePiBackend extracts the provider prefix from the engine model (if any) and maps
// it to the matching UniversalLLMBackend.  A model without a slash (e.g. "claude-sonnet-4")
// defaults to the Copilot backend for backward compatibility.
//
// "github-copilot/" is accepted as an alias for "copilot/" since that is the
// provider name used by Pi CLI's built-in model registry.
func resolvePiBackend(workflowData *WorkflowData) UniversalLLMBackend {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Model == "" {
		return UniversalLLMBackendCopilot
	}
	model := workflowData.EngineConfig.Model
	if !strings.Contains(model, "/") {
		// No provider prefix — default to Copilot (backward compatibility).
		return UniversalLLMBackendCopilot
	}
	// "github-copilot" is Pi CLI's internal name for GitHub Copilot.  Accept it as
	// an alias so workflows can use either "copilot/..." or "github-copilot/...".
	parts := strings.SplitN(model, "/", 2)
	if strings.EqualFold(parts[0], "github-copilot") {
		return UniversalLLMBackendCopilot
	}
	backend, err := resolveUniversalLLMBackendFromModel(model)
	if err != nil {
		piLog.Printf("Could not resolve backend for Pi model %q, defaulting to copilot: %v", model, err)
		return UniversalLLMBackendCopilot
	}
	return backend
}

// extractPiModelID returns the model ID portion of a provider/model string.
// For "copilot/claude-sonnet-4" it returns "claude-sonnet-4".
// For a bare model name (no slash) the whole string is returned unchanged.
func extractPiModelID(model string) string {
	if _, after, found := strings.Cut(model, "/"); found {
		return after
	}
	return model
}

// piNativeProviderName maps an AWF UniversalLLMBackend to the corresponding
// Pi CLI built-in provider name.  Used when there is no AWF gateway to proxy
// through (firewall disabled) so Pi can call the provider's API directly.
func piNativeProviderName(backend UniversalLLMBackend) string {
	switch backend {
	case UniversalLLMBackendAnthropic:
		return "anthropic"
	case UniversalLLMBackendCodex:
		return "openai"
	default:
		return "github-copilot"
	}
}

// buildPiModelsJSON returns a minimal Pi models.json payload that registers a
// single custom provider named "aw-gateway" pointing at the AWF LLM gateway
// sidecar.  Pi's resolveConfigValue() resolves the "apiKey" value by looking
// up process.env[apiKey], so passing the secret env-var name (e.g.
// "COPILOT_GITHUB_TOKEN") causes Pi to automatically use the value that is
// already present in the container environment.
//
// The baseUrl uses the "api-proxy" Docker service hostname (not host.docker.internal)
// so that Pi can reach the sidecar container within the AWF Docker network.
// host.docker.internal points to the Docker host (runner), not the api-proxy
// container, and is only available when --enable-host-access is set.
//
// All dynamic values are marshaled via encoding/json to prevent JSON injection.
func buildPiModelsJSON(gatewayPort int, secretEnvVarName, modelID string) string {
	payload := map[string]any{
		"providers": map[string]any{
			"aw-gateway": map[string]any{
				"baseUrl": fmt.Sprintf("http://api-proxy:%d", gatewayPort),
				"api":     "openai-completions",
				"apiKey":  secretEnvVarName,
				"models":  []map[string]any{{"id": modelID}},
			},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		// json.Marshal only fails for non-serialisable types; our map is always
		// serialisable, so this branch is unreachable in practice.
		panic(fmt.Sprintf("BUG: buildPiModelsJSON failed to marshal JSON: %v", err))
	}
	return string(b)
}

// GetRequiredSecretNames returns the list of secrets required by the Pi engine.
// When the model uses provider/model format the provider-specific secret is required
// (e.g. ANTHROPIC_API_KEY for "anthropic/..."); otherwise Pi routes through the
// Copilot LLM gateway and reuses COPILOT_GITHUB_TOKEN.
func (e *PiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	piLog.Print("Collecting required secrets for Pi engine")
	backend := resolvePiBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, hasCopilotRequestsWritePermission(workflowData))
	secrets := append([]string{}, profile.coreSecretNames...)
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)
	return secrets
}

// GetSecretValidationStep returns the secret validation step for the Pi engine.
// The validated secret depends on the resolved provider backend.
func (e *PiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	backend := resolvePiBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, hasCopilotRequestsWritePermission(workflowData))
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

	npmSteps := GenerateNpmInstallSteps(
		"@earendil-works/pi-coding-agent",
		version,
		"Install Pi CLI",
		"pi",
		true,  // Include Node.js setup
		false, // Engine CLI installs must never run install scripts
		false, // Pi installs should not apply the default npm release-age cooldown
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

	// Resolve backend and profile early so we can use them when building piArgs.
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
	backend := resolvePiBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, hasCopilotRequestsWritePermission(workflowData))
	firewallEnabled := isFirewallEnabled(workflowData)

	// Build the pi command.  Pi v0.72+ uses flags-only syntax (no "run" subcommand).
	// --print: non-interactive, process prompt from stdin and exit.
	// --mode json: emit structured JSONL events to stdout.
	// --no-session: don't persist a session file (appropriate for CI).
	piArgs := []string{"--print", "--mode", "json", "--no-session"}

	// Append any user-supplied extra args from engine.args
	if workflowData.EngineConfig != nil {
		piArgs = append(piArgs, filterPiArgs(workflowData.EngineConfig.Args)...)
	}

	// Pi v0.72+ does not support a PI_MODEL env var for CLI model selection; the model must be passed as
	// the --model CLI flag.  When the firewall is enabled we route LLM traffic
	// through the AWF gateway sidecar by generating a temporary models.json that
	// registers a custom "aw-gateway" provider pointing at the gateway port.  When
	// the firewall is disabled we use Pi's built-in provider directly.
	var piModelsJSONSetup string // shell fragment prepended to piCommand when needed
	if modelConfigured {
		modelID := extractPiModelID(workflowData.EngineConfig.Model)
		if firewallEnabled && len(profile.coreSecretNames) > 0 {
			// Firewall case: write a models.json that redirects Pi's LLM calls to the
			// AWF gateway sidecar port.  The "apiKey" field value is the name of the env
			// var that holds the secret; Pi's resolveConfigValue() looks up
			// process.env[apiKey] to obtain the actual token value at runtime.
			//
			// printf '%s\n' '<json>' is safe here because JSON uses only double quotes
			// (never single quotes), so single-quoting via shellEscapeArg requires no
			// further escaping in practice.
			modelsJSON := buildPiModelsJSON(profile.gatewayPort, profile.coreSecretNames[0], modelID)
			piModelsJSONSetup = fmt.Sprintf(
				`mkdir -p /tmp/gh-aw/pi-agent-dir && printf '%%s\n' %s > /tmp/gh-aw/pi-agent-dir/models.json && `,
				shellEscapeArg(modelsJSON))
			piArgs = append(piArgs, "--model", "aw-gateway/"+modelID)
			piLog.Printf("Pi: using models.json gateway routing for model %q via aw-gateway (port %d)", modelID, profile.gatewayPort)
		} else {
			// No firewall: use Pi's built-in provider so it can reach the real LLM API.
			nativeProvider := piNativeProviderName(backend)
			piArgs = append(piArgs, "--model", nativeProvider+"/"+modelID)
			piLog.Printf("Pi: using native provider %q for model %q (no firewall)", nativeProvider, modelID)
		}
	}

	// The prompt is piped from a file via stdin substitution.
	// Two extensions are automatically loaded (in order):
	//   1. pi_provider.cjs  — calls /reflect to discover the open LLM inference paths
	//   2. pi_steering_extension.cjs — injects time-pressure steering messages
	// Pi CLI supports multiple --extension flags; built-in extensions load after any
	// user-specified extensions (via engine.args) so the built-in behaviour wins.
	// ${RUNNER_TEMP} is a Linux shell variable expanded by bash at runtime; gh-aw
	// container environments are Linux-only so this is safe across all runners.
	// stdout (JSONL) and stderr are both piped through tee so that PiStreamingLogFile
	// captures all structured events while agent-stdio.log captures the same output.
	piCommand := fmt.Sprintf(
		`cat /tmp/gh-aw/aw-prompts/prompt.txt | %s %s --extension "${RUNNER_TEMP}/gh-aw/actions/pi_provider.cjs" --extension "${RUNNER_TEMP}/gh-aw/actions/pi_steering_extension.cjs" 2>&1 | tee %s`,
		commandName, shellJoinArgs(piArgs), PiStreamingLogFile)

	// Prepend models.json generation when the gateway-routing approach is used.
	if piModelsJSONSetup != "" {
		piCommand = piModelsJSONSetup + piCommand
	}

	var command string
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
			allowedDomains = mustGetAllowedDomainsForEngineWithModel(constants.PiEngine, model, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
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
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, piCommand, logFile)
	}

	// Build the environment map.  Provider-specific credentials are injected via
	// the backend profile.  The base URL env var (e.g. GITHUB_COPILOT_BASE_URL) is
	// NOT set for Pi because Pi v0.72+ does not read provider-specific base URL env
	// vars; routing is instead handled through models.json (firewall case) or by Pi's
	// native provider (no-firewall case).
	env := map[string]string{
		"GH_AW_PROMPT":        constants.AwPromptsFile,
		"GITHUB_AW":           "true",
		"GITHUB_WORKSPACE":    "${{ github.workspace }}",
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
		"RUNNER_TEMP":         "${{ runner.temp }}",
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	if modelConfigured {
		env["GH_AW_PI_MODEL"] = workflowData.EngineConfig.Model
	}

	// Inject provider-specific credentials from the backend profile.
	maps.Copy(env, profile.env)

	// Pi CLI reads OPENAI_API_KEY and routes traffic to api.openai.com when the env
	// var is present, bypassing the github-copilot provider and the AWF gateway.
	// For the Copilot backend Pi authenticates via COPILOT_GITHUB_TOKEN directly
	// (either through the native github-copilot provider or via models.json apiKey
	// resolution), so OPENAI_API_KEY must not be exposed in the container env.
	if backend == UniversalLLMBackendCopilot {
		delete(env, "OPENAI_API_KEY")
	}

	// When the models.json gateway approach is used, tell Pi where to find it.
	if piModelsJSONSetup != "" {
		env["PI_CODING_AGENT_DIR"] = constants.TmpPiAgentDir
		piLog.Printf("Pi: setting PI_CODING_AGENT_DIR for models.json gateway config")
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
	// for commit authorship and signal to the Pi extension that the AWF api-proxy
	// sidecar is running so the /reflect preflight is not skipped.
	if firewallEnabled {
		maps.Copy(env, getGitIdentityEnvVars())
		env["AWF_REFLECT_ENABLED"] = "1"
	}

	// Apply safe-outputs env
	applySafeOutputEnvToMap(env, workflowData)

	// Propagate W3C trace context so engine spans nest under the gh-aw.agent.setup span.
	applyTraceContextEnvToMap(env)

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	} else {
		env["GH_AW_MAX_TURNS"] = compilerenv.BuildDefaultMaxTurnsExpression()
	}

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

// filterPiArgs removes redundant Pi CLI flags that gh-aw should not pass through.
// Pi runs in yolo mode by default, so explicit --yolo flags are ignored while all
// other engine args are preserved in order.
func filterPiArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--yolo" || strings.HasPrefix(arg, "--yolo=") {
			piLog.Printf("Pi: dropping redundant arg %q because Pi runs in yolo mode by default", arg)
			continue
		}
		filtered = append(filtered, arg)
	}

	return filtered
}
