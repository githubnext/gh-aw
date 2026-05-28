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

func NewClaudeEngine() *ClaudeEngine {
	return &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:           "claude",
			displayName:  "Claude Code",
			description:  "Uses Claude Code with full MCP tool support and allow-listing",
			experimental: false,
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

// GetAPMTarget returns "claude" so that apm-action packs Claude-specific primitives.
func (e *ClaudeEngine) GetAPMTarget() string {
	return "claude"
}

// GetRequiredSecretNames returns the list of secrets required by the Claude engine
// This includes ANTHROPIC_API_KEY and optionally MCP_GATEWAY_API_KEY and mcp-scripts secrets
func (e *ClaudeEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	return append([]string{"ANTHROPIC_API_KEY"}, collectCommonMCPSecrets(workflowData)...)
}

// GetSecretValidationStep returns the secret validation step for the Claude engine.
// Returns an empty step if custom command is specified.
func (e *ClaudeEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"ANTHROPIC_API_KEY"},
		"Claude Code",
		"https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code",
	)
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
		true,  // Include Node.js setup
		true,  // Claude Code requires post-install scripts for native binaries
		false, // Agentic engine installs should not apply npm release-age cooldown
	)
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
	execConfig := e.buildClaudeExecConfig(workflowData, logFile, toolsWithMountedCLIs)
	commandName := e.getClaudeCommandName(workflowData)
	harnessScriptName := e.getClaudeHarnessScriptName(workflowData)
	claudeCommand := e.buildClaudeCommand(workflowData, execConfig, commandName, harnessScriptName)
	command := e.buildClaudeExecutionCommand(workflowData, logFile, claudeCommand)
	env := e.buildClaudeEnv(workflowData)
	step := e.buildClaudeExecutionStep(workflowData, command, env, execConfig.allowedTools)

	return []GitHubActionStep{step}
}

type claudeExecConfig struct {
	args            []string
	allowedTools    string
	mcpConfigArg    string
	modelConfigured bool
}

func (e *ClaudeEngine) buildClaudeExecConfig(workflowData *WorkflowData, logFile string, tools map[string]any) claudeExecConfig {
	args := buildClaudeBaseArgs(logFile)
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	args = appendClaudeMaxTurnsArg(args, workflowData)
	mcpConfigArg := buildClaudeMCPConfigArg(workflowData)
	allowedTools := e.computeAllowedClaudeToolsString(tools, workflowData.SafeOutputs, workflowData.CacheMemoryConfig, workflowData.MCPScripts, workflowData.SandboxConfig)
	if allowedTools != "" {
		args = append(args, "--allowed-tools", allowedTools)
	}
	args = append(args, "--permission-mode", getClaudePermissionMode(workflowData))
	args = append(args, "--output-format", "stream-json")
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Bare {
		claudeLog.Print("Bare mode enabled: adding --bare")
		args = append(args, "--bare")
	}
	args = appendClaudeCustomArgs(args, workflowData)

	return claudeExecConfig{
		args:            args,
		allowedTools:    allowedTools,
		mcpConfigArg:    mcpConfigArg,
		modelConfigured: modelConfigured,
	}
}

func buildClaudeBaseArgs(logFile string) []string {
	return []string{"--print", "--no-chrome", "--debug-file", logFile, "--verbose"}
}

func appendClaudeMaxTurnsArg(args []string, workflowData *WorkflowData) []string {
	if workflowData.EngineConfig == nil || workflowData.EngineConfig.MaxTurns == "" {
		return args
	}
	claudeLog.Printf("Setting max turns: %s", workflowData.EngineConfig.MaxTurns)
	return append(args, "--max-turns", workflowData.EngineConfig.MaxTurns)
}

func buildClaudeMCPConfigArg(workflowData *WorkflowData) string {
	if !HasMCPServers(workflowData) {
		return ""
	}
	claudeLog.Print("Adding MCP configuration")
	return ` --mcp-config "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json"`
}

func getClaudePermissionMode(workflowData *WorkflowData) string {
	permissionMode := "acceptEdits"
	if isEditToolExplicitlyDisabled(workflowData.Tools) {
		claudeLog.Print("tools.edit=false detected: using auto permission mode")
		permissionMode = "auto"
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.PermissionMode != "" {
		permissionMode = workflowData.EngineConfig.PermissionMode
		claudeLog.Printf("Using engine.permission-mode override: %s", permissionMode)
	}
	return permissionMode
}

func appendClaudeCustomArgs(args []string, workflowData *WorkflowData) []string {
	if workflowData.EngineConfig == nil || len(workflowData.EngineConfig.Args) == 0 {
		return args
	}
	engineArgs, permissionModeFromArgs := stripClaudePermissionModeArgs(workflowData.EngineConfig.Args)
	if permissionModeFromArgs != "" && workflowData.EngineConfig.PermissionMode == "" {
		claudeLog.Printf("Using legacy engine.args permission mode override: %s", permissionModeFromArgs)
		args = setClaudePermissionModeArg(args, permissionModeFromArgs)
	}
	return append(args, engineArgs...)
}

func setClaudePermissionModeArg(args []string, permissionMode string) []string {
	updated := append([]string(nil), args...)
	for i := range updated[:len(updated)-1] {
		if updated[i] == "--permission-mode" {
			updated[i+1] = permissionMode
			break
		}
	}
	return updated
}

func (e *ClaudeEngine) getClaudeCommandName(workflowData *WorkflowData) string {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		claudeLog.Printf("Using custom command: %s", workflowData.EngineConfig.Command)
		return workflowData.EngineConfig.Command
	}
	return "claude"
}

func (e *ClaudeEngine) getClaudeHarnessScriptName(workflowData *WorkflowData) string {
	harnessScriptName := e.GetHarnessScriptName()
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.HarnessScript != "" {
		harnessScriptName = workflowData.EngineConfig.HarnessScript
		claudeLog.Printf("Using custom harness script: %s", harnessScriptName)
	}
	return harnessScriptName
}

func (e *ClaudeEngine) buildClaudeCommand(workflowData *WorkflowData, config claudeExecConfig, commandName string, harnessScriptName string) string {
	claudeCommand := buildDirectClaudeCommand(commandName, config.args, config.mcpConfigArg)
	if harnessScriptName != "" {
		claudeCommand = buildHarnessWrappedClaudeCommand(commandName, harnessScriptName, config.args, config.mcpConfigArg)
	}
	if config.modelConfigured {
		return claudeCommand
	}
	return appendClaudeModelFallback(claudeCommand, workflowData)
}

func buildHarnessWrappedClaudeCommand(commandName string, harnessScriptName string, args []string, mcpConfigArg string) string {
	execPrefix := fmt.Sprintf(`%s %s/%s %s`, nodeRuntimeResolutionCommand, SetupActionDestinationShell, harnessScriptName, commandName)
	return fmt.Sprintf("%s %s%s --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt", execPrefix, shellJoinArgs(args), mcpConfigArg)
}

func buildDirectClaudeCommand(commandName string, args []string, mcpConfigArg string) string {
	promptCommand := `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`
	return fmt.Sprintf("%s%s %s", shellJoinArgs(append([]string{commandName}, args...)), mcpConfigArg, promptCommand)
}

func appendClaudeModelFallback(claudeCommand string, workflowData *WorkflowData) string {
	modelEnvVar := constants.EnvVarModelAgentClaude
	if workflowData.SafeOutputs == nil {
		modelEnvVar = constants.EnvVarModelDetectionClaude
	}
	return fmt.Sprintf(`%s${%s:+ --model "$%s"}`, claudeCommand, modelEnvVar, modelEnvVar)
}

func (e *ClaudeEngine) buildClaudeExecutionCommand(workflowData *WorkflowData, logFile string, claudeCommand string) string {
	if isFirewallEnabled(workflowData) {
		return buildFirewallClaudeCommand(workflowData, logFile, claudeCommand)
	}
	return fmt.Sprintf(`set -o pipefail
          printf '%%s' "$(date +%%s%%3N)" > %s
          touch %s
          (umask 177 && touch %s)
          # Execute Claude Code CLI with prompt from file
          %s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, claudeCommand, logFile)
}

func buildFirewallClaudeCommand(workflowData *WorkflowData, logFile string, claudeCommand string) string {
	allowedDomains := getClaudeAllowedDomains(workflowData)
	claudeCommandWithPath := fmt.Sprintf(`%s && %s`, GetNpmBinPathSetup(), claudeCommand)
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
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"ANTHROPIC_API_KEY"}),
	})
}

func getClaudeAllowedDomains(workflowData *WorkflowData) string {
	allowedDomains := workflowData.CachedAllowedDomainsStr
	if !workflowData.CachedAllowedDomainsComputed {
		allowedDomains = GetAllowedDomainsForEngine(constants.ClaudeEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	return allowedDomains
}

func (e *ClaudeEngine) buildClaudeEnv(workflowData *WorkflowData) map[string]string {
	env := buildClaudeBaseEnv()
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	applyClaudePhaseEnv(env, workflowData)
	applyClaudeMCPEnv(env, workflowData)
	applyClaudeTimeoutEnv(env, workflowData)
	applySafeOutputEnvToMap(env, workflowData)
	applyClaudeConfiguredEnv(env, workflowData)
	applyClaudeModelEnv(env, workflowData)
	return env
}

func buildClaudeBaseEnv() map[string]string {
	return map[string]string{
		"ANTHROPIC_API_KEY":             "${{ secrets.ANTHROPIC_API_KEY }}",
		"DISABLE_TELEMETRY":             "1",
		"DISABLE_ERROR_REPORTING":       "1",
		"DISABLE_BUG_COMMAND":           "1",
		"CLAUDE_CODE_DISABLE_FAST_MODE": "1",
		"GH_AW_PROMPT":                  "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_AW":                     "true",
		"GITHUB_STEP_SUMMARY":           AgentStepSummaryPath,
		"GITHUB_WORKSPACE":              "${{ github.workspace }}",
	}
}

func applyClaudePhaseEnv(env map[string]string, workflowData *WorkflowData) {
	env["GH_AW_PHASE"] = "agent"
	if workflowData.IsDetectionRun {
		env["GH_AW_PHASE"] = "detection"
	}
	env["GH_AW_VERSION"] = "dev"
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
	}
	if isFirewallEnabled(workflowData) {
		maps.Copy(env, getGitIdentityEnvVars())
	}
}

func applyClaudeMCPEnv(env map[string]string, workflowData *WorkflowData) {
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ runner.temp }}/gh-aw/mcp-config/mcp-servers.json"
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
	if workflowData.ToolsStartupTimeout != "" {
		env["GH_AW_STARTUP_TIMEOUT"] = workflowData.ToolsStartupTimeout
	}
	if workflowData.ToolsTimeout != "" {
		env["GH_AW_TOOL_TIMEOUT"] = workflowData.ToolsTimeout
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	}
}

func applyClaudeConfiguredEnv(env map[string]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		claudeLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}
	if !IsMCPScriptsEnabled(workflowData.MCPScripts) {
		return
	}
	for varName, secretExpr := range collectMCPScriptsSecrets(workflowData.MCPScripts) {
		if _, exists := env[varName]; !exists {
			env[varName] = secretExpr
		}
	}
}

func applyClaudeModelEnv(env map[string]string, workflowData *WorkflowData) {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		claudeLog.Printf("Setting %s env var for model: %s", constants.ClaudeCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.ClaudeCLIModelEnvVar] = workflowData.EngineConfig.Model
		return
	}
	if workflowData.SafeOutputs == nil {
		env[constants.EnvVarModelDetectionClaude] = compilerenv.BuildModelOverrideExpressionEmptyFallback(constants.EnvVarModelDetectionClaude, compilerenv.DefaultModelClaude)
		return
	}
	env[constants.EnvVarModelAgentClaude] = compilerenv.BuildModelOverrideExpressionEmptyFallback(constants.EnvVarModelAgentClaude, compilerenv.DefaultModelClaude)
}

func (e *ClaudeEngine) buildClaudeExecutionStep(workflowData *WorkflowData, command string, env map[string]string, allowedTools string) GitHubActionStep {
	stepLines := []string{"      - name: Execute Claude Code CLI", "        id: agentic_execution"}
	if allowedToolsComment := e.generateAllowedToolsComment(allowedTools, "        "); allowedToolsComment != "" {
		stepLines = append(stepLines, strings.Split(strings.TrimSuffix(allowedToolsComment, "\n"), "\n")...)
	}
	stepLines = append(stepLines, getClaudeStepTimeout(workflowData))
	filteredEnv := FilterEnvForSecrets(env, e.GetRequiredSecretNames(workflowData))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	return GitHubActionStep(FormatStepWithCommandAndEnv(stepLines, command, filteredEnv))
}

func getClaudeStepTimeout(workflowData *WorkflowData) string {
	if workflowData.TimeoutMinutes != "" {
		timeoutValue := strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout-minutes: ")
		return "        timeout-minutes: " + timeoutValue
	}
	return fmt.Sprintf("        timeout-minutes: %d", int(constants.DefaultAgenticWorkflowTimeout/time.Minute))
}

// GetLogParserScriptId returns the JavaScript script name for parsing Claude logs
func (e *ClaudeEngine) GetLogParserScriptId() string {
	return "parse_claude_log"
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
