package workflow

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var claudeLog = logger.New("workflow:claude_engine")

// ClaudeEngine represents the Claude Code agentic engine
type ClaudeEngine struct {
	BaseEngine
}

var _ CodingAgentEngine = (*ClaudeEngine)(nil)

func NewClaudeEngine() *ClaudeEngine {
	return &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:               "claude",
			displayName:      "Claude Code",
			description:      "Uses Claude Code with full MCP tool support and allow-listing",
			experimental:     false,
			ghSkillAgentName: "claude-code",
			capabilities: EngineCapabilities{
				ToolsAllowlist:   true,
				MaxTurns:         true,  // Claude supports max-turns feature
				MaxContinuations: false, // Claude Code does not support --max-autopilot-continues-style continuation
				WebSearch:        true,  // Claude has built-in WebSearch support
				NativeAgentFile:  false, // Claude does not support agent file natively; the compiler prepends the agent file content to prompt.txt
				BareMode:         true,  // Claude CLI supports --bare
			},
			dedicatedLLMGatewayPort: constants.ClaudeLLMGatewayPort,
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Claude Code CLI uses
// for model selection. Setting ANTHROPIC_MODEL is equivalent to passing --model to the CLI.
func (e *ClaudeEngine) GetModelEnvVarName() string {
	return constants.ClaudeCLIModelEnvVar
}

// ResolveLLMProvider returns the effective provider for Claude inference.
// Default is anthropic, overridable via engine.provider (or engine.model-provider).
func (e *ClaudeEngine) ResolveLLMProvider(workflowData *WorkflowData) string {
	return resolveEngineLLMProvider(workflowData, LLMProviderAnthropic)
}

// GetAPMTarget returns "claude" so that apm-action packs Claude-specific primitives.
func (e *ClaudeEngine) GetAPMTarget() string {
	return "claude"
}

// GetRequiredSecretNames returns the list of secrets required by the Claude engine.
// When Anthropic WIF (github-oidc + provider=anthropic) is configured, no static API key
// is needed and only common MCP secrets are returned.
func (e *ClaudeEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	provider := e.ResolveLLMProvider(workflowData)
	if provider == LLMProviderAnthropic && isAnthropicWIF(workflowData) {
		return collectCommonMCPSecrets(workflowData)
	}
	return append(llmProviderSecretNames(provider), collectCommonMCPSecrets(workflowData)...)
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the Claude engine
// supports as defined in the AWF specification.
func (e *ClaudeEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.AnthropicAPIKey,
	}
}

// GetSecretValidationStep returns the secret validation step for the Claude engine.
// Returns an empty step if custom command is specified or if Anthropic WIF is configured.
func (e *ClaudeEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	provider := e.ResolveLLMProvider(workflowData)
	if provider == LLMProviderAnthropic && isAnthropicWIF(workflowData) {
		return GitHubActionStep{}
	}
	providerSecrets := llmProviderSecretNames(provider)
	return BuildDefaultSecretValidationStep(
		workflowData,
		providerSecrets,
		"Claude Code",
		llmProviderDocsURL(provider),
	)
}

// isAnthropicWIF returns true when the workflow is configured to use Anthropic
// Workload Identity Federation (github-oidc auth type with provider=anthropic).
func isAnthropicWIF(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Auth == nil {
		return false
	}
	auth := workflowData.EngineConfig.Auth
	return auth.Type == "github-oidc" && auth.Provider == "anthropic"
}

func (e *ClaudeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	claudeLog.Printf("Generating installation steps for Claude engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		claudeLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Use version from engine config if provided, otherwise default to pinned version
	version := string(constants.DefaultClaudeCodeVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	// Claude Code requires post-install scripts (native binaries) so --ignore-scripts must
	// NOT be passed. This is intentionally different from other engine installs.
	npmSteps := GenerateNpmInstallSteps(
		"@anthropic-ai/claude-code",
		version,
		"Install Claude Code CLI",
		"claude",
		NPMInstallOptions{
			IncludeNodeSetup:  true,
			RunInstallScripts: true,
			CooldownEnabled:   false,
		},
	)
	if isDockerSbxRuntime(workflowData) {
		npmSteps = append(npmSteps, GenerateDockerSbxNpmCLIInstallStep(
			"@anthropic-ai/claude-code",
			version,
			"Install Claude Code CLI in docker-sbx path",
			"claude",
			true,
			false,
		))
	}
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetDeclaredOutputFiles returns the output files that Claude may produce
func (e *ClaudeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetAgentManifestFiles returns Claude-specific instruction files that should be
// treated as security-sensitive manifests.  Modifying these files can change the
// agent's instructions, guidelines, or permissions on the next run.
// CLAUDE.md is the primary per-project instruction file; AGENTS.md is the
// cross-engine convention that Claude Code also reads.
func (e *ClaudeEngine) GetAgentManifestFiles() []string {
	return []string{"CLAUDE.md", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Claude-specific config directory prefixes.
// The .claude/ directory contains settings, custom commands, hooks, and other
// engine configuration that could affect agent behaviour.
func (e *ClaudeEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".claude/"}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Claude
func (e *ClaudeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	claudeLog.Printf("Generating execution steps for Claude engine: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	toolsWithMountedCLIs := withMountedCLIShellCommandsInRestrictedBash(workflowData)
	cliConfig := e.buildClaudeCLIConfig(workflowData, toolsWithMountedCLIs, logFile)
	claudeCommand := e.buildClaudeCommand(workflowData, cliConfig)
	command := e.wrapClaudeCommand(workflowData, claudeCommand, logFile)
	env := e.buildClaudeExecutionEnv(workflowData, cliConfig.modelConfigured)
	step := e.buildClaudeExecutionStep(workflowData, command, env, cliConfig.allowedTools)
	return []GitHubActionStep{step}
}

type claudeCLIConfig struct {
	args            []string
	allowedTools    string
	mcpConfigArg    string
	modelConfigured bool
	commandName     string
	harnessScript   string
}

func (e *ClaudeEngine) buildClaudeCLIConfig(workflowData *WorkflowData, toolsWithMountedCLIs map[string]any, logFile string) claudeCLIConfig {
	config := claudeCLIConfig{
		args:            []string{"--print", "--no-chrome"},
		modelConfigured: workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "",
		commandName:     "claude",
		harnessScript:   e.GetHarnessScriptName(),
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		claudeLog.Printf("Setting max turns: %s", workflowData.EngineConfig.MaxTurns)
		config.args = append(config.args, "--max-turns", workflowData.EngineConfig.MaxTurns)
	}
	if HasMCPServers(workflowData) {
		claudeLog.Print("Adding MCP configuration")
		config.mcpConfigArg = ` --mcp-config "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json"`
	}
	config.allowedTools = e.computeAllowedClaudeToolsString(toolsWithMountedCLIs, workflowData.SafeOutputs, workflowData.CacheMemoryConfig, workflowData.MCPScripts, workflowData.SandboxConfig)
	if config.allowedTools != "" {
		config.args = append(config.args, "--allowed-tools", config.allowedTools)
	}
	config.args = append(config.args, "--debug-file", logFile, "--verbose")
	config.args = appendClaudePermissionModeArgs(config.args, workflowData)
	config.args = append(config.args, "--output-format", "stream-json")
	config.args = appendClaudeBareAndEngineArgs(config.args, workflowData)
	config.commandName = resolveClaudeCommandName(workflowData)
	config.harnessScript = resolveClaudeHarnessScript(e, workflowData)
	return config
}

func appendClaudePermissionModeArgs(args []string, workflowData *WorkflowData) []string {
	permissionMode := "acceptEdits"
	if isEditToolExplicitlyDisabled(workflowData.Tools) {
		claudeLog.Print("tools.edit=false detected: using auto permission mode")
		permissionMode = "auto"
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.PermissionMode != "" {
		permissionMode = workflowData.EngineConfig.PermissionMode
		claudeLog.Printf("Using engine.permission-mode override: %s", permissionMode)
	}
	return append(args, "--permission-mode", permissionMode)
}

func appendClaudeBareAndEngineArgs(args []string, workflowData *WorkflowData) []string {
	permissionModeValueIndex := -1
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--permission-mode" {
			permissionModeValueIndex = i + 1
		}
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Bare {
		claudeLog.Print("Bare mode enabled: adding --bare")
		args = append(args, "--bare")
	}
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		engineArgs, permissionModeFromArgs := stripClaudePermissionModeArgs(workflowData.EngineConfig.Args)
		if permissionModeFromArgs != "" && workflowData.EngineConfig.PermissionMode == "" && permissionModeValueIndex >= 0 {
			claudeLog.Printf("Using legacy engine.args permission mode override: %s", permissionModeFromArgs)
			args[permissionModeValueIndex] = permissionModeFromArgs
		}
		args = append(args, engineArgs...)
	}
	return args
}

func resolveClaudeCommandName(workflowData *WorkflowData) string {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		claudeLog.Printf("Using custom command: %s", workflowData.EngineConfig.Command)
		return workflowData.EngineConfig.Command
	}
	return "claude"
}

func resolveClaudeHarnessScript(e *ClaudeEngine, workflowData *WorkflowData) string {
	harnessScriptName := e.GetHarnessScriptName()
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.HarnessScript != "" {
		harnessScriptName = workflowData.EngineConfig.HarnessScript
		claudeLog.Printf("Using custom harness script: %s", harnessScriptName)
	}
	return harnessScriptName
}

func (e *ClaudeEngine) buildClaudeCommand(workflowData *WorkflowData, cliConfig claudeCLIConfig) string {
	var claudeCommand string
	if cliConfig.harnessScript != "" {
		execPrefix := fmt.Sprintf(`%s %s/%s %s`, nodeRuntimeResolutionCommand, SetupActionDestinationShell, cliConfig.harnessScript, cliConfig.commandName)
		claudeCommand = fmt.Sprintf("%s %s%s --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt", execPrefix, shellJoinArgs(cliConfig.args), cliConfig.mcpConfigArg)
	} else {
		promptCommand := `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`
		claudeCommand = getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + fmt.Sprintf("%s%s %s", shellJoinArgs(append([]string{cliConfig.commandName}, cliConfig.args...)), cliConfig.mcpConfigArg, promptCommand)
	}
	if !cliConfig.modelConfigured {
		modelEnvVar := constants.EnvVarModelAgentClaude
		if workflowData.SafeOutputs == nil {
			modelEnvVar = constants.EnvVarModelDetectionClaude
		}
		claudeCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, claudeCommand, modelEnvVar, modelEnvVar)
	}
	return claudeCommand
}

func (e *ClaudeEngine) wrapClaudeCommand(workflowData *WorkflowData, claudeCommand, logFile string) string {
	if isFirewallEnabled(workflowData) {
		return e.buildAWFClaudeCommand(workflowData, claudeCommand, logFile)
	}
	return fmt.Sprintf(`set -o pipefail
          printf '%%s' "$(date +%%s%%3N)" > %s
          touch %s
          (umask 177 && touch %s)
          # Execute Claude Code CLI with prompt from file
          %s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, claudeCommand, logFile)
}

func (e *ClaudeEngine) buildAWFClaudeCommand(workflowData *WorkflowData, claudeCommand, logFile string) string {
	allowedDomains := allowedClaudeDomains(workflowData)
	npmPathSetup := GetNpmBinPathSetup()
	claudeCommandWithPath := fmt.Sprintf(`%s && %s`, npmPathSetup, claudeCommand)
	if dockerSbxCLIPath := GetDockerSbxNpmCLIPathSetup(workflowData); dockerSbxCLIPath != "" {
		claudeCommandWithPath = fmt.Sprintf("%s && %s", dockerSbxCLIPath, claudeCommandWithPath)
	}
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		claudeCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, claudeCommandWithPath)
	}
	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         "claude",
		EngineCommand:      claudeCommandWithPath,
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            true,
		AllowedDomains:     allowedDomains,
		PathSetup:          "touch " + AgentStepSummaryPath,
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, llmProviderSecretNames(e.ResolveLLMProvider(workflowData))),
	})
}

func allowedClaudeDomains(workflowData *WorkflowData) string {
	var allowedDomains string
	if workflowData.CachedAllowedDomainsComputed {
		allowedDomains = workflowData.CachedAllowedDomainsStr
	} else {
		allowedDomains = GetAllowedDomainsForEngine(constants.ClaudeEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	return allowedDomains
}

func (e *ClaudeEngine) buildClaudeExecutionEnv(workflowData *WorkflowData, modelConfigured bool) map[string]string {
	provider := e.ResolveLLMProvider(workflowData)
	env := baseClaudeExecutionEnv(provider, workflowData)
	if isFirewallEnabled(workflowData) && provider != LLMProviderAnthropic {
		env["ANTHROPIC_BASE_URL"] = llmProviderGatewayBaseURL(provider)
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	applyClaudePhaseAndVersionEnv(env, workflowData)
	applyClaudeMCPAndGitEnv(env, workflowData)
	applyClaudeTimeoutEnv(env, workflowData)
	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)
	applyOptionalEngineToolTimeouts(env, workflowData)
	applyEngineMaxTurnsEnv(env, workflowData)
	applyEngineHarnessRetryEnv(env, workflowData)
	applyClaudeModelEnv(env, workflowData, modelConfigured)
	applyEngineCwdEnv(env, workflowData)
	applyEngineAndAgentEnv(env, workflowData, claudeLog)
	applyMCPScriptsSecretEnv(env, workflowData)
	return env
}

func baseClaudeExecutionEnv(provider string, workflowData *WorkflowData) map[string]string {
	return map[string]string{
		"ANTHROPIC_API_KEY":             llmProviderSecretExpression(provider, workflowData),
		"DISABLE_TELEMETRY":             "1",
		"DISABLE_ERROR_REPORTING":       "1",
		"DISABLE_BUG_COMMAND":           "1",
		"CLAUDE_CODE_DISABLE_FAST_MODE": "1",
		"GH_AW_PROMPT":                  constants.AwPromptsFile,
		"GITHUB_AW":                     "true",
		"GITHUB_STEP_SUMMARY":           AgentStepSummaryPath,
		"GITHUB_WORKSPACE":              "${{ github.workspace }}",
		"RUNNER_TEMP":                   "${{ runner.temp }}",
		"GH_AW_LLM_PROVIDER":            provider,
	}
}

func applyClaudePhaseAndVersionEnv(env map[string]string, workflowData *WorkflowData) {
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
}

func applyClaudeMCPAndGitEnv(env map[string]string, workflowData *WorkflowData) {
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = constants.McpServersJsonPathExpr
	}
	if isFirewallEnabled(workflowData) {
		maps.Copy(env, getGitIdentityEnvVars())
	}
}

func applyClaudeTimeoutEnv(env map[string]string, workflowData *WorkflowData) {
	startupTimeoutMs := int(constants.DefaultMCPStartupTimeout / time.Millisecond)
	if n := templatableIntValue(&workflowData.ToolsStartupTimeout); n > 0 {
		startupTimeoutMs = n * 1000
	}
	timeoutMs := int(constants.DefaultToolTimeout / time.Millisecond)
	if n := templatableIntValue(&workflowData.ToolsTimeout); n > 0 {
		timeoutMs = n * 1000
	}
	env["MCP_TIMEOUT"] = strconv.Itoa(startupTimeoutMs)
	env["MCP_TOOL_TIMEOUT"] = strconv.Itoa(timeoutMs)
	env["BASH_DEFAULT_TIMEOUT_MS"] = strconv.Itoa(timeoutMs)
	env["BASH_MAX_TIMEOUT_MS"] = strconv.Itoa(timeoutMs)
}

func applyClaudeModelEnv(env map[string]string, workflowData *WorkflowData, modelConfigured bool) {
	if modelConfigured {
		claudeLog.Printf("Setting %s env var for model: %s", constants.ClaudeCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.ClaudeCLIModelEnvVar] = workflowData.EngineConfig.Model
		return
	}
	if workflowData.SafeOutputs == nil {
		env[constants.EnvVarModelDetectionClaude] = compilerenv.BuildModelOverrideExpressionEmptyFallback(constants.EnvVarModelDetectionClaude, compilerenv.DefaultModelClaude)
	} else {
		env[constants.EnvVarModelAgentClaude] = compilerenv.BuildModelOverrideExpressionEmptyFallback(constants.EnvVarModelAgentClaude, compilerenv.DefaultModelClaude)
	}
}

func (e *ClaudeEngine) buildClaudeExecutionStep(workflowData *WorkflowData, command string, env map[string]string, allowedTools string) GitHubActionStep {
	stepLines := []string{
		"      - name: Execute Claude Code CLI",
		"        id: agentic_execution",
	}
	if allowedToolsComment := e.generateAllowedToolsComment(allowedTools, "        "); allowedToolsComment != "" {
		commentLines := strings.Split(strings.TrimSuffix(allowedToolsComment, "\n"), "\n")
		stepLines = append(stepLines, commentLines...)
	}
	if workflowData.TimeoutMinutes != "" {
		timeoutValue := strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout-minutes: ")
		stepLines = append(stepLines, "        timeout-minutes: "+timeoutValue)
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", int(constants.DefaultAgenticWorkflowTimeout/time.Minute)))
	}
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	return GitHubActionStep(FormatStepWithCommandAndEnv(stepLines, command, filteredEnv))
}

// GetLogParserScriptId returns the JavaScript script name for parsing Claude logs
func (e *ClaudeEngine) GetLogParserScriptId() string {
	return "parse_claude_log"
}

// GetErrorDetectionScriptId returns the JavaScript script name for detecting
// post-run agent errors from the host runner (including invalid/unsupported model names).
func (e *ClaudeEngine) GetErrorDetectionScriptId() string {
	return "detect_agent_errors"
}

// GetHarnessScriptName returns the filename of the JavaScript harness script that wraps
// the Claude Code CLI with retry logic for transient Anthropic API errors (overload, rate limit).
func (e *ClaudeEngine) GetHarnessScriptName() string {
	return "claude_harness.cjs"
}

// GetSquidLogsSteps returns the steps for uploading and parsing Squid logs (after secret redaction)
func (e *ClaudeEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	return defaultGetSquidLogsSteps(workflowData, claudeLog)
}

func isEditToolExplicitlyDisabled(tools map[string]any) bool {
	if tools == nil {
		return false
	}

	editConfig, hasEdit := tools["edit"]
	if !hasEdit {
		return false
	}

	enabled, isBool := editConfig.(bool)
	return isBool && !enabled
}

// stripClaudePermissionModeArgs removes all --permission-mode flags from args
// (both "--permission-mode <value>" and "--permission-mode=<value>" forms).
// It returns the filtered argument list and the last permission-mode value found.
// The returned permission-mode value is an empty string when no such flag exists.
func stripClaudePermissionModeArgs(args []string) ([]string, string) {
	filtered := make([]string, 0, len(args))
	permissionMode := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--permission-mode":
			if i+1 < len(args) {
				permissionMode = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--permission-mode="):
			permissionMode = strings.TrimPrefix(arg, "--permission-mode=")
		default:
			filtered = append(filtered, arg)
		}
	}

	return filtered, permissionMode
}
