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
	geminiLog.Printf("Generating execution steps for Gemini engine: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Write .gemini/settings.json with context.includeDirectories and tools.core.
	// This step runs after the MCP gateway setup (which may have written mcpServers config)
	// and merges the context/tools settings into any existing settings.json.
	settingsStep := e.generateGeminiSettingsStep(workflowData)
	steps = append(steps, settingsStep)

	// Build gemini CLI arguments based on configuration
	modelConfigured := geminiModelConfigured(workflowData)
	geminiCommand := buildGeminiCommand(workflowData)

	// Build the full command with AWF wrapping if enabled
	firewallEnabled := isFirewallEnabled(workflowData)
	command := buildGeminiExecutionCommand(workflowData, geminiCommand, logFile, firewallEnabled)

	// Build environment variables
	env := e.buildGeminiEnv(workflowData, firewallEnabled, modelConfigured)

	// Generate the execution step
	stepLines := []string{
		"      - name: Execute Gemini CLI",
		"        id: agentic_execution",
	}

	// Filter environment variables for security
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Inject GH_TOKEN for CLI proxy (added after filtering since it uses a special
	// fallback expression that is always allowed when cli-proxy is enabled)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)

	// Format step with command and env
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))
	return steps
}

func geminiModelConfigured(workflowData *WorkflowData) bool {
	return workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
}

func buildGeminiCommand(workflowData *WorkflowData) string {
	geminiArgs := []string{"--yolo", "--skip-trust", "--output-format", "stream-json"}
	commandName := "gemini"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	geminiCommand := fmt.Sprintf(`%s %s --prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, commandName, shellJoinArgs(geminiArgs))
	return getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + geminiCommand
}

func buildGeminiExecutionCommand(workflowData *WorkflowData, geminiCommand string, logFile string, firewallEnabled bool) string {
	if firewallEnabled {
		return BuildAWFCommand(AWFCommandConfig{
			EngineName: "gemini", EngineCommand: geminiCommandWithPath(workflowData, geminiCommand),
			LogFile: logFile, WorkflowData: workflowData, UsesTTY: false,
			AllowedDomains:     geminiAllowedDomains(workflowData),
			PathSetup:          "touch " + AgentStepSummaryPath,
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"GEMINI_API_KEY"}),
		})
	}
	return fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, geminiCommand, logFile)
}

func geminiCommandWithPath(workflowData *WorkflowData, geminiCommand string) string {
	geminiCommandWithPath := fmt.Sprintf("%s && %s", GetNpmBinPathSetup(), geminiCommand)
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		geminiCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, geminiCommandWithPath)
	}
	return geminiCommandWithPath
}

func geminiAllowedDomains(workflowData *WorkflowData) string {
	var allowedDomains string
	if workflowData.CachedAllowedDomainsComputed {
		allowedDomains = workflowData.CachedAllowedDomainsStr
	} else {
		allowedDomains = GetAllowedDomainsForEngine(constants.GeminiEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	return allowedDomains
}

func (e *GeminiEngine) buildGeminiEnv(workflowData *WorkflowData, firewallEnabled bool, modelConfigured bool) map[string]string {
	env := baseGeminiEnv(workflowData)
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	applyGeminiOptionalEnv(env, workflowData, firewallEnabled, modelConfigured)
	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)
	applyEngineCwdEnv(env, workflowData)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		geminiLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}
	return env
}

func baseGeminiEnv(workflowData *WorkflowData) map[string]string {
	env := map[string]string{
		"GEMINI_API_KEY": "${{ secrets.GEMINI_API_KEY }}", "GH_AW_PROMPT": constants.AwPromptsFile,
		"GITHUB_AW": "true", "GITHUB_WORKSPACE": "${{ github.workspace }}", "RUNNER_TEMP": "${{ runner.temp }}",
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath, "DEBUG": "gemini-cli:*", "GEMINI_CLI_TRUST_WORKSPACE": "true",
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
	return env
}

func applyGeminiOptionalEnv(env map[string]string, workflowData *WorkflowData, firewallEnabled bool, modelConfigured bool) {
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/.gemini/settings.json"
	}
	if firewallEnabled {
		env["GEMINI_API_BASE_URL"] = fmt.Sprintf("http://host.docker.internal:%d", constants.GeminiLLMGatewayPort)
		maps.Copy(env, getGitIdentityEnvVars())
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	} else {
		env["GH_AW_MAX_TURNS"] = compilerenv.BuildDefaultMaxTurnsExpression()
	}
	if modelConfigured {
		geminiLog.Printf("Setting %s env var for model: %s", constants.GeminiCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.GeminiCLIModelEnvVar] = workflowData.EngineConfig.Model
	}
}
