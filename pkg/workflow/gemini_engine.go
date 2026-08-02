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
				ToolsAllowlist:       true,
				MaxTurns:             true,
				MaxContinuations:     false, // Gemini CLI does not support --max-autopilot-continues-style continuation mode
				WebSearch:            false,
				NativeAgentFile:      false, // Gemini does not support agent file natively; the compiler prepends the agent file content to prompt.txt
				BashCommandAllowlist: true,  // Gemini enforces tools.bash allowlist via tools.core: [run_shell_command(cmd)]
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
// HTTP MCP header secrets, and mcp-scripts secrets.
// When Google/Vertex WIF (github-oidc + provider=google) is configured, no static API key
// is needed and only common MCP secrets are returned.
func (e *GeminiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	geminiLog.Print("Collecting required secrets for Gemini engine")

	var secrets []string
	if !isGeminiVertexWIF(workflowData) {
		secrets = append(secrets, "GEMINI_API_KEY")
	}

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
// Returns an empty step if custom command is specified or if Google/Vertex WIF is configured.
func (e *GeminiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	if isGeminiVertexWIF(workflowData) {
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"GEMINI_API_KEY"},
		"Gemini CLI",
		"https://geminicli.com/docs/get-started/authentication/",
	)
}

// isGeminiVertexWIF returns true when the workflow is configured to use Google
// Workload Identity Federation (github-oidc auth type with provider=gcp) and
// has the required fields set (workload-identity-provider, service-account, project).
func isGeminiVertexWIF(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Auth == nil {
		return false
	}
	auth := workflowData.EngineConfig.Auth
	return auth.Type == "github-oidc" && auth.Provider == "gcp" &&
		auth.GoogleWorkloadIdentityProvider != "" &&
		auth.GoogleServiceAccount != "" &&
		auth.GoogleProject != ""
}

func (e *GeminiEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	geminiLog.Printf("Generating installation steps for Gemini engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		geminiLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Normalize engine config version when not explicitly set, so downstream consumers
	// (e.g. execution steps) observe the effective installed version.
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version == "" {
		workflowData.EngineConfig.Version = string(constants.DefaultGeminiVersion)
		geminiLog.Printf("No engine.version specified, using default Gemini CLI version: %s", workflowData.EngineConfig.Version)
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
	geminiLog.Printf("Generating execution steps for Gemini engine: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))
	settingsStep := e.generateGeminiSettingsStep(workflowData)
	modelConfigured := workflowData.Model != ""
	firewallEnabled := isFirewallEnabled(workflowData)
	vertexWIF := isGeminiVertexWIF(workflowData)
	geminiCommand := e.buildGeminiCLICommand(workflowData)
	command := e.buildGeminiExecutionCommand(workflowData, logFile, geminiCommand, firewallEnabled)
	env := e.buildGeminiExecutionEnv(workflowData, firewallEnabled, vertexWIF, modelConfigured)
	step := e.buildGeminiExecutionStep(workflowData, command, env)
	return []GitHubActionStep{settingsStep, step}
}

func (e *GeminiEngine) buildGeminiCLICommand(workflowData *WorkflowData) string {
	geminiArgs := []string{"--yolo", "--skip-trust", "--output-format", "stream-json"}
	commandName := "gemini"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	geminiCommand := fmt.Sprintf(`%s %s --prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, commandName, shellJoinArgs(geminiArgs))
	return getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + geminiCommand
}

func (e *GeminiEngine) buildGeminiExecutionCommand(workflowData *WorkflowData, logFile, geminiCommand string, firewallEnabled bool) string {
	if !firewallEnabled {
		return fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, geminiCommand, logFile)
	}

	geminiCommandWithPath := fmt.Sprintf("%s && %s", GetNpmBinPathSetup(), geminiCommand)
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		geminiCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, geminiCommandWithPath)
	}
	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         "gemini",
		EngineCommand:      geminiCommandWithPath,
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            false,
		AllowedDomains:     e.geminiAllowedDomains(workflowData),
		PathSetup:          "touch " + AgentStepSummaryPath,
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, e.GetRequiredSecretNames(workflowData)),
	})
}

func (e *GeminiEngine) geminiAllowedDomains(workflowData *WorkflowData) string {
	allowedDomains := workflowData.CachedAllowedDomainsStr
	if !workflowData.CachedAllowedDomainsComputed {
		allowedDomains = GetAllowedDomainsForEngine(constants.GeminiEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	return allowedDomains
}

func (e *GeminiEngine) buildGeminiExecutionEnv(workflowData *WorkflowData, firewallEnabled, vertexWIF, modelConfigured bool) map[string]string {
	env := e.baseGeminiExecutionEnv(workflowData, vertexWIF)
	e.applyGeminiRuntimeEnv(env, workflowData, firewallEnabled, modelConfigured)
	applyEngineCwdEnv(env, workflowData)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}
	e.applyGeminiAgentEnv(env, workflowData)
	if vertexWIF {
		applyGeminiVertexWIFEnv(env, workflowData.EngineConfig.Auth)
	}
	return env
}

func (e *GeminiEngine) baseGeminiExecutionEnv(workflowData *WorkflowData, vertexWIF bool) map[string]string {
	env := map[string]string{
		"DEBUG":                      "gemini-cli:*",
		"GEMINI_CLI_TRUST_WORKSPACE": "true",
		"GH_AW_PROMPT":               constants.AwPromptsFile,
		"GITHUB_AW":                  "true",
		"GITHUB_STEP_SUMMARY":        AgentStepSummaryPath,
		"GITHUB_WORKSPACE":           "${{ github.workspace }}",
		"RUNNER_TEMP":                "${{ runner.temp }}",
	}
	if !vertexWIF {
		env["GEMINI_API_KEY"] = "${{ secrets.GEMINI_API_KEY }}"
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	return env
}

func (e *GeminiEngine) applyGeminiRuntimeEnv(env map[string]string, workflowData *WorkflowData, firewallEnabled, modelConfigured bool) {
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
}

func (e *GeminiEngine) applyGeminiAgentEnv(env map[string]string, workflowData *WorkflowData) {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig == nil || len(agentConfig.Env) == 0 {
		return
	}
	maps.Copy(env, agentConfig.Env)
	geminiLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
}

func applyGeminiVertexWIFEnv(env map[string]string, auth *EngineAuthConfig) {
	location := auth.GoogleLocation
	if location == "" {
		location = "us-central1"
	}
	env["GOOGLE_CLOUD_LOCATION"] = location
	env["GOOGLE_CLOUD_PROJECT"] = auth.GoogleProject
	env["GOOGLE_GENAI_USE_VERTEXAI"] = "true"
}

func (e *GeminiEngine) buildGeminiExecutionStep(workflowData *WorkflowData, command string, env map[string]string) GitHubActionStep {
	stepLines := []string{
		"      - name: Execute Gemini CLI",
		"        id: agentic_execution",
		"        timeout-minutes: " + resolveStepTimeoutValue(workflowData),
	}
	filteredEnv := FilterEnvForSecrets(env, e.GetRequiredSecretNames(workflowData))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	return GitHubActionStep(FormatStepWithCommandAndEnv(stepLines, command, filteredEnv))
}
