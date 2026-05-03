package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var piLog = logger.New("workflow:pi_engine")

// PiEngine represents the Pi AI coding agent (experimental).
// Pi is an agentic coding assistant that communicates via stdin/stdout and
// emits a streaming JSONL log for structured event capture.
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

// GetRequiredSecretNames returns the list of secrets required by the Pi engine.
// Pi routes through the Copilot LLM gateway and reuses COPILOT_GITHUB_TOKEN
// rather than a dedicated PI_API_KEY.
func (e *PiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	piLog.Print("Collecting required secrets for Pi engine")
	secrets := []string{"COPILOT_GITHUB_TOKEN"}
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)
	return secrets
}

// GetSecretValidationStep returns the secret validation step for the Pi engine.
// Pi reuses COPILOT_GITHUB_TOKEN (no dedicated PI_API_KEY).
func (e *PiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"COPILOT_GITHUB_TOKEN"},
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
	// The built-in steering extension is automatically loaded so that every Pi session
	// receives time-pressure steering messages without requiring workflow configuration.
	// Pi CLI supports multiple --extension flags; user-specified extensions (via engine.args)
	// are appended before this flag so the built-in extension loads last, consistent with the
	// aw-harness spec's "built-in extensions after user extensions" ordering.
	// ${RUNNER_TEMP} is a Linux shell variable expanded by bash at runtime; gh-aw container
	// environments are Linux-only so this is safe across all supported runner configurations.
	piCommand := fmt.Sprintf(
		`cat /tmp/gh-aw/aw-prompts/prompt.txt | %s %s --extension "${RUNNER_TEMP}/gh-aw/actions/pi_steering_extension.cjs"`,
		commandName, shellJoinArgs(piArgs))

	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	var command string
	firewallEnabled := isFirewallEnabled(workflowData)
	if firewallEnabled {
		allowedDomains := GetPiAllowedDomains(workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
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
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"COPILOT_GITHUB_TOKEN"}),
		})
	} else {
		command = fmt.Sprintf(`set -o pipefail
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentStepSummaryPath, logFile, piCommand, logFile)
	}

	// #nosec G101 -- This is NOT a hardcoded credential. It is a GitHub Actions expression
	// template that the runtime replaces with the actual secret value.
	env := map[string]string{
		"COPILOT_GITHUB_TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}",
		"GH_AW_PROMPT":         "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_AW":            "true",
		"GITHUB_WORKSPACE":     "${{ github.workspace }}",
		"GITHUB_STEP_SUMMARY":  AgentStepSummaryPath,
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
	// for commit authorship. Pi uses the copilot/claude/codex LLM gateway ports
	// directly (no dedicated Pi gateway port).
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
