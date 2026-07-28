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
	antigravityLog.Printf("Generating execution steps for Antigravity engine: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	command, firewallEnabled, modelConfigured := e.buildAntigravityExecutionCommand(workflowData, logFile)
	env := e.buildAntigravityExecutionEnv(workflowData, firewallEnabled, modelConfigured)
	step := e.buildAntigravityExecutionStep(workflowData, command, env)
	return []GitHubActionStep{
		e.generateAntigravitySettingsStep(workflowData),
		step,
	}
}

func (e *AntigravityEngine) buildAntigravityExecutionCommand(workflowData *WorkflowData, logFile string) (string, bool, bool) {
	agyCommand, modelConfigured := e.buildAntigravityCLICommand(workflowData)
	firewallEnabled := isFirewallEnabled(workflowData)
	if firewallEnabled {
		return e.buildAntigravityAWFCommand(workflowData, logFile, agyCommand), true, modelConfigured
	}
	command := fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, agyCommand, logFile)
	return command, false, modelConfigured
}

func (e *AntigravityEngine) buildAntigravityCLICommand(workflowData *WorkflowData) (string, bool) {
	commandName := "agy"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	modelConfigured := workflowData.Model != ""
	agyCommand := fmt.Sprintf(`%s %s --prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`,
		commandName,
		shellJoinArgs([]string{"--dangerously-skip-permissions"}),
	)
	return getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + agyCommand, modelConfigured
}

func (e *AntigravityEngine) buildAntigravityAWFCommand(workflowData *WorkflowData, logFile, agyCommand string) string {
	agyCommandWithPath := fmt.Sprintf("%s && %s", GetNpmBinPathSetup(), agyCommand)
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		agyCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, agyCommandWithPath)
	}
	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         "antigravity",
		EngineCommand:      agyCommandWithPath,
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            false,
		AllowedDomains:     e.resolveAntigravityAllowedDomains(workflowData),
		PathSetup:          "touch " + AgentStepSummaryPath,
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"ANTIGRAVITY_API_KEY", "GEMINI_API_KEY"}),
	})
}

func (e *AntigravityEngine) resolveAntigravityAllowedDomains(workflowData *WorkflowData) string {
	apiTarget := ""
	if workflowData.EngineConfig != nil {
		apiTarget = workflowData.EngineConfig.APITarget
	}
	if workflowData.CachedAllowedDomainsComputed {
		return mergeAPITargetDomains(workflowData.CachedAllowedDomainsStr, apiTarget)
	}
	allowedDomains := GetAllowedDomainsForEngine(constants.AntigravityEngine,
		workflowData.NetworkPermissions,
		workflowData.Tools,
		workflowData.Runtimes,
	)
	if apiTarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, apiTarget)
	}
	return allowedDomains
}

func (e *AntigravityEngine) buildAntigravityExecutionEnv(workflowData *WorkflowData, firewallEnabled, modelConfigured bool) map[string]string {
	env := buildAntigravityBaseEnv()
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	applyAntigravityPhaseEnv(env, workflowData)
	applyAntigravityMCPAndFirewallEnv(env, workflowData, firewallEnabled)
	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)
	applyAntigravityMaxTurnsEnv(env, workflowData)
	applyAntigravityCustomEnv(env, workflowData, modelConfigured)
	ensureAntigravityGeminiAPIKey(env)
	return env
}

func buildAntigravityBaseEnv() map[string]string {
	return map[string]string{
		"ANTIGRAVITY_API_KEY":             "${{ secrets.ANTIGRAVITY_API_KEY }}",
		"GH_AW_PROMPT":                    constants.AwPromptsFile,
		"GITHUB_AW":                       "true",
		"GITHUB_WORKSPACE":                "${{ github.workspace }}",
		"RUNNER_TEMP":                     "${{ runner.temp }}",
		"GITHUB_STEP_SUMMARY":             AgentStepSummaryPath,
		"DEBUG":                           "antigravity-cli:*",
		"ANTIGRAVITY_CLI_TRUST_WORKSPACE": "true",
	}
}

func applyAntigravityPhaseEnv(env map[string]string, workflowData *WorkflowData) {
	env["GH_AW_PHASE"] = workflowRunPhase(workflowData)
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
		return
	}
	env["GH_AW_VERSION"] = "dev"
}

func applyAntigravityMCPAndFirewallEnv(env map[string]string, workflowData *WorkflowData, firewallEnabled bool) {
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/.antigravity/settings.json"
	}
	if !firewallEnabled {
		return
	}
	env["ANTIGRAVITY_API_BASE_URL"] = fmt.Sprintf("http://host.docker.internal:%d", constants.AntigravityLLMGatewayPort)
	maps.Copy(env, getGitIdentityEnvVars())
}

func applyAntigravityMaxTurnsEnv(env map[string]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
		return
	}
	env["GH_AW_MAX_TURNS"] = compilerenv.BuildDefaultMaxTurnsExpression()
}

func applyAntigravityCustomEnv(env map[string]string, workflowData *WorkflowData, modelConfigured bool) {
	if modelConfigured {
		antigravityLog.Printf("Setting %s env var for model: %s", constants.AntigravityCLIModelEnvVar, workflowData.Model)
		env[constants.AntigravityCLIModelEnvVar] = workflowData.Model
	}
	applyEngineCwdEnv(env, workflowData)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}
	if agentConfig := getAgentConfig(workflowData); agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		antigravityLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}
}

func ensureAntigravityGeminiAPIKey(env map[string]string) {
	if _, hasGeminiKey := env["GEMINI_API_KEY"]; hasGeminiKey {
		return
	}
	env["GEMINI_API_KEY"] = env["ANTIGRAVITY_API_KEY"]
}

func (e *AntigravityEngine) buildAntigravityExecutionStep(workflowData *WorkflowData, command string, env map[string]string) GitHubActionStep {
	stepLines := []string{
		"      - name: Execute Antigravity CLI",
		"        id: agentic_execution",
	}
	allowedSecrets := append([]string{"GEMINI_API_KEY"}, e.GetRequiredSecretNames(workflowData)...)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	return GitHubActionStep(FormatStepWithCommandAndEnv(stepLines, command, filteredEnv))
}
