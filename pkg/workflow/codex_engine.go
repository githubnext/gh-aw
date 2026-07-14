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
// Default is openai, overridable via engine.model-provider.
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
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}

		// gVisor must be installed and registered BEFORE AWF starts the agent container.
		if isGVisorRuntime(workflowData) {
			steps = append(steps, generateGVisorInstallStep())
		}

		// docker-sbx must be installed, authenticated, and smoke-tested BEFORE AWF.
		if isDockerSbxRuntime(workflowData) {
			steps = append(steps, generateDockerSbxKVMCheckStep())
			steps = append(steps, generateDockerSbxSecretsCheckStep())
			steps = append(steps, generateDockerSbxInstallStep())
			steps = append(steps, generateDockerSbxAuthAndDaemonStep())
			steps = append(steps, generateDockerSbxPreFlightStep())
		}

		// Install AWF binary (or skip if custom command is specified)
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
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

	// Codex does not support a native model environment variable, so model selection
	// always uses GH_AW_MODEL_AGENT_CODEX or GH_AW_MODEL_DETECTION_CODEX with shell expansion
	// via the --model flag. This also correctly handles GitHub Actions expressions like ${{ inputs.model }}.
	// Note: Codex also supports config-layer model selection (config key `model`, including `-c model="..."`),
	// but `--model` is a direct CLI flag and avoids TOML quoting/parsing edge cases in automation.
	isDetectionJob := workflowData.SafeOutputs == nil
	var modelEnvVar string
	if isDetectionJob {
		modelEnvVar = constants.EnvVarModelDetectionCodex
	} else {
		modelEnvVar = constants.EnvVarModelAgentCodex
	}
	modelParam := fmt.Sprintf(`${%s:+ --model "$%s"}`, modelEnvVar, modelEnvVar)

	// Build search parameter: disable web search by default, enable only if web-search tool is present.
	// Codex enables web search by default, so we must explicitly set web_search="disabled" to disable it.
	// The --no-search flag does not exist; use the -c web_search="disabled" config option instead.
	// See https://developers.openai.com/codex/cli/features#web-search
	// Leading space is intentional: these params are concatenated directly and need their own separator.
	webSearchParam := ` -c web_search="disabled"`
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebSearch != nil {
		// Web search is enabled by default in Codex; no extra flag needed.
		webSearchParam = ""
	}

	// Build fetch parameter: enforce AWF default-deny for fetch unless web-fetch tool is present.
	// Codex enables fetch by default, so this code explicitly sets fetch="disabled" unless web-fetch is configured.
	// Leading space is intentional: these params are concatenated directly and need their own separator.
	webFetchParam := ` -c fetch="disabled"`
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebFetch != nil {
		// When web-fetch is configured, omit override so Codex default fetch behavior remains enabled.
		webFetchParam = ""
	}

	// See https://github.com/github/gh-aw/issues/892
	// In AWF mode we bypass Codex approvals/sandboxing because AWF provides the sandbox layer.
	// Outside AWF, keep Codex sandboxing enabled and disable approvals for non-interactive execution.
	executionPolicyParam := ` --sandbox workspace-write --skip-git-repo-check -c approval_policy="never" `
	if firewallEnabled {
		executionPolicyParam = " --dangerously-bypass-approvals-and-sandbox --skip-git-repo-check "
	}

	// Build custom args parameter if specified in engineConfig
	var customArgsParam string
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		var customArgsParamSb strings.Builder
		for _, arg := range workflowData.EngineConfig.Args {
			customArgsParamSb.WriteString(arg + " ")
		}
		customArgsParam += customArgsParamSb.String()
	}

	// Build structured output parameter for detection runs.
	// Use --output-schema to constrain Codex output to the threat detection JSON schema,
	// and -o (--output-last-message) to write the final structured verdict directly to a
	// file. The parser (parse_threat_detection_results.cjs) reads detection_result.json
	// first, bypassing the noisy log stream that caused false parse_error warnings.
	//
	// The schema file is written to detectionSchemaFilePath before Codex runs:
	//   - AWF mode: in PathSetup (runs on host before the AWF container starts)
	//   - Non-AWF mode: in the command preamble (inline shell command)
	// Because /tmp/gh-aw/ is the read-write runtime tree mounted in both the host and
	// the AWF container, the schema file is accessible inside the container and the
	// result file written inside the container is accessible on the host after exit.
	var structuredOutputParam string
	var detectionSchemaWriteCmd string
	if workflowData.IsDetectionRun {
		// --output-schema <file>: constrain model output to the threat detection schema
		// -o <file>: write the final structured verdict to a file for direct parsing
		structuredOutputParam = fmt.Sprintf(` --output-schema %s -o %s`, detectionSchemaFilePath, detectionResultFilePath)
		// Shell command to write the schema file before Codex runs.
		// printf '%s' avoids the need to escape the JSON (no single quotes in schema).
		detectionSchemaWriteCmd = fmt.Sprintf("mkdir -p /tmp/gh-aw/threat-detection && printf '%%s' '%s' > %s", detectionResponseSchema, detectionSchemaFilePath)
		codexEngineLog.Printf("Enabling structured outputs for Codex detection run")
	}

	// Build the Codex command
	// Determine which command to use
	var commandName string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
		codexEngineLog.Printf("Using custom command: %s", commandName)
	} else {
		// Use regular codex command - PATH is inherited via --env-all in AWF mode
		commandName = "codex"
	}

	// Determine harness script to wrap codex execution.
	// The built-in harness provides retry logic for transient OpenAI API errors
	// (rate limits, server errors).  A custom engine.harness overrides the built-in one.
	harnessScriptName := e.GetHarnessScriptName()
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.HarnessScript != "" {
		harnessScriptName = workflowData.EngineConfig.HarnessScript
		codexEngineLog.Printf("Using custom harness script: %s", harnessScriptName)
	}

	// Build the Codex command.
	// The default harness (codex_harness.cjs) wraps execution with retry logic and reads the
	// prompt via --prompt-file.  The else branch is a defensive fallback for the case where
	// harnessScriptName is empty (e.g. a future code path that does not set a harness).
	var codexCommand string
	if harnessScriptName != "" {
		// Harness-wrapped execution: the harness reads --prompt-file and passes its content
		// as the last positional arg.  The harness also provides retry logic.
		// The harness sets cwd=GITHUB_WORKSPACE when spawning the codex process, so no
		// shell-level cd prefix is needed.
		execPrefix := fmt.Sprintf(`%s %s/%s %s`, nodeRuntimeResolutionCommand, SetupActionDestinationShell, harnessScriptName, commandName)
		codexCommand = fmt.Sprintf("%s exec%s%s%s%s%s%s --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt",
			execPrefix, modelParam, webSearchParam, webFetchParam, executionPolicyParam, structuredOutputParam, customArgsParam)
	} else {
		// Without harness: use shell expansion for the prompt (no retry logic).
		// Apply workspace prefix here since there is no JS harness to set the cwd.
		codexCommand = getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + fmt.Sprintf("%s exec%s%s%s%s%s%s \"$INSTRUCTION\"",
			commandName, modelParam, webSearchParam, webFetchParam, executionPolicyParam, structuredOutputParam, customArgsParam)
	}

	// Build the full command with agent file handling and AWF wrapping if enabled
	var command string
	if firewallEnabled {
		// Build AWF-wrapped command using helper function
		// Get allowed domains: prefer the pre-warmed cache on WorkflowData to avoid
		// re-running the expensive map+sort operation.
		var allowedDomains string
		if workflowData.CachedAllowedDomainsComputed {
			allowedDomains = workflowData.CachedAllowedDomainsStr
		} else {
			allowedDomains = GetAllowedDomainsForEngine(constants.CodexEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
		}
		// Add GHES/custom API target domains to the firewall allow-list when engine.api-target is set
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
		}

		// AWF v0.15.0+ with --env-all handles most PATH setup natively (chroot mode is default):
		// - GOROOT, JAVA_HOME, etc. are handled via AWF_HOST_PATH and entrypoint.sh
		// However, npm-installed CLIs (like codex) need hostedtoolcache bin directories in PATH.
		npmPathSetup := GetNpmBinPathSetup()

		// Build the codex command with PATH setup inside the AWF container.
		// For engines that do not support native agent-file handling (including Codex),
		// the compiler prepends the agent file content to prompt.txt.
		// When using the harness, --prompt-file is passed directly; otherwise the prompt
		// is read via shell variable expansion.
		var codexCommandWithSetup string
		if harnessScriptName != "" {
			// Harness handles prompt reading via --prompt-file; no INSTRUCTION variable needed.
			codexCommandWithSetup = fmt.Sprintf(`%s && %s`, npmPathSetup, codexCommand)
		} else {
			codexCommandWithSetup = fmt.Sprintf(`%s && INSTRUCTION="$(cat /tmp/gh-aw/aw-prompts/prompt.txt)" && %s`, npmPathSetup, codexCommand)
		}
		if dockerSbxCLIPath := GetDockerSbxNpmCLIPathSetup(workflowData); dockerSbxCLIPath != "" {
			codexCommandWithSetup = fmt.Sprintf("%s && %s", dockerSbxCLIPath, codexCommandWithSetup)
		}
		// Add MCP CLI bin directory to PATH when cli-proxy is enabled.
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			codexCommandWithSetup = fmt.Sprintf("%s && %s", mcpCLIPath, codexCommandWithSetup)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "codex",
			EngineCommand:  codexCommandWithSetup,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false, // Codex is not a TUI, outputs to stdout/stderr
			AllowedDomains: allowedDomains,
			// Create logs directory and agent step summary file before AWF.
			// For detection runs, also write the JSON schema file that --output-schema
			// references. PathSetup runs on the host before the AWF container starts;
			// /tmp/gh-aw/ is the read-write runtime tree mounted in both environments,
			// so the schema file is accessible inside the container.
			PathSetup: func() string {
				base := "mkdir -p \"$CODEX_HOME/logs\" && touch " + AgentStepSummaryPath
				if workflowData.IsDetectionRun {
					return base + " && " + detectionSchemaWriteCmd
				}
				return base
			}(),
			// Exclude Codex/OpenAI API key env vars from the AWF container.
			// AWF's API proxy handles auth, so raw token values should not be
			// visible to in-container tools (e.g., env/printenv).
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"CODEX_API_KEY", "OPENAI_API_KEY"}),
		})
	} else {
		// Build the command without AWF wrapping.
		// For engines that do not support native agent-file handling (including Codex),
		// the compiler prepends the agent file content to prompt.txt so no special
		// shell variable juggling is needed here.

		// Optionally prefix the detection schema write command for detection runs.
		// Keep it chained with "&&" so a schema write failure stops before codex runs.
		schemaWritePrefix := ""
		if workflowData.IsDetectionRun {
			schemaWritePrefix = detectionSchemaWriteCmd + " && "
		}

		if harnessScriptName != "" {
			// Harness handles prompt reading via --prompt-file; no INSTRUCTION variable needed.
			command = fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
mkdir -p "$CODEX_HOME/logs"
%s%s 2>&1 | tee %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, schemaWritePrefix, codexCommand, logFile)
		} else {
			command = fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
INSTRUCTION="$(cat "$GH_AW_PROMPT")"
mkdir -p "$CODEX_HOME/logs"
%s%s 2>&1 | tee %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, schemaWritePrefix, codexCommand, logFile)
		}
	}

	// Get effective GitHub token based on precedence: custom token > default
	effectiveGitHubToken := getEffectiveGitHubToken("")

	env := map[string]string{
		"CODEX_API_KEY": "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
		// Override GITHUB_STEP_SUMMARY with a path that exists inside the sandbox.
		// The runner's original path is unreachable within the AWF isolated filesystem;
		// we create this file before the agent starts and append it to the real
		// $GITHUB_STEP_SUMMARY after secret redaction.
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
		"GH_AW_PROMPT":        constants.AwPromptsFile,
		// Tag the step as a GitHub AW agentic execution for discoverability by agents
		"GITHUB_AW":        "true",
		"RUNNER_TEMP":      "${{ runner.temp }}",
		"GH_AW_MCP_CONFIG": constants.CodexMcpConfigTomlPath,
		// Keep Codex runtime state in /tmp/gh-aw because ${RUNNER_TEMP}/gh-aw is
		// mounted read-only inside the AWF chroot sandbox.
		"CODEX_HOME": constants.TmpMcpConfigDir,
		// Enable verbose RUST_LOG only in debug mode (runner.debug == 1); default to warn to avoid noisy output.
		"RUST_LOG":                     "${{ runner.debug == 1 && 'trace,hyper_util=info,mio=info,reqwest=info,os_info=info,codex_otel=warn,codex_core=debug,ocodex_exec=debug' || 'warn' }}",
		"GH_AW_GITHUB_TOKEN":           effectiveGitHubToken,
		"GITHUB_PERSONAL_ACCESS_TOKEN": effectiveGitHubToken,                                     // Used by GitHub MCP server via env_vars
		"OPENAI_API_KEY":               "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}", // Fallback for CODEX_API_KEY
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	// Indicate the phase: "agent" for the main run, "detection" for threat detection
	// Include the compiler version so agents can identify which gh-aw version generated the workflow
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

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Propagate W3C trace context so engine spans nest under the gh-aw.agent.setup span.
	applyTraceContextEnvToMap(env)

	// In sandbox (AWF) mode, set git identity environment variables so the first git commit
	// succeeds inside the container. AWF's --env-all forwards these to the container, ensuring
	// git does not rely on the host-side ~/.gitconfig which is not visible in the sandbox.
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

	// Generate the step for Codex execution
	stepName := "Execute Codex CLI"
	var stepLines []string

	stepLines = append(stepLines, "      - name: "+stepName)
	stepLines = append(stepLines, "        id: agentic_execution")

	// Filter environment variables to only include allowed secrets
	// This is a security measure to prevent exposing unnecessary secrets to the AWF container
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Inject GH_TOKEN for CLI proxy (added after filtering since it uses a special
	// fallback expression that is always allowed when cli-proxy is enabled)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)

	// Format step with command and filtered environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
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

	// Create a copy of the tools config
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

	// Copy custom tools
	maps.Copy(result.Custom, toolsConfig.Custom)

	// Copy raw map
	maps.Copy(result.raw, toolsConfig.raw)

	// Handle playwright tool by converting it to an MCP tool configuration with copilot agent tools
	if toolsConfig.Playwright != nil {
		// Create an updated Playwright config preserving all fields including Mode
		playwrightConfig := &PlaywrightToolConfig{
			Version: toolsConfig.Playwright.Version,
			Args:    toolsConfig.Playwright.Args,
			Mode:    toolsConfig.Playwright.Mode,
		}

		result.Playwright = playwrightConfig

		// In CLI mode, playwright is not an MCP server — remove from raw map and skip MCP config entry.
		// result.raw is populated by maps.Copy(result.raw, toolsConfig.raw) earlier in this function,
		// so delete is safe regardless of whether the key was originally present.
		if playwrightConfig.IsCLIMode() {
			delete(result.raw, "playwright")
		} else {
			// Also update the Custom map entry for playwright with allowed tools list
			playwrightMCP := map[string]any{
				"allowed": GetPlaywrightTools(),
			}
			if playwrightConfig.Version != "" {
				playwrightMCP["version"] = playwrightConfig.Version
			}
			if len(playwrightConfig.Args) > 0 {
				playwrightMCP["args"] = playwrightConfig.Args
			}

			// Update raw map for backward compatibility
			result.raw["playwright"] = playwrightMCP
		}
	}

	return result
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
