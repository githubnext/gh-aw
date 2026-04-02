package workflow

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var codexEngineLog = logger.New("workflow:codex_engine")

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
			id:                       "codex",
			displayName:              "Codex",
			description:              "Uses OpenAI Codex CLI with MCP server support",
			experimental:             false,
			supportsToolsAllowlist:   true,
			supportsMaxTurns:         false, // Codex does not support max-turns feature
			supportsMaxContinuations: false, // Codex does not support --max-autopilot-continues-style continuation mode
			supportsWebSearch:        true,  // Codex has built-in web-search support
			llmGatewayPort:           constants.CodexLLMGatewayPort,
		},
	}
}

// GetModelEnvVarName returns an empty string because the Codex CLI does not support
// selecting the model via a native environment variable. Model selection for Codex
// is done via the -c model=... configuration override in the shell command.
func (e *CodexEngine) GetModelEnvVarName() string {
	return ""
}

// GetRequiredSecretNames returns the list of secrets required by the Codex engine
// This includes CODEX_API_KEY, OPENAI_API_KEY, and optionally MCP_GATEWAY_API_KEY
func (e *CodexEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	secrets := []string{"CODEX_API_KEY", "OPENAI_API_KEY"}

	// Add MCP gateway API key if MCP servers are present (gateway is always started with MCP servers)
	if HasMCPServers(workflowData) {
		secrets = append(secrets, "MCP_GATEWAY_API_KEY")
	}

	// Add mcp-scripts secret names
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpScriptsSecrets := collectMCPScriptsSecrets(workflowData.MCPScripts)
		for varName := range mcpScriptsSecrets {
			secrets = append(secrets, varName)
		}
	}

	return secrets
}

// GetSecretValidationStep returns the secret validation step for the Codex engine.
// Returns an empty step if custom command is specified.
func (e *CodexEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		codexEngineLog.Printf("Skipping secret validation step: custom command specified (%s)", workflowData.EngineConfig.Command)
		return GitHubActionStep{}
	}
	return GenerateMultiSecretValidationStep(
		[]string{"CODEX_API_KEY", "OPENAI_API_KEY"},
		"Codex",
		"https://github.github.com/gh-aw/reference/engines/#openai-codex",
		getEngineEnvOverrides(workflowData),
	)
}

func (e *CodexEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	codexEngineLog.Printf("Generating installation steps for Codex engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		codexEngineLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Use base installation steps (npm install only; secret validation is in the activation job)
	steps := GetBaseInstallationSteps(EngineInstallConfig{
		Secrets:         []string{"CODEX_API_KEY", "OPENAI_API_KEY"},
		DocsURL:         "https://github.github.com/gh-aw/reference/engines/#openai-codex",
		NpmPackage:      "@openai/codex",
		Version:         string(constants.DefaultCodexVersion),
		Name:            "Codex CLI",
		InstallStepName: "Install Codex CLI",
		CliName:         "codex",
	}, workflowData)

	// Add AWF installation step if firewall is enabled
	if isFirewallEnabled(workflowData) {
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}

		// Install AWF binary (or skip if custom command is specified)
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
	}

	return steps
}

// GetDeclaredOutputFiles returns the output files that Codex may produce
// Codex (written in Rust) writes logs to ~/.codex/log/codex-tui.log
func (e *CodexEngine) GetDeclaredOutputFiles() []string {
	// Return the Codex log directory for artifact collection
	// Using mcp-config folder structure for consistency with other engines
	return []string{
		"/tmp/gh-aw/mcp-config/logs/",
	}
}

// GetAgentManifestFiles returns Codex-specific instruction files that should be
// treated as security-sensitive manifests.  AGENTS.md is the standard OpenAI
// Codex agent-instruction file; modifying it can redirect agent behaviour.
func (e *CodexEngine) GetAgentManifestFiles() []string {
	return []string{"AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Codex-specific config directory prefixes.
// The .codex/ directory can contain agent configuration and task-specific settings.
func (e *CodexEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".codex/"}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Codex
func (e *CodexEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
	firewallEnabled := isFirewallEnabled(workflowData)
	codexEngineLog.Printf("Building Codex execution steps: workflow=%s, modelConfigured=%v, has_agent_file=%v, firewall=%v",
		workflowData.Name, modelConfigured, workflowData.AgentFile != "", firewallEnabled)

	var steps []GitHubActionStep

	// Codex does not support a native model environment variable, so model selection
	// always uses GH_AW_MODEL_AGENT_CODEX or GH_AW_MODEL_DETECTION_CODEX with shell expansion.
	// This also correctly handles GitHub Actions expressions like ${{ inputs.model }}.
	isDetectionJob := workflowData.SafeOutputs == nil
	var modelEnvVar string
	if isDetectionJob {
		modelEnvVar = constants.EnvVarModelDetectionCodex
	} else {
		modelEnvVar = constants.EnvVarModelAgentCodex
	}
	modelParam := fmt.Sprintf(`${%s:+-c model="$%s" }`, modelEnvVar, modelEnvVar)

	// Build search parameter: disable web search by default, enable only if web-search tool is present.
	// Codex enables web search by default, so we must explicitly set web_search="disabled" to disable it.
	// The --no-search flag does not exist; use the -c web_search="disabled" config option instead.
	// See https://developers.openai.com/codex/cli/features#web-search
	// Leading space is intentional: the format string concatenates this directly after "exec" with no space separator.
	webSearchParam := ` -c web_search="disabled"`
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebSearch != nil {
		// Web search is enabled by default in Codex; no extra flag needed.
		webSearchParam = ""
	}

	// Build fetch parameter: disable the native fetch tool by default, enable only if web-fetch tool is present.
	// Codex enables the fetch tool by default, so we must explicitly set fetch="disabled" to disable it.
	// See https://developers.openai.com/api/docs/mcp#fetch-tool
	// Leading space is intentional: the format string concatenates this directly after webSearchParam with no space separator.
	webFetchParam := ` -c fetch="disabled"`
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.WebFetch != nil {
		// Fetch is enabled by default in Codex; no extra flag needed.
		webFetchParam = ""
	}

	// See https://github.com/github/gh-aw/issues/892
	// --dangerously-bypass-approvals-and-sandbox: Skips all confirmation prompts and disables sandboxing
	// This is safe because AWF already provides a container-level sandbox layer
	// --skip-git-repo-check: Allows running in directories without a git repo
	fullAutoParam := " --dangerously-bypass-approvals-and-sandbox --skip-git-repo-check "

	// Build custom args parameter if specified in engineConfig
	var customArgsParam string
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		var customArgsParamSb strings.Builder
		for _, arg := range workflowData.EngineConfig.Args {
			customArgsParamSb.WriteString(arg + " ")
		}
		customArgsParam += customArgsParamSb.String()
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

	codexCommand := fmt.Sprintf("%s %sexec%s%s%s%s\"$INSTRUCTION\"",
		commandName, modelParam, webSearchParam, webFetchParam, fullAutoParam, customArgsParam)

	// Build the full command with agent file handling and AWF wrapping if enabled
	var command string
	if firewallEnabled {
		// Build AWF-wrapped command using helper function
		// Get allowed domains (Codex defaults + network permissions + HTTP MCP server URLs + runtime ecosystem domains)
		allowedDomains := GetCodexAllowedDomainsWithToolsAndRuntimes(workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
		// Add GHES/custom API target domains to the firewall allow-list when engine.api-target is set
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
		}

		// Build the command with agent file handling if specified
		// INSTRUCTION reading is done inside the AWF command to avoid Docker Compose interpolation
		// issues with $ characters in the prompt.
		//
		// AWF v0.15.0+ with --env-all handles most PATH setup natively (chroot mode is default):
		// - GOROOT, JAVA_HOME, etc. are handled via AWF_HOST_PATH and entrypoint.sh
		// However, npm-installed CLIs (like codex) need hostedtoolcache bin directories in PATH.
		npmPathSetup := GetNpmBinPathSetup()

		// Codex reads both agent file and prompt inside AWF container (PATH setup + agent file reading + codex command)
		var codexCommandWithSetup string
		if workflowData.AgentFile != "" {
			agentPath, err := ResolveAgentFilePath(workflowData.AgentFile)
			if err != nil {
				codexEngineLog.Printf("Error resolving agent file path: %v", err)
				return BuildInvalidAgentPathStep("Execute Codex CLI", workflowData.AgentFile, err)
			}
			// Read agent file and prompt inside AWF container, with PATH setup for npm binaries
			codexCommandWithSetup = fmt.Sprintf(`%s && AGENT_CONTENT="$(awk 'BEGIN{skip=1} /^---$/{if(skip){skip=0;next}else{skip=1;next}} !skip' %s)" && INSTRUCTION="$(printf "%%s\n\n%%s" "$AGENT_CONTENT" "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)")" && %s`, npmPathSetup, agentPath, codexCommand)
		} else {
			// Read prompt inside AWF container to avoid Docker Compose interpolation issues, with PATH setup
			codexCommandWithSetup = fmt.Sprintf(`%s && INSTRUCTION="$(cat /tmp/gh-aw/aw-prompts/prompt.txt)" && %s`, npmPathSetup, codexCommand)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "codex",
			EngineCommand:  codexCommandWithSetup,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false, // Codex is not a TUI, outputs to stdout/stderr
			AllowedDomains: allowedDomains,
			// Create logs directory and agent step summary file before AWF.
			// The agent writes its step summary content to AgentStepSummaryPath, which is
			// appended to $GITHUB_STEP_SUMMARY after secret redaction.
			PathSetup: "mkdir -p \"$CODEX_HOME/logs\" && touch " + AgentStepSummaryPath,
			// Exclude every env var whose step-env value is a secret so the agent
			// cannot read raw token values via bash tools (env / printenv).
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"CODEX_API_KEY", "OPENAI_API_KEY"}),
		})
	} else {
		// Build the command without AWF wrapping
		// Reuse commandName already determined above
		if workflowData.AgentFile != "" {
			agentPath, err := ResolveAgentFilePath(workflowData.AgentFile)
			if err != nil {
				codexEngineLog.Printf("Error resolving agent file path: %v", err)
				return BuildInvalidAgentPathStep("Execute Codex CLI", workflowData.AgentFile, err)
			}
			command = fmt.Sprintf(`set -o pipefail
touch %s
AGENT_CONTENT="$(awk 'BEGIN{skip=1} /^---$/{if(skip){skip=0;next}else{skip=1;next}} !skip' %s)"
INSTRUCTION="$(printf "%%s\n\n%%s" "$AGENT_CONTENT" "$(cat "$GH_AW_PROMPT")")"
mkdir -p "$CODEX_HOME/logs"
%s %sexec%s%s%s%s"$INSTRUCTION" 2>&1 | tee %s`, AgentStepSummaryPath, agentPath, commandName, modelParam, webSearchParam, webFetchParam, fullAutoParam, customArgsParam, logFile)
		} else {
			command = fmt.Sprintf(`set -o pipefail
touch %s
INSTRUCTION="$(cat "$GH_AW_PROMPT")"
mkdir -p "$CODEX_HOME/logs"
%s %sexec%s%s%s%s"$INSTRUCTION" 2>&1 | tee %s`, AgentStepSummaryPath, commandName, modelParam, webSearchParam, webFetchParam, fullAutoParam, customArgsParam, logFile)
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
		"GH_AW_PROMPT":        "/tmp/gh-aw/aw-prompts/prompt.txt",
		// Tag the step as a GitHub AW agentic execution for discoverability by agents
		"GITHUB_AW":                    "true",
		"GH_AW_MCP_CONFIG":             "/tmp/gh-aw/mcp-config/config.toml",
		"CODEX_HOME":                   "/tmp/gh-aw/mcp-config",
		"RUST_LOG":                     "trace,hyper_util=info,mio=info,reqwest=info,os_info=info,codex_otel=warn,codex_core=debug,ocodex_exec=debug",
		"GH_AW_GITHUB_TOKEN":           effectiveGitHubToken,
		"GITHUB_PERSONAL_ACCESS_TOKEN": effectiveGitHubToken,                                     // Used by GitHub MCP server via env_vars
		"OPENAI_API_KEY":               "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}", // Fallback for CODEX_API_KEY
	}
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

	// In sandbox (AWF) mode, set git identity environment variables so the first git commit
	// succeeds inside the container. AWF's --env-all forwards these to the container, ensuring
	// git does not rely on the host-side ~/.gitconfig which is not visible in the sandbox.
	if firewallEnabled {
		maps.Copy(env, getGitIdentityEnvVars())
	}

	// Add GH_AW_STARTUP_TIMEOUT environment variable (in seconds) if startup-timeout is specified
	// Supports both literal integers and GitHub Actions expressions (e.g. "${{ inputs.startup-timeout }}")
	if workflowData.ToolsStartupTimeout != "" {
		env["GH_AW_STARTUP_TIMEOUT"] = workflowData.ToolsStartupTimeout
	}

	// Add GH_AW_TOOL_TIMEOUT environment variable (in seconds) if timeout is specified
	// Supports both literal integers and GitHub Actions expressions (e.g. "${{ inputs.tool-timeout }}")
	if workflowData.ToolsTimeout != "" {
		env["GH_AW_TOOL_TIMEOUT"] = workflowData.ToolsTimeout
	}

	// Set the model environment variable.
	// Codex has no native model env var, so model selection always goes through
	// GH_AW_MODEL_AGENT_CODEX / GH_AW_MODEL_DETECTION_CODEX with shell expansion.
	// When model is configured (static or GitHub Actions expression), set the env var directly.
	// When not configured, use the GitHub variable fallback so users can set a default.
	if modelConfigured {
		codexEngineLog.Printf("Setting %s env var for model: %s", modelEnvVar, workflowData.EngineConfig.Model)
		env[modelEnvVar] = workflowData.EngineConfig.Model
	} else {
		env[modelEnvVar] = fmt.Sprintf("${{ vars.%s || '' }}", modelEnvVar)
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Add custom environment variables from agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		codexEngineLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}

	// Add mcp-scripts secrets to env for passthrough to MCP servers
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpScriptsSecrets := collectMCPScriptsSecrets(workflowData.MCPScripts)
		for varName, secretExpr := range mcpScriptsSecrets {
			// Only add if not already in env
			if _, exists := env[varName]; !exists {
				env[varName] = secretExpr
			}
		}
	}

	// Generate the step for Codex execution
	stepName := "Execute Codex CLI"
	var stepLines []string

	stepLines = append(stepLines, "      - name: "+stepName)
	stepLines = append(stepLines, "        id: agentic_execution")

	// Filter environment variables to only include allowed secrets
	// This is a security measure to prevent exposing unnecessary secrets to the AWF container
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Format step with command and filtered environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// GetFirewallLogsCollectionStep returns the step for collecting firewall logs (before secret redaction).
// This method is part of the firewall integration interface. It returns an empty slice because
// firewall logs are written to a known location (/tmp/gh-aw/sandbox/firewall/logs/) and don't need
// a separate collection step. The method is still called from compiler_yaml_main_job.go to maintain
// consistent behavior with other engines that may need log collection steps.
func (e *CodexEngine) GetFirewallLogsCollectionStep(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{}
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
		// Create an updated Playwright config with the allowed tools
		playwrightConfig := &PlaywrightToolConfig{
			Version: toolsConfig.Playwright.Version,
			Args:    toolsConfig.Playwright.Args,
		}

		result.Playwright = playwrightConfig

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

	return result
}

// expandNeutralToolsToCodexToolsFromMap is a backward compatibility wrapper
// that accepts map[string]any instead of *ToolsConfig
func (e *CodexEngine) expandNeutralToolsToCodexToolsFromMap(tools map[string]any) map[string]any {
	toolsConfig, _ := ParseToolsConfig(tools)
	result := e.expandNeutralToolsToCodexTools(toolsConfig)
	return result.ToMap()
}

// renderShellEnvironmentPolicy generates the [shell_environment_policy] section for config.toml
// This controls which environment variables are passed through to MCP servers for security
func (e *CodexEngine) renderShellEnvironmentPolicy(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Collect all environment variables needed by MCP servers
	envVars := make(map[string]bool)

	// Always include core environment variables
	envVars["PATH"] = true
	envVars["HOME"] = true

	// Add CODEX_API_KEY for authentication
	envVars["CODEX_API_KEY"] = true
	envVars["OPENAI_API_KEY"] = true // Fallback for CODEX_API_KEY

	// Check each MCP tool for required environment variables
	for _, toolName := range mcpTools {
		switch toolName {
		case "github":
			// GitHub MCP server needs GITHUB_PERSONAL_ACCESS_TOKEN
			envVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = true
		case "agentic-workflows":
			// Agentic workflows MCP server needs GITHUB_TOKEN
			envVars["GITHUB_TOKEN"] = true
		case "safe-outputs":
			// Safe outputs MCP server needs several environment variables
			envVars["GH_AW_SAFE_OUTPUTS"] = true
			envVars["GH_AW_ASSETS_BRANCH"] = true
			envVars["GH_AW_ASSETS_MAX_SIZE_KB"] = true
			envVars["GH_AW_ASSETS_ALLOWED_EXTS"] = true
			envVars["GITHUB_REPOSITORY"] = true
			envVars["GITHUB_SERVER_URL"] = true
		default:
			// For custom MCP tools, check if they have env configuration
			if toolValue, ok := tools[toolName]; ok {
				if toolConfig, ok := toolValue.(map[string]any); ok {
					// Extract environment variable names from env configuration
					if env, hasEnv := toolConfig["env"].(map[string]any); hasEnv {
						for envKey := range env {
							envVars[envKey] = true
						}
					}
				}
			}
		}
	}

	// Sort environment variable names for consistent output
	var sortedEnvVars []string
	for envVar := range envVars {
		sortedEnvVars = append(sortedEnvVars, envVar)
	}
	sort.Strings(sortedEnvVars)

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

// RenderMCPConfig is implemented in codex_mcp.go

// renderCodexMCPConfig is implemented in codex_mcp.go

// ParseLogMetrics is implemented in codex_logs.go

// parseCodexToolCallsWithSequence is implemented in codex_logs.go

// updateMostRecentToolWithDuration is implemented in codex_logs.go

// extractCodexTokenUsage is implemented in codex_logs.go

// GetLogParserScriptId is implemented in codex_logs.go
