package workflow

import (
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var codexEngineLog = logger.New("workflow:codex_engine")

// detectionResponseSchema is the JSON Schema for Codex detection runs.
// It constrains the model output to exactly the threat detection result fields.
// The schema is written to detectionSchemaFilePath before Codex runs and passed
// via --output-schema; the structured result is written to detectionResultFilePath
// via --output-last-message for direct parsing without log scraping.
const detectionResponseSchema = `{"type":"object","properties":{"prompt_injection":{"type":"boolean"},"secret_leak":{"type":"boolean"},"malicious_patch":{"type":"boolean"},"reasons":{"type":"array","items":{"type":"string"}}},"required":["prompt_injection","secret_leak","malicious_patch","reasons"],"additionalProperties":false}`

// detectionSchemaFilePath is the path where the detection JSON schema is written
// before Codex runs. It is referenced by --output-schema.
const detectionSchemaFilePath = "/tmp/gh-aw/threat-detection/detection_schema.json"

// detectionResultFilePath is the path where Codex writes the final structured
// verdict via --output-last-message. The parser reads this file directly instead
// of scraping the log stream, eliminating false parse_error warnings from noisy
// SSE/tracing output.
const detectionResultFilePath = "/tmp/gh-aw/threat-detection/detection_result.json"

// Pre-compiled regexes for Codex log parsing (performance optimization)
var (
	codexToolCallOldFormat    = regexp.MustCompile(`\] tool ([^(]+)\(`)
	codexToolCallNewFormat    = regexp.MustCompile(`^tool ([^(]+)\(`)
	codexExecCommandOldFormat = regexp.MustCompile(`\] exec (.+?) in`)
	codexExecCommandNewFormat = regexp.MustCompile(`^exec (.+?) in`)
	codexDurationPattern      = regexp.MustCompile(`in\s+(\d+(?:\.\d+)?)\s*s`)
	codexTokenUsagePattern    = regexp.MustCompile(`(?i)tokens\s+used[:\s]+(\d+)`)
	codexTotalTokensPattern   = regexp.MustCompile(`total_tokens:\s*(\d+)`)
)

// CodexEngine represents the Codex agentic engine
type CodexEngine struct {
	BaseEngine
}

var _ CodingAgentEngine = (*CodexEngine)(nil)

func NewCodexEngine() *CodexEngine {
	return &CodexEngine{
		BaseEngine: BaseEngine{
			id:               "codex",
			displayName:      "Codex",
			description:      "Uses OpenAI Codex CLI with MCP server support",
			experimental:     false,
			ghSkillAgentName: "codex",
			capabilities: EngineCapabilities{
				ToolsAllowlist:   true,
				MaxTurns:         true,  // AWF max-turns is supported for Codex runs
				MaxContinuations: false, // Codex does not support --max-autopilot-continues-style continuation mode
				WebSearch:        true,  // Codex has built-in web-search support
				NativeAgentFile:  false, // Codex does not support agent file natively; the compiler prepends the agent file content to prompt.txt
			},
			dedicatedLLMGatewayPort: constants.CodexLLMGatewayPort,
		},
	}
}

// GetModelEnvVarName returns an empty string because the Codex CLI does not support
// selecting the model via a native environment variable. Model selection for Codex
// is done via the --model flag in the shell command.
func (e *CodexEngine) GetModelEnvVarName() string {
	return ""
}

// ResolveLLMProvider returns the effective provider for Codex inference.
// Default is openai, overridable via engine.provider (or engine.model-provider).
func (e *CodexEngine) ResolveLLMProvider(workflowData *WorkflowData) string {
	return resolveEngineLLMProvider(workflowData, LLMProviderOpenAI)
}

// GetRequiredSecretNames returns the list of secrets required by the Codex engine
// This includes CODEX_API_KEY, OPENAI_API_KEY, and optionally MCP_GATEWAY_API_KEY and mcp-scripts secrets
func (e *CodexEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	return append([]string{"CODEX_API_KEY", "OPENAI_API_KEY"}, collectCommonMCPSecrets(workflowData)...)
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the Codex engine
// supports as defined in the AWF specification.
func (e *CodexEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.CodexAPIKey,
		constants.OpenAIAPIKey,
	}
}

// GetSecretValidationStep returns the secret validation step for the Codex engine.
// Returns an empty step if custom command is specified.
func (e *CodexEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"CODEX_API_KEY", "OPENAI_API_KEY"},
		"Codex",
		"https://github.github.com/gh-aw/reference/engines/#openai-codex",
	)
}

func (e *CodexEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	codexEngineLog.Printf("Generating installation steps for Codex engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		codexEngineLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	steps := BuildStandardNpmEngineInstallStepsNoCooldown(
		"@openai/codex",
		string(constants.DefaultCodexVersion),
		"Install Codex CLI",
		"codex",
		workflowData,
	)
	if isDockerSbxRuntime(workflowData) {
		version := string(constants.DefaultCodexVersion)
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
			version = workflowData.EngineConfig.Version
		}
		steps = append(steps, GenerateDockerSbxNpmCLIInstallStep(
			"@openai/codex",
			version,
			"Install Codex CLI in docker-sbx path",
			"codex",
			false,
			false,
		))
	}

	// Add AWF installation step if firewall is enabled
	if isFirewallEnabled(workflowData) {
		steps = append(steps, generateCodexAWFInstallationSteps(workflowData)...)
	}

	return steps
}

func generateCodexAWFInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	firewallConfig := getFirewallConfig(workflowData)
	agentConfig := getAgentConfig(workflowData)
	var awfVersion string
	if firewallConfig != nil {
		awfVersion = firewallConfig.Version
	}

	var steps []GitHubActionStep
	if isGVisorRuntime(workflowData) {
		steps = append(steps, generateGVisorInstallStep())
	}
	if isDockerSbxRuntime(workflowData) {
		steps = append(steps, generateDockerSbxKVMCheckStep())
		steps = append(steps, generateDockerSbxSecretsCheckStep())
		steps = append(steps, generateDockerSbxInstallStep())
		steps = append(steps, generateDockerSbxAuthAndDaemonStep())
		steps = append(steps, generateDockerSbxPreFlightStep())
	}
	if awfInstall := generateAWFInstallationStep(awfVersion, agentConfig); len(awfInstall) > 0 {
		steps = append(steps, awfInstall)
	}
	return steps
}

// GetDeclaredOutputFiles returns the output files that Codex may produce.
// Use /tmp/gh-aw for Codex runtime logs because ${RUNNER_TEMP}/gh-aw is
// mounted read-only inside the AWF chroot sandbox.
func (e *CodexEngine) GetDeclaredOutputFiles() []string {
	// Return the Codex log directory for artifact collection.
	return []string{
		constants.TmpMcpConfigLogsDir,
	}
}

// GetAgentManifestFiles returns Codex-specific instruction files that should be
// treated as security-sensitive manifests.  AGENTS.md is the primary OpenAI
// Codex agent-instruction file; modifying it can redirect agent behaviour.
// CLAUDE.md and GEMINI.md are also listed because repositories often use multiple
// engines and Codex runs alongside them.
func (e *CodexEngine) GetAgentManifestFiles() []string {
	return []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}
}

// GetAgentManifestPathPrefixes returns Codex-specific config directory prefixes.
// The .codex/ directory can contain agent configuration and task-specific settings.
func (e *CodexEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".codex/"}
}

// GetHarnessScriptName returns the filename of the JavaScript harness script that wraps
// Codex CLI execution with retry logic for transient OpenAI API errors.
func (e *CodexEngine) GetHarnessScriptName() string {
	return "codex_harness.cjs"
}

// GetExecutionSteps returns the GitHub Actions steps for executing Codex
func (e *CodexEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
	firewallEnabled := isFirewallEnabled(workflowData)
	codexEngineLog.Printf("Building Codex execution steps: workflow=%s, modelConfigured=%v, firewall=%v",
		workflowData.Name, modelConfigured, firewallEnabled)

	var steps []GitHubActionStep

	isDetectionJob := workflowData.SafeOutputs == nil
	modelEnvVar := codexModelEnvVar(isDetectionJob)
	modelParam := fmt.Sprintf(`${%s:+ --model "$%s"}`, modelEnvVar, modelEnvVar)

	codexCommandConfig := e.buildCodexCommandConfig(workflowData, modelParam, firewallEnabled)
	command := buildCodexExecutionCommand(workflowData, logFile, firewallEnabled, codexCommandConfig)

	// Get effective GitHub token based on precedence: custom token > default
	env := buildCodexExecutionEnv(workflowData)
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	setCodexPhaseAndVersionEnv(env, workflowData)
	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)

	if firewallEnabled {
		maps.Copy(env, getGitIdentityEnvVars())
	}

	applyOptionalEngineToolTimeouts(env, workflowData)
	applyEngineMaxTurnsEnv(env, workflowData)
	applyEngineHarnessRetryEnv(env, workflowData)

	// Set the model environment variable.
	// Codex has no native model env var, so model selection always goes through
	// GH_AW_MODEL_AGENT_CODEX / GH_AW_MODEL_DETECTION_CODEX with shell expansion.
	// When model is configured (static or GitHub Actions expression), set the env var directly.
	// When not configured, use the GitHub variable fallback so users can set a default.
	if modelConfigured {
		codexEngineLog.Printf("Setting %s env var for model: %s", modelEnvVar, workflowData.EngineConfig.Model)
		env[modelEnvVar] = workflowData.EngineConfig.Model
	} else {
		env[modelEnvVar] = compilerenv.BuildModelOverrideExpression(modelEnvVar, compilerenv.DefaultModelCodex, constants.CodexDefaultModel)
	}

	applyEngineCwdEnv(env, workflowData)
	applyEngineAndAgentEnv(env, workflowData, codexEngineLog)
	applyMCPScriptsSecretEnv(env, workflowData)

	steps = append(steps, e.codexExecutionStep(workflowData, command, env))

	return steps
}

type codexCommandConfig struct {
	commandName             string
	harnessScriptName       string
	codexCommand            string
	detectionSchemaWriteCmd string
}

func codexModelEnvVar(isDetectionJob bool) string {
	if isDetectionJob {
		return constants.EnvVarModelDetectionCodex
	}
	return constants.EnvVarModelAgentCodex
}

func (e *CodexEngine) buildCodexCommandConfig(workflowData *WorkflowData, modelParam string, firewallEnabled bool) codexCommandConfig {
	commandName := codexCommandName(workflowData)
	harnessScriptName := e.codexHarnessScriptName(workflowData)
	structuredOutputParam, detectionSchemaWriteCmd := codexStructuredOutputParams(workflowData)
	codexCommand := buildCodexCommand(workflowData, commandName, harnessScriptName, modelParam, structuredOutputParam, firewallEnabled)
	return codexCommandConfig{commandName, harnessScriptName, codexCommand, detectionSchemaWriteCmd}
}

func codexCommandName(workflowData *WorkflowData) string {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		codexEngineLog.Printf("Using custom command: %s", workflowData.EngineConfig.Command)
		return workflowData.EngineConfig.Command
	}
	return "codex"
}

func (e *CodexEngine) codexHarnessScriptName(workflowData *WorkflowData) string {
	harnessScriptName := e.GetHarnessScriptName()
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.HarnessScript != "" {
		harnessScriptName = workflowData.EngineConfig.HarnessScript
		codexEngineLog.Printf("Using custom harness script: %s", harnessScriptName)
	}
	return harnessScriptName
}

func codexStructuredOutputParams(workflowData *WorkflowData) (string, string) {
	if !workflowData.IsDetectionRun {
		return "", ""
	}
	structuredOutputParam := fmt.Sprintf(` --output-schema %s -o %s`, detectionSchemaFilePath, detectionResultFilePath)
	detectionSchemaWriteCmd := fmt.Sprintf("mkdir -p /tmp/gh-aw/threat-detection && printf '%%s' '%s' > %s", detectionResponseSchema, detectionSchemaFilePath)
	codexEngineLog.Printf("Enabling structured outputs for Codex detection run")
	return structuredOutputParam, detectionSchemaWriteCmd
}

func buildCodexCommand(workflowData *WorkflowData, commandName, harnessScriptName, modelParam, structuredOutputParam string, firewallEnabled bool) string {
	webSearchParam, webFetchParam := codexWebParams(workflowData)
	executionPolicyParam := codexExecutionPolicyParam(firewallEnabled)
	customArgsParam := codexCustomArgsParam(workflowData)
	if harnessScriptName != "" {
		execPrefix := fmt.Sprintf(`%s %s/%s %s`, nodeRuntimeResolutionCommand, SetupActionDestinationShell, harnessScriptName, commandName)
		return fmt.Sprintf("%s exec%s%s%s%s%s%s --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt",
			execPrefix, modelParam, webSearchParam, webFetchParam, executionPolicyParam, structuredOutputParam, customArgsParam)
	}
	return getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + fmt.Sprintf("%s exec%s%s%s%s%s%s \"$INSTRUCTION\"",
		commandName, modelParam, webSearchParam, webFetchParam, executionPolicyParam, structuredOutputParam, customArgsParam)
}

func codexWebParams(workflowData *WorkflowData) (string, string) {
	webSearchParam := ` -c web_search="disabled"`
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebSearch != nil {
		webSearchParam = ""
	}
	webFetchParam := ` -c fetch="disabled"`
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebFetch != nil {
		webFetchParam = ""
	}
	return webSearchParam, webFetchParam
}

func codexExecutionPolicyParam(firewallEnabled bool) string {
	if firewallEnabled {
		return " --dangerously-bypass-approvals-and-sandbox --skip-git-repo-check "
	}
	return ` --sandbox workspace-write --skip-git-repo-check -c approval_policy="never" `
}

func codexCustomArgsParam(workflowData *WorkflowData) string {
	if workflowData.EngineConfig == nil || len(workflowData.EngineConfig.Args) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, arg := range workflowData.EngineConfig.Args {
		sb.WriteString(arg + " ")
	}
	return sb.String()
}

func buildCodexExecutionCommand(workflowData *WorkflowData, logFile string, firewallEnabled bool, config codexCommandConfig) string {
	if firewallEnabled {
		return buildAWFCodexExecutionCommand(workflowData, logFile, config)
	}
	return buildDirectCodexExecutionCommand(workflowData, logFile, config)
}

func buildAWFCodexExecutionCommand(workflowData *WorkflowData, logFile string, config codexCommandConfig) string {
	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         "codex",
		EngineCommand:      codexCommandWithSandboxPathSetup(workflowData, config.codexCommand, config.harnessScriptName),
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            false,
		AllowedDomains:     codexAllowedDomains(workflowData),
		PathSetup:          codexAWFPathSetup(workflowData, config.detectionSchemaWriteCmd),
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"CODEX_API_KEY", "OPENAI_API_KEY"}),
	})
}

func codexAllowedDomains(workflowData *WorkflowData) string {
	var allowedDomains string
	if workflowData.CachedAllowedDomainsComputed {
		allowedDomains = workflowData.CachedAllowedDomainsStr
	} else {
		allowedDomains = GetAllowedDomainsForEngine(constants.CodexEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
	}
	return allowedDomains
}

func codexCommandWithSandboxPathSetup(workflowData *WorkflowData, codexCommand, harnessScriptName string) string {
	var commandWithSetup string
	if harnessScriptName != "" {
		commandWithSetup = fmt.Sprintf(`%s && %s`, GetNpmBinPathSetup(), codexCommand)
	} else {
		commandWithSetup = fmt.Sprintf(`%s && INSTRUCTION="$(cat /tmp/gh-aw/aw-prompts/prompt.txt)" && %s`, GetNpmBinPathSetup(), codexCommand)
	}
	if dockerSbxCLIPath := GetDockerSbxNpmCLIPathSetup(workflowData); dockerSbxCLIPath != "" {
		commandWithSetup = fmt.Sprintf("%s && %s", dockerSbxCLIPath, commandWithSetup)
	}
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		commandWithSetup = fmt.Sprintf("%s && %s", mcpCLIPath, commandWithSetup)
	}
	return commandWithSetup
}

func codexAWFPathSetup(workflowData *WorkflowData, detectionSchemaWriteCmd string) string {
	base := "mkdir -p \"$CODEX_HOME/logs\" && touch " + AgentStepSummaryPath
	if workflowData.IsDetectionRun {
		return base + " && " + detectionSchemaWriteCmd
	}
	return base
}

func buildDirectCodexExecutionCommand(workflowData *WorkflowData, logFile string, config codexCommandConfig) string {
	schemaWritePrefix := ""
	if workflowData.IsDetectionRun {
		schemaWritePrefix = config.detectionSchemaWriteCmd + " && "
	}
	if config.harnessScriptName != "" {
		return fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
mkdir -p "$CODEX_HOME/logs"
%s%s 2>&1 | tee %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, schemaWritePrefix, config.codexCommand, logFile)
	}
	return fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
INSTRUCTION="$(cat "$GH_AW_PROMPT")"
mkdir -p "$CODEX_HOME/logs"
%s%s 2>&1 | tee %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, schemaWritePrefix, config.codexCommand, logFile)
}

func buildCodexExecutionEnv(workflowData *WorkflowData) map[string]string {
	effectiveGitHubToken := getEffectiveGitHubToken("")
	return map[string]string{
		"CODEX_API_KEY":                "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
		"GITHUB_STEP_SUMMARY":          AgentStepSummaryPath,
		"GH_AW_PROMPT":                 constants.AwPromptsFile,
		"GITHUB_AW":                    "true",
		"RUNNER_TEMP":                  "${{ runner.temp }}",
		"GH_AW_MCP_CONFIG":             constants.CodexMcpConfigTomlPath,
		"CODEX_HOME":                   constants.TmpMcpConfigDir,
		"RUST_LOG":                     "${{ runner.debug == 1 && 'trace,hyper_util=info,mio=info,reqwest=info,os_info=info,codex_otel=warn,codex_core=debug,ocodex_exec=debug' || 'warn' }}",
		"GH_AW_GITHUB_TOKEN":           effectiveGitHubToken,
		"GITHUB_PERSONAL_ACCESS_TOKEN": effectiveGitHubToken,
		"OPENAI_API_KEY":               "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
	}
}

func setCodexPhaseAndVersionEnv(env map[string]string, workflowData *WorkflowData) {
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

func (e *CodexEngine) codexExecutionStep(workflowData *WorkflowData, command string, env map[string]string) GitHubActionStep {
	stepLines := []string{
		"      - name: Execute Codex CLI",
		"        id: agentic_execution",
	}
	filteredEnv := FilterEnvForSecrets(env, e.GetRequiredSecretNames(workflowData))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	return GitHubActionStep(FormatStepWithCommandAndEnv(stepLines, command, filteredEnv))
}

// GetSquidLogsSteps returns the steps for uploading and parsing Squid logs (after secret redaction)
func (e *CodexEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	return defaultGetSquidLogsSteps(workflowData, codexEngineLog)
}

// expandNeutralToolsToCodexTools converts neutral tools to Codex-specific tools format
// This ensures that playwright tools get the same allowlist as the copilot agent
// Updated to use ToolsConfig instead of map[string]any
func (e *CodexEngine) expandNeutralToolsToCodexTools(toolsConfig *ToolsConfig) *ToolsConfig {
	if toolsConfig == nil {
		return &ToolsConfig{
			Custom: make(map[string]MCPServerConfig),
			raw:    make(map[string]any),
		}
	}

	result := cloneCodexToolsConfig(toolsConfig)

	// Handle playwright tool by converting it to an MCP tool configuration with copilot agent tools
	if toolsConfig.Playwright != nil {
		applyCodexPlaywrightConfig(result, toolsConfig.Playwright)
	}

	return result
}

func cloneCodexToolsConfig(toolsConfig *ToolsConfig) *ToolsConfig {
	result := &ToolsConfig{
		GitHub:           toolsConfig.GitHub,
		Bash:             toolsConfig.Bash,
		WebFetch:         toolsConfig.WebFetch,
		WebSearch:        toolsConfig.WebSearch,
		Edit:             toolsConfig.Edit,
		Playwright:       toolsConfig.Playwright,
		AgenticWorkflows: toolsConfig.AgenticWorkflows,
		CacheMemory:      toolsConfig.CacheMemory,
		Timeout:          toolsConfig.Timeout,
		StartupTimeout:   toolsConfig.StartupTimeout,
		Custom:           make(map[string]MCPServerConfig),
		raw:              make(map[string]any),
	}
	maps.Copy(result.Custom, toolsConfig.Custom)
	maps.Copy(result.raw, toolsConfig.raw)
	return result
}

func applyCodexPlaywrightConfig(result *ToolsConfig, source *PlaywrightToolConfig) {
	playwrightConfig := &PlaywrightToolConfig{
		Version: source.Version,
		Args:    source.Args,
		Mode:    source.Mode,
	}
	result.Playwright = playwrightConfig
	if playwrightConfig.IsCLIMode() {
		delete(result.raw, "playwright")
		return
	}
	playwrightMCP := map[string]any{"allowed": GetPlaywrightTools()}
	if playwrightConfig.Version != "" {
		playwrightMCP["version"] = playwrightConfig.Version
	}
	if len(playwrightConfig.Args) > 0 {
		playwrightMCP["args"] = playwrightConfig.Args
	}
	result.raw["playwright"] = playwrightMCP
}

// expandNeutralToolsToCodexToolsFromMap is a backward compatibility wrapper
// that accepts map[string]any instead of *ToolsConfig
func (e *CodexEngine) expandNeutralToolsToCodexToolsFromMap(tools map[string]any) map[string]any {
	toolsConfig, _ := ParseToolsConfig(tools)
	result := e.expandNeutralToolsToCodexTools(toolsConfig)
	return result.ToMap()
}

func (e *CodexEngine) getShellEnvironmentPolicyVars(tools map[string]any, mcpTools []string) []string {
	// Collect all environment variables needed by MCP servers
	envVars := make(map[string]struct{})

	// Always include core environment variables
	envVars["PATH"] = struct{}{}
	envVars["HOME"] = struct{}{}

	// Add CODEX_API_KEY for authentication
	envVars["CODEX_API_KEY"] = struct{}{}
	envVars["OPENAI_API_KEY"] = struct{}{} // Fallback for CODEX_API_KEY

	// Check each MCP tool for required environment variables
	for _, toolName := range mcpTools {
		addMCPToolEnvVars(toolName, tools, envVars)
	}

	sortedEnvVars := sliceutil.SortedKeys(envVars)

	// Codex expects regex patterns for shell_environment_policy.include_only, not literal names.
	// Anchor each variable name to avoid accidental substring matches (for example "PATH" matching "PATH_SUFFIX").
	var includeOnlyPatterns []string
	for _, envVar := range sortedEnvVars {
		includeOnlyPatterns = append(includeOnlyPatterns, "^"+regexp.QuoteMeta(envVar)+"$")
	}
	return includeOnlyPatterns
}

// addMCPToolEnvVars adds the environment variables required by the named MCP tool
// to the envVars set. For custom tools, it reads the "env" configuration map.
func addMCPToolEnvVars(toolName string, tools map[string]any, envVars map[string]struct{}) {
	switch toolName {
	case "github":
		// GitHub MCP server needs GITHUB_PERSONAL_ACCESS_TOKEN
		envVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = struct{}{}
	case "agentic-workflows":
		// Agentic workflows MCP server needs GITHUB_TOKEN
		envVars["GITHUB_TOKEN"] = struct{}{}
	case "safe-outputs":
		// Safe outputs MCP server needs several environment variables
		envVars["GH_AW_SAFE_OUTPUTS"] = struct{}{}
		envVars["GH_AW_ASSETS_BRANCH"] = struct{}{}
		envVars["GH_AW_ASSETS_MAX_SIZE_KB"] = struct{}{}
		envVars["GH_AW_ASSETS_ALLOWED_EXTS"] = struct{}{}
		envVars["GITHUB_REPOSITORY"] = struct{}{}
		envVars["GITHUB_SERVER_URL"] = struct{}{}
	default:
		// For custom MCP tools, check if they have env configuration
		if toolValue, ok := tools[toolName]; ok {
			if toolConfig, ok := toolValue.(map[string]any); ok {
				// Extract environment variable names from env configuration
				if env, hasEnv := toolConfig["env"].(map[string]any); hasEnv {
					for envKey := range env {
						envVars[envKey] = struct{}{}
					}
				}
			}
		}
	}
}

// renderShellEnvironmentPolicy generates the [shell_environment_policy] section for config.toml
// This controls which environment variables are passed through to MCP servers for security
func (e *CodexEngine) renderShellEnvironmentPolicy(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	sortedEnvVars := e.getShellEnvironmentPolicyVars(tools, mcpTools)

	// Render [shell_environment_policy] section
	yaml.WriteString("          \n")
	yaml.WriteString("          [shell_environment_policy]\n")
	yaml.WriteString("          inherit = \"core\"\n")
	yaml.WriteString("          include_only = [")
	for i, envVar := range sortedEnvVars {
		if i > 0 {
			yaml.WriteString(", ")
		}
		yaml.WriteString("\"" + envVar + "\"")
	}
	yaml.WriteString("]\n")
}

func (e *CodexEngine) renderShellEnvironmentPolicyToml(yaml *strings.Builder, tools map[string]any, mcpTools []string, indent string) {
	sortedEnvVars := e.getShellEnvironmentPolicyVars(tools, mcpTools)

	yaml.WriteString(indent + "[shell_environment_policy]\n")
	yaml.WriteString(indent + "inherit = \"core\"\n")
	yaml.WriteString(indent + "include_only = [")
	for i, envVar := range sortedEnvVars {
		if i > 0 {
			yaml.WriteString(", ")
		}
		yaml.WriteString("\"" + envVar + "\"")
	}
	yaml.WriteString("]\n")
}

// RenderMCPConfig is implemented in codex_mcp.go

// renderCodexMCPConfig is implemented in codex_mcp.go

// ParseLogMetrics is implemented in codex_logs.go

// parseCodexToolCallsWithSequence is implemented in codex_logs.go

// updateMostRecentToolWithDuration is implemented in codex_logs.go

// extractCodexTokenUsage is implemented in codex_logs.go

// GetLogParserScriptId is implemented in codex_logs.go
