package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var geminiLog = logger.New("workflow:gemini_engine")

// GeminiEngine represents the Google Gemini CLI agentic engine
type GeminiEngine struct {
	BaseEngine
}

var _ CodingAgentEngine = (*GeminiEngine)(nil)

func NewGeminiEngine() *GeminiEngine {
	return &GeminiEngine{
		BaseEngine: BaseEngine{
			id:               "gemini",
			displayName:      "Google Gemini CLI",
			description:      "Google Gemini CLI with headless mode and LLM gateway support",
			experimental:     false,
			ghSkillAgentName: "gemini-cli",
			capabilities: EngineCapabilities{
				ToolsAllowlist:   true,
				MaxTurns:         true,
				MaxContinuations: false, // Gemini CLI does not support --max-autopilot-continues-style continuation mode
				WebSearch:        false,
				NativeAgentFile:  false, // Gemini does not support agent file natively; the compiler prepends the agent file content to prompt.txt
			},
			dedicatedLLMGatewayPort: constants.GeminiLLMGatewayPort,
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Gemini CLI uses
// for model selection. Setting GEMINI_MODEL is equivalent to passing --model to the CLI.
func (e *GeminiEngine) GetModelEnvVarName() string {
	return constants.GeminiCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the Gemini engine
// This includes GEMINI_API_KEY and optionally MCP_GATEWAY_API_KEY, GITHUB_MCP_SERVER_TOKEN,
// HTTP MCP header secrets, and mcp-scripts secrets
func (e *GeminiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	geminiLog.Print("Collecting required secrets for Gemini engine")
	secrets := []string{"GEMINI_API_KEY"}

	// Add common MCP secrets (MCP_GATEWAY_API_KEY if MCP servers present, mcp-scripts secrets)
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)

	// Add GitHub token for GitHub MCP server if present
	if hasGitHubTool(workflowData.ParsedTools) {
		geminiLog.Print("Adding GITHUB_MCP_SERVER_TOKEN secret")
		secrets = append(secrets, "GITHUB_MCP_SERVER_TOKEN")
	}

	// Add HTTP MCP header secret names
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName := range headerSecrets {
		secrets = append(secrets, varName)
	}
	if len(headerSecrets) > 0 {
		geminiLog.Printf("Added %d HTTP MCP header secrets", len(headerSecrets))
	}

	return secrets
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the Gemini engine
// supports as defined in the AWF specification.
func (e *GeminiEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.GeminiAPIKey,
	}
}

// GetSecretValidationStep returns the secret validation step for the Gemini engine.
// Returns an empty step if custom command is specified.
func (e *GeminiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"GEMINI_API_KEY"},
		"Gemini CLI",
		"https://geminicli.com/docs/get-started/authentication/",
	)
}

func (e *GeminiEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	geminiLog.Printf("Generating installation steps for Gemini engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		geminiLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	npmSteps := BuildStandardNpmEngineInstallStepsNoCooldown(
		"@google/gemini-cli",
		string(constants.DefaultGeminiVersion),
		"Install Gemini CLI",
		"gemini",
		workflowData,
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetDeclaredOutputFiles returns the output files that Gemini may produce.
// Gemini CLI writes structured error reports to /tmp/gemini-client-error-*.json
// with a timestamp in the filename (e.g. gemini-client-error-Turn.run-sendMessageStream-2026-02-21T20-45-59-824Z.json).
// These files provide detailed diagnostics when the Gemini API call fails.
// GetPreBundleSteps moves these files into /tmp/gh-aw/ so all artifact paths share a common
// ancestor under /tmp/gh-aw/ and the actions/upload-artifact LCA calculation stays correct.
func (e *GeminiEngine) GetDeclaredOutputFiles() []string {
	return []string{
		constants.TmpGeminiClientErrorGlob,
	}
}

// GetAgentManifestFiles returns Gemini-specific instruction files that should be
// treated as security-sensitive manifests.  A fork PR that modifies these files
// can redirect the agent's behaviour or expand which files it treats as instructions.
// GEMINI.md is the primary per-project context file; AGENTS.md is the cross-engine
// convention that Gemini CLI also reads.
func (e *GeminiEngine) GetAgentManifestFiles() []string {
	return []string{"GEMINI.md", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Gemini-specific config directory prefixes.
// The .gemini/ directory contains settings.json and other configuration that could
// expand which files are treated as instructions or alter agent behaviour.
// Protecting this directory prevents fork PRs from injecting malicious configuration.
func (e *GeminiEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".gemini/"}
}

// GetPreBundleSteps returns a step that moves Gemini CLI error reports from /tmp/ into
// /tmp/gh-aw/ before the unified artifact upload. This keeps all artifact paths under
// /tmp/gh-aw/ so that actions/upload-artifact computes the correct least-common-ancestor
// path and downstream jobs find files at the expected locations.
func (e *GeminiEngine) GetPreBundleSteps(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{
		{
			"      - name: Move Gemini error files to artifact directory",
			"        if: always()",
			"        run: mv /tmp/gemini-client-error-*.json /tmp/gh-aw/ 2>/dev/null || true",
		},
	}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Gemini
func (e *GeminiEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	firewallEnabled := isFirewallEnabled(workflowData)
	geminiLog.Printf("Generating execution steps for Gemini engine: workflow=%s, firewall=%v", workflowData.Name, firewallEnabled)

	modelConfigured := workflowData.Model != ""
	steps := e.buildGeminiSetupPhase(workflowData)
	command := buildGeminiWrappedCommand(workflowData, logFile, buildGeminiCommand(workflowData), firewallEnabled)
	env := buildGeminiExecutionEnv(workflowData, modelConfigured, firewallEnabled)
	steps = append(steps, e.buildGeminiExecutionStep(workflowData, command, env))
	return steps
}

func (e *GeminiEngine) buildGeminiSetupPhase(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{e.generateGeminiSettingsStep(workflowData)}
}

func buildGeminiArgs() []string {
	return []string{"--yolo", "--skip-trust", "--output-format", "stream-json"}
}

func geminiCommandName(workflowData *WorkflowData) string {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		return workflowData.EngineConfig.Command
	}
	return "gemini"
}

func buildGeminiCommand(workflowData *WorkflowData) string {
	command := fmt.Sprintf(`%s %s --prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`,
		geminiCommandName(workflowData),
		shellJoinArgs(buildGeminiArgs()),
	)
	return getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + command
}

func buildGeminiWrappedCommand(workflowData *WorkflowData, logFile, geminiCommand string, firewallEnabled bool) string {
	if firewallEnabled {
		return buildGeminiAWFCommand(workflowData, logFile, geminiCommand)
	}
	return fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, geminiCommand, logFile)
}

func buildGeminiAWFCommand(workflowData *WorkflowData, logFile, geminiCommand string) string {
	allowedDomains := workflowData.CachedAllowedDomainsStr
	if !workflowData.CachedAllowedDomainsComputed {
		allowedDomains = GetAllowedDomainsForEngine(constants.GeminiEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	engineCommand := fmt.Sprintf("%s && %s", GetNpmBinPathSetup(), geminiCommand)
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		engineCommand = fmt.Sprintf("%s && %s", mcpCLIPath, engineCommand)
	}
	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         "gemini",
		EngineCommand:      engineCommand,
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            false,
		AllowedDomains:     allowedDomains,
		PathSetup:          "touch " + AgentStepSummaryPath,
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"GEMINI_API_KEY"}),
	})
}

func buildGeminiExecutionEnv(workflowData *WorkflowData, modelConfigured, firewallEnabled bool) map[string]string {
	env := map[string]string{
		"GEMINI_API_KEY":             "${{ secrets.GEMINI_API_KEY }}",
		"GH_AW_PROMPT":               constants.AwPromptsFile,
		"GITHUB_AW":                  "true",
		"GITHUB_WORKSPACE":           "${{ github.workspace }}",
		"RUNNER_TEMP":                "${{ runner.temp }}",
		"GITHUB_STEP_SUMMARY":        AgentStepSummaryPath,
		"DEBUG":                      "gemini-cli:*",
		"GEMINI_CLI_TRUST_WORKSPACE": "true",
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	env["GH_AW_PHASE"] = workflowRunPhase(workflowData)
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
	} else {
		env["GH_AW_VERSION"] = "dev"
	}
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/.gemini/settings.json"
	}
	if firewallEnabled {
		env["GEMINI_API_BASE_URL"] = fmt.Sprintf("http://host.docker.internal:%d", constants.GeminiLLMGatewayPort)
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
		geminiLog.Printf("Setting %s env var for model: %s", constants.GeminiCLIModelEnvVar, workflowData.Model)
		env[constants.GeminiCLIModelEnvVar] = workflowData.Model
	}
	applyEngineCwdEnv(env, workflowData)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}
	applyGeminiAgentEnv(env, workflowData)
	return env
}

func applyGeminiAgentEnv(env map[string]string, workflowData *WorkflowData) {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		geminiLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}
}

func (e *GeminiEngine) buildGeminiExecutionStep(workflowData *WorkflowData, command string, env map[string]string) GitHubActionStep {
	stepLines := []string{
		"      - name: Execute Gemini CLI",
		"        id: agentic_execution",
	}
	filteredEnv := FilterEnvForSecrets(env, e.GetRequiredSecretNames(workflowData))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)
	return GitHubActionStep(stepLines)
}
