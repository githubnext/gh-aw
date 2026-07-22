package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var antigravityLog = logger.New("workflow:antigravity_engine")

// AntigravityEngine represents the Google Antigravity CLI agentic engine
type AntigravityEngine struct {
	BaseEngine
}

var _ CodingAgentEngine = (*AntigravityEngine)(nil)

func NewAntigravityEngine() *AntigravityEngine {
	return &AntigravityEngine{
		BaseEngine: BaseEngine{
			id:               "antigravity",
			displayName:      "Antigravity CLI",
			description:      "Antigravity CLI with headless mode and LLM gateway support",
			experimental:     true,
			ghSkillAgentName: "antigravity",
			capabilities: EngineCapabilities{
				ToolsAllowlist:   true,
				MaxTurns:         true,
				MaxContinuations: false, // Antigravity CLI does not support --max-autopilot-continues-style continuation mode
				WebSearch:        false,
				NativeAgentFile:  false, // Antigravity does not support agent file natively; the compiler prepends the agent file content to prompt.txt
			},
			dedicatedLLMGatewayPort: constants.AntigravityLLMGatewayPort,
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Antigravity CLI uses
// for model selection. Setting ANTIGRAVITY_MODEL is equivalent to passing --model to the CLI.
func (e *AntigravityEngine) GetModelEnvVarName() string {
	return constants.AntigravityCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the Antigravity engine
// This includes ANTIGRAVITY_API_KEY and optionally MCP_GATEWAY_API_KEY, GITHUB_MCP_SERVER_TOKEN,
// HTTP MCP header secrets, and mcp-scripts secrets
func (e *AntigravityEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	antigravityLog.Print("Collecting required secrets for Antigravity engine")
	secrets := []string{"ANTIGRAVITY_API_KEY"}

	// Add common MCP secrets (MCP_GATEWAY_API_KEY if MCP servers present, mcp-scripts secrets)
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)

	// Add GitHub token for GitHub MCP server if present
	if hasGitHubTool(workflowData.ParsedTools) {
		antigravityLog.Print("Adding GITHUB_MCP_SERVER_TOKEN secret")
		secrets = append(secrets, "GITHUB_MCP_SERVER_TOKEN")
	}

	// Add HTTP MCP header secret names
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName := range headerSecrets {
		secrets = append(secrets, varName)
	}
	if len(headerSecrets) > 0 {
		antigravityLog.Printf("Added %d HTTP MCP header secrets", len(headerSecrets))
	}

	return secrets
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the Antigravity engine
// supports as defined in the AWF specification.
func (e *AntigravityEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.AntigravityAPIKey,
	}
}

// GetSecretValidationStep returns the secret validation step for the Antigravity engine.
// Returns an empty step if custom command is specified.
func (e *AntigravityEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"ANTIGRAVITY_API_KEY"},
		"Antigravity CLI",
		"https://antigravity.google/docs/cli-overview",
	)
}

func (e *AntigravityEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	antigravityLog.Printf("Generating installation steps for Antigravity engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		antigravityLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	version := string(constants.DefaultAntigravityVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}
	installSteps := GenerateAntigravityInstallerSteps(version, "Install Antigravity CLI", false)
	return BuildNpmEngineInstallStepsWithAWF(installSteps, workflowData)
}

// GetDeclaredOutputFiles returns the output files that Antigravity may produce.
// Antigravity CLI writes structured error reports to /tmp/antigravity-client-error-*.json
// with a timestamp in the filename (e.g. antigravity-client-error-Turn.run-sendMessageStream-2026-02-21T20-45-59-824Z.json).
// These files provide detailed diagnostics when the Antigravity API call fails.
// GetPreBundleSteps moves these files into /tmp/gh-aw/ so all artifact paths share a common
// ancestor under /tmp/gh-aw/ and the actions/upload-artifact LCA calculation stays correct.
func (e *AntigravityEngine) GetDeclaredOutputFiles() []string {
	return []string{
		constants.TmpAntigravityClientErrorGlob,
	}
}

// GetAgentManifestFiles returns Antigravity-specific instruction files that should be
// treated as security-sensitive manifests.  A fork PR that modifies these files
// can redirect the agent's behaviour or expand which files it treats as instructions.
// ANTIGRAVITY.md is the primary per-project context file; AGENTS.md is the cross-engine
// convention that Antigravity CLI also reads.
func (e *AntigravityEngine) GetAgentManifestFiles() []string {
	return []string{"ANTIGRAVITY.md", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Antigravity-specific config directory prefixes.
// The .antigravity/ directory contains settings.json and other configuration that could
// expand which files are treated as instructions or alter agent behaviour.
// Protecting this directory prevents fork PRs from injecting malicious configuration.
func (e *AntigravityEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".antigravity/"}
}

// GetPreBundleSteps returns a step that moves Antigravity CLI error reports from /tmp/ into
// /tmp/gh-aw/ before the unified artifact upload. This keeps all artifact paths under
// /tmp/gh-aw/ so that actions/upload-artifact computes the correct least-common-ancestor
// path and downstream jobs find files at the expected locations.
func (e *AntigravityEngine) GetPreBundleSteps(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{
		{
			"      - name: Move Antigravity error files to artifact directory",
			"        if: always()",
			"        run: mv /tmp/antigravity-client-error-*.json /tmp/gh-aw/ 2>/dev/null || true",
		},
	}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Antigravity
func (e *AntigravityEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	firewallEnabled := isFirewallEnabled(workflowData)
	antigravityLog.Printf("Generating execution steps for Antigravity engine: workflow=%s, firewall=%v", workflowData.Name, firewallEnabled)

	modelConfigured := workflowData.Model != ""
	steps := e.buildAntigravitySetupPhase(workflowData)
	command := buildAntigravityWrappedCommand(workflowData, logFile, buildAntigravityCommand(workflowData), firewallEnabled)
	env := buildAntigravityExecutionEnv(workflowData, modelConfigured, firewallEnabled)
	steps = append(steps, e.buildAntigravityExecutionStep(workflowData, command, env))
	return steps
}

func (e *AntigravityEngine) buildAntigravitySetupPhase(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{e.generateAntigravitySettingsStep(workflowData)}
}

func buildAntigravityArgs() []string {
	return []string{"--dangerously-skip-permissions"}
}

func antigravityCommandName(workflowData *WorkflowData) string {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		return workflowData.EngineConfig.Command
	}
	return "agy"
}

func buildAntigravityCommand(workflowData *WorkflowData) string {
	command := fmt.Sprintf(`%s %s --prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`,
		antigravityCommandName(workflowData),
		shellJoinArgs(buildAntigravityArgs()),
	)
	return getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + command
}

func buildAntigravityWrappedCommand(workflowData *WorkflowData, logFile, agyCommand string, firewallEnabled bool) string {
	if firewallEnabled {
		return buildAntigravityAWFCommand(workflowData, logFile, agyCommand)
	}
	return fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, agyCommand, logFile)
}

func buildAntigravityAWFCommand(workflowData *WorkflowData, logFile, agyCommand string) string {
	allowedDomains := workflowData.CachedAllowedDomainsStr
	if !workflowData.CachedAllowedDomainsComputed {
		allowedDomains = GetAllowedDomainsForEngine(constants.AntigravityEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	engineCommand := fmt.Sprintf("%s && %s", GetNpmBinPathSetup(), agyCommand)
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		engineCommand = fmt.Sprintf("%s && %s", mcpCLIPath, engineCommand)
	}
	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         "antigravity",
		EngineCommand:      engineCommand,
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            false,
		AllowedDomains:     allowedDomains,
		PathSetup:          "touch " + AgentStepSummaryPath,
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"ANTIGRAVITY_API_KEY", "GEMINI_API_KEY"}),
	})
}

func buildAntigravityExecutionEnv(workflowData *WorkflowData, modelConfigured, firewallEnabled bool) map[string]string {
	env := map[string]string{
		"ANTIGRAVITY_API_KEY":             "${{ secrets.ANTIGRAVITY_API_KEY }}",
		"GH_AW_PROMPT":                    constants.AwPromptsFile,
		"GITHUB_AW":                       "true",
		"GITHUB_WORKSPACE":                "${{ github.workspace }}",
		"RUNNER_TEMP":                     "${{ runner.temp }}",
		"GITHUB_STEP_SUMMARY":             AgentStepSummaryPath,
		"DEBUG":                           "antigravity-cli:*",
		"ANTIGRAVITY_CLI_TRUST_WORKSPACE": "true",
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	env["GH_AW_PHASE"] = workflowRunPhase(workflowData)
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
	} else {
		env["GH_AW_VERSION"] = "dev"
	}
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/.antigravity/settings.json"
	}
	if firewallEnabled {
		env["ANTIGRAVITY_API_BASE_URL"] = fmt.Sprintf("http://host.docker.internal:%d", constants.AntigravityLLMGatewayPort)
		maps.Copy(env, getGitIdentityEnvVars())
	}
	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	} else {
		env["GH_AW_MAX_TURNS"] = compilerenv.BuildDefaultMaxTurnsExpression()
	}
	if modelConfigured {
		antigravityLog.Printf("Setting %s env var for model: %s", constants.AntigravityCLIModelEnvVar, workflowData.Model)
		env[constants.AntigravityCLIModelEnvVar] = workflowData.Model
	}
	applyEngineCwdEnv(env, workflowData)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}
	applyAntigravityAgentEnv(env, workflowData)
	if _, hasGeminiKey := env["GEMINI_API_KEY"]; !hasGeminiKey {
		env["GEMINI_API_KEY"] = env["ANTIGRAVITY_API_KEY"]
	}
	return env
}

func applyAntigravityAgentEnv(env map[string]string, workflowData *WorkflowData) {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		antigravityLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}
}

func (e *AntigravityEngine) buildAntigravityExecutionStep(workflowData *WorkflowData, command string, env map[string]string) GitHubActionStep {
	stepLines := []string{
		"      - name: Execute Antigravity CLI",
		"        id: agentic_execution",
	}
	allowedSecrets := append([]string{"GEMINI_API_KEY"}, e.GetRequiredSecretNames(workflowData)...)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)
	return GitHubActionStep(stepLines)
}
