package workflow

// This file provides the shared base for Google CLI-based agentic engines
// (Gemini and Antigravity). The two engines share the same execution model,
// settings-file format, MCP config approach, and log-parsing strategy — they
// differ only in engine-specific constants (binary name, API-key secret name,
// config directory, CLI flags, etc.).
//
// googleCLIEngine embeds BaseEngine and holds a googleCLIEngineConfig that
// carries every per-engine constant. GeminiEngine and AntigravityEngine embed
// googleCLIEngine and override only GetInstallationSteps (which are completely
// different — npm package vs. GCS binary installer).
//
// Shared utility functions:
//   - appendBashTools       – maps bash neutral config to run_shell_command entries
//   - computeGoogleCLIToolsCore – full neutral→CLI tool name mapping
//
// computeGeminiToolsCore and computeAntigravityToolsCore in their respective
// files are thin wrappers that delegate here so existing tests remain unchanged.

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

// googleCLIEngineConfig holds all per-engine constants for Google CLI-based engines.
// Set every field in NewGeminiEngine / NewAntigravityEngine.
type googleCLIEngineConfig struct {
	// log is the engine-instance logger (e.g. "workflow:gemini_engine").
	log *logger.Logger

	// apiKeySecretName is the GitHub Actions secret name for the engine API key
	// (e.g. "GEMINI_API_KEY" or "ANTIGRAVITY_API_KEY").
	apiKeySecretName string

	// apiBaseURLEnvVar is the environment variable name used to override the
	// API base URL when the LLM gateway proxy is active
	// (e.g. "GEMINI_API_BASE_URL" or "ANTIGRAVITY_API_BASE_URL").
	apiBaseURLEnvVar string

	// modelEnvVar is the native CLI environment variable for model selection
	// (e.g. "GEMINI_MODEL" or "ANTIGRAVITY_MODEL").
	modelEnvVar string

	// trustWorkspaceEnvVar is set to "true" to prevent CLI v1.x from overriding
	// --yolo with "default" approval mode for untrusted workspaces
	// (e.g. "GEMINI_CLI_TRUST_WORKSPACE" or "ANTIGRAVITY_CLI_TRUST_WORKSPACE").
	trustWorkspaceEnvVar string

	// debugEnvValue is the value of the DEBUG environment variable passed to the CLI
	// to enable verbose internal debug channels (e.g. "gemini-cli:*").
	debugEnvValue string

	// defaultCLIBinary is the executable name when no engine.command override is set
	// (e.g. "gemini" or "agy").
	defaultCLIBinary string

	// cliArgs are the pre-prompt CLI arguments appended before --prompt
	// (e.g. ["--yolo", "--skip-trust", "--output-format", "stream-json"] for Gemini).
	cliArgs []string

	// configDir is the project-level config directory relative to GITHUB_WORKSPACE
	// (e.g. ".gemini" or ".antigravity").
	configDir string

	// baseConfigEnvVar is the environment variable name used to pass the JSON
	// settings blob into the settings-write step
	// (e.g. "GH_AW_GEMINI_BASE_CONFIG" or "GH_AW_ANTIGRAVITY_BASE_CONFIG").
	baseConfigEnvVar string

	// secretValidationURL is the documentation URL shown in the secret validation step.
	secretValidationURL string

	// secretValidationLabel is the engine name shown in the secret validation step
	// message (e.g. "Gemini CLI", "Antigravity CLI"). This may differ from
	// GetDisplayName() which carries the full product name.
	secretValidationLabel string

	// configStepName is the human-readable name for the settings-file write step
	// (e.g. "Write Gemini Config" or "Write Antigravity Config").
	configStepName string

	// executionStepName is the human-readable name for the CLI execution step
	// (e.g. "Execute Gemini CLI" or "Execute Antigravity CLI").
	executionStepName string

	// errorMoveStepName is the human-readable name for the pre-bundle error-file
	// relocation step (e.g. "Move Gemini error files to artifact directory").
	errorMoveStepName string

	// errorFileSrcGlob is the glob pattern for error files produced by the CLI
	// that must be moved into /tmp/gh-aw/ before the artifact upload
	// (e.g. "/tmp/gemini-client-error-*.json").
	errorFileSrcGlob string

	// errorFileDstGlob is the declared output file glob under /tmp/gh-aw/ used
	// in GetDeclaredOutputFiles (e.g. constants.TmpGeminiClientErrorGlob).
	errorFileDstGlob string

	// agentManifestFiles are the engine-specific instruction files treated as
	// security-sensitive manifests (e.g. ["GEMINI.md", "AGENTS.md"]).
	agentManifestFiles []string

	// agentManifestPrefixes are the engine-specific config directory prefixes
	// (e.g. [".gemini/"]).
	agentManifestPrefixes []string

	// logParserScriptID is the JavaScript script ID for log parsing
	// (e.g. "parse_gemini_log" or "parse_antigravity_log").
	logParserScriptID string

	// logParserEngineName is the display name passed to parseStatsJSONLMetrics
	// (e.g. "Gemini" or "Antigravity").
	logParserEngineName string

	// excludeAPIKeys lists the API-key env var names passed to
	// ComputeAWFExcludeEnvVarNames so they are stripped from the sandboxed env
	// (e.g. ["GEMINI_API_KEY"] for Gemini, ["ANTIGRAVITY_API_KEY", "GEMINI_API_KEY"] for Antigravity).
	excludeAPIKeys []string

	// extraAllowedSecrets are additional secrets prepended to the allowed-secrets
	// list passed to FilterEnvForSecrets. Antigravity needs "GEMINI_API_KEY" here
	// because it mirrors the Antigravity key into GEMINI_API_KEY for the proxy.
	extraAllowedSecrets []string

	// mirrorAPIKeyAs, when non-empty, causes the value of apiKeySecretName to be
	// copied into this env var if not already set. Antigravity sets this to
	// "GEMINI_API_KEY" so the Gemini proxy sidecar receives a valid key.
	mirrorAPIKeyAs string
}

// googleCLIEngine is the shared embeddable base for Gemini and Antigravity engines.
type googleCLIEngine struct {
	BaseEngine
	cfg googleCLIEngineConfig
}

// ── Shared utility functions ────────────────────────────────────────────────

// appendBashTools maps the bash neutral tool configuration to run_shell_command
// entries and appends them to toolsCore. log is used for debug messages.
//
// A single pass over bashCommands is used so that a wildcard found anywhere in
// the list (even after specific commands) causes only "run_shell_command" to be
// returned and any pre-wildcard specific entries are discarded. This preserves
// the semantics of "any wildcard means allow all shell commands".
func appendBashTools(toolsCore []string, bashConfig any, log *logger.Logger) []string {
	bashCommands, ok := bashConfig.([]any)
	if !ok || len(bashCommands) == 0 {
		// bash with no specific commands – allow all shell commands
		log.Print("bash (no specific commands) → run_shell_command")
		return append(toolsCore, "run_shell_command")
	}

	// Single pass: accumulate per-command entries in specific; return early on wildcard.
	var specific []string
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}
		if cmdStr == "*" || cmdStr == ":*" {
			log.Print("bash wildcard → run_shell_command")
			return append(toolsCore, "run_shell_command")
		}
		// Normalize trailing " *" wildcard (e.g. "jq *" → "jq") so that all
		// engines emit the canonical prefix form (run_shell_command(jq)).
		normalized, _ := normalizeBashCommand(cmdStr)
		entry := fmt.Sprintf("run_shell_command(%s)", normalized)
		log.Printf("bash %q → %s", cmdStr, entry)
		specific = append(specific, entry)
	}
	return append(toolsCore, specific...)
}

// computeGoogleCLIToolsCore maps neutral tool names to Google CLI built-in tool
// names for the tools.core allowlist in the engine settings file.
//
// Neutral tool → Google CLI tool mapping:
//   - bash: [cmd, ...]     → run_shell_command(cmd), ... (one entry per command)
//   - bash: * or bash: nil → run_shell_command           (allow all shell commands)
//   - edit: {}             → replace, write_file          (file write tools)
//   - web-fetch: {}        → web_fetch
//
// Read-only file system tools are always included as they are essential for
// agentic workflows: glob, grep_search, list_directory, read_file, read_many_files.
func computeGoogleCLIToolsCore(tools map[string]any, log *logger.Logger) []string {
	// Always include essential read-only file system tools.
	toolsCore := []string{
		"glob",
		"grep_search",
		"list_directory",
		"read_file",
		"read_many_files",
	}

	if tools == nil {
		return toolsCore
	}

	// Map bash neutral tool to run_shell_command.
	if bashConfig, hasBash := tools["bash"]; hasBash {
		toolsCore = appendBashTools(toolsCore, bashConfig, log)
	}

	// Map edit neutral tool to write_file and replace (file write tools).
	if _, hasEdit := tools["edit"]; hasEdit {
		log.Print("edit → replace, write_file")
		toolsCore = append(toolsCore, "replace")
		toolsCore = append(toolsCore, "write_file")
	}

	// Map web-fetch neutral tool to the native web_fetch tool.
	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		log.Print("web-fetch → web_fetch")
		toolsCore = append(toolsCore, "web_fetch")
	}

	sort.Strings(toolsCore)
	return toolsCore
}

// ── googleCLIEngine method implementations ───────────────────────────────────

// GetModelEnvVarName returns the native CLI environment variable for model selection.
func (e *googleCLIEngine) GetModelEnvVarName() string {
	return e.cfg.modelEnvVar
}

// GetSupportedEnvVarKeys returns the engine.env variable names this engine supports.
func (e *googleCLIEngine) GetSupportedEnvVarKeys() []string {
	return []string{e.cfg.apiKeySecretName}
}

// GetRequiredSecretNames returns the list of secrets required by the engine.
func (e *googleCLIEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	e.cfg.log.Printf("Collecting required secrets for %s engine", e.GetDisplayName())
	secrets := []string{e.cfg.apiKeySecretName}

	// Add common MCP secrets (MCP_GATEWAY_API_KEY if MCP servers present, mcp-scripts secrets).
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)

	// Add GitHub token for GitHub MCP server if present.
	if hasGitHubTool(workflowData.ParsedTools) {
		e.cfg.log.Print("Adding GITHUB_MCP_SERVER_TOKEN secret")
		secrets = append(secrets, "GITHUB_MCP_SERVER_TOKEN")
	}

	// Add HTTP MCP header secret names.
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName := range headerSecrets {
		secrets = append(secrets, varName)
	}
	if len(headerSecrets) > 0 {
		e.cfg.log.Printf("Added %d HTTP MCP header secrets", len(headerSecrets))
	}

	return secrets
}

// GetSecretValidationStep returns the secret validation step for the engine.
// Returns an empty step if a custom command is specified.
func (e *googleCLIEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{e.cfg.apiKeySecretName},
		e.cfg.secretValidationLabel,
		e.cfg.secretValidationURL,
	)
}

// GetDeclaredOutputFiles returns the output files the engine may produce.
func (e *googleCLIEngine) GetDeclaredOutputFiles() []string {
	return []string{e.cfg.errorFileDstGlob}
}

// GetAgentManifestFiles returns engine-specific instruction files treated as
// security-sensitive manifests.
func (e *googleCLIEngine) GetAgentManifestFiles() []string {
	return e.cfg.agentManifestFiles
}

// GetAgentManifestPathPrefixes returns engine-specific config directory prefixes.
func (e *googleCLIEngine) GetAgentManifestPathPrefixes() []string {
	return e.cfg.agentManifestPrefixes
}

// GetPreBundleSteps returns a step that moves CLI error reports from /tmp/ into
// /tmp/gh-aw/ before the unified artifact upload.
func (e *googleCLIEngine) GetPreBundleSteps(workflowData *WorkflowData) []GitHubActionStep {
	return []GitHubActionStep{
		{
			"      - name: " + e.cfg.errorMoveStepName,
			"        if: always()",
			fmt.Sprintf("        run: mv %s /tmp/gh-aw/ 2>/dev/null || true", e.cfg.errorFileSrcGlob),
		},
	}
}

// RenderMCPConfig renders MCP server configuration for the engine.
// Both Gemini and Antigravity use the JSON format without Copilot-specific fields.
func (e *googleCLIEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	e.cfg.log.Printf("Rendering MCP config for %s: tool_count=%d, mcp_tool_count=%d",
		e.GetDisplayName(), len(tools), len(mcpTools))
	return renderDefaultJSONMCPConfig(yaml, tools, mcpTools, workflowData, constants.ShellMcpServersJsonPath)
}

// ParseLogMetrics parses CLI log output and extracts metrics.
func (e *googleCLIEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	return parseStatsJSONLMetrics(logContent, verbose, e.cfg.logParserEngineName, e.cfg.log)
}

// GetLogParserScriptId returns the script ID for parsing engine logs.
func (e *googleCLIEngine) GetLogParserScriptId() string {
	return e.cfg.logParserScriptID
}

// generateSettingsStep creates a GitHub Actions step that writes or merges the
// engine's project settings file (e.g. .gemini/settings.json) before execution.
//
// This step:
//  1. Sets context.includeDirectories to ["/tmp/"] so that file-system tools
//     (write_file, replace) can access files in /tmp/ including /tmp/gh-aw/.
//  2. Sets tools.core to the built-in tool list derived from the workflow's
//     neutral tool configuration.
//  3. Merges the above settings with any existing settings.json (written by
//     the MCP gateway setup script), preserving mcpServers config.
func (e *googleCLIEngine) generateSettingsStep(workflowData *WorkflowData) GitHubActionStep {
	e.cfg.log.Printf("Generating %s settings step for: %s", e.GetDisplayName(), workflowData.Name)

	tools := workflowData.Tools
	if tools == nil {
		tools = make(map[string]any)
	}
	workflowDataWithEffectiveTools := *workflowData
	workflowDataWithEffectiveTools.Tools = tools
	tools = withMountedCLIShellCommandsInRestrictedBash(&workflowDataWithEffectiveTools)

	// Compute tools.core from neutral tool configuration.
	toolsCore := computeGoogleCLIToolsCore(tools, e.cfg.log)
	e.cfg.log.Printf("tools.core entries: %d", len(toolsCore))

	// Build the settings JSON object.
	config := map[string]any{
		"context": map[string]any{
			"includeDirectories": []string{"/tmp/"},
		},
		"tools": map[string]any{
			"core": toolsCore,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		e.cfg.log.Printf("ERROR: Failed to marshal %s settings: %v", e.GetDisplayName(), err)
		configJSON = []byte(`{"context":{"includeDirectories":["/tmp/"]},"tools":{"core":[]}}`)
	}

	// Generate a shell script that:
	// - Creates the engine config directory if needed.
	// - Merges settings into an existing settings.json (from MCP gateway setup), or
	// - Creates a new settings.json when no MCP servers are configured.
	//
	// The JSON config is passed via an environment variable to avoid shell quoting
	// issues with special characters in the JSON.
	//
	// jq merge: '$existing * $base' means the RIGHT operand ($base) overrides the
	// LEFT operand ($existing) for conflicting keys. Non-conflicting keys from
	// $existing (e.g. mcpServers) are preserved.
	command := fmt.Sprintf(
		`mkdir -p "$GITHUB_WORKSPACE/%[1]s"
SETTINGS="$GITHUB_WORKSPACE/%[1]s/settings.json"
BASE_CONFIG="$%[2]s"
if [ -f "$SETTINGS" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$SETTINGS")" '$existing * $base')
  echo "$MERGED" > "$SETTINGS"
else
  echo "$BASE_CONFIG" > "$SETTINGS"
fi`,
		e.cfg.configDir, e.cfg.baseConfigEnvVar,
	)

	stepLines := []string{
		"      - name: " + e.cfg.configStepName,
	}
	env := map[string]string{
		e.cfg.baseConfigEnvVar: string(configJSON),
	}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)
	return GitHubActionStep(stepLines)
}

// GetExecutionSteps returns the GitHub Actions steps for executing the CLI engine.
func (e *googleCLIEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	e.cfg.log.Printf("Generating execution steps for %s engine: workflow=%s, firewall=%v",
		e.GetDisplayName(), workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Write the engine settings file with context.includeDirectories and tools.core.
	// This step runs after the MCP gateway setup (which may have written mcpServers
	// config) and merges the context/tools settings into any existing settings.json.
	settingsStep := e.generateSettingsStep(workflowData)
	steps = append(steps, settingsStep)

	// Model is passed via the native model env var when explicitly configured to
	// avoid embedding the value in the shell command (which fails template injection
	// validation for GitHub Actions expressions like ${{ inputs.model }}).
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Build CLI arguments.
	cliArgs := make([]string, len(e.cfg.cliArgs))
	copy(cliArgs, e.cfg.cliArgs)

	// Build the command name.
	commandName := e.cfg.defaultCLIBinary
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}

	// Append the prompt arg raw (not through shellJoinArgs) to preserve shell expansion.
	cliCommand := fmt.Sprintf(`%s %s --prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`,
		commandName, shellJoinArgs(cliArgs))
	cliCommand = getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + cliCommand

	// Build the full command with AWF wrapping if the firewall is enabled.
	var command string
	firewallEnabled := isFirewallEnabled(workflowData)
	if firewallEnabled {
		// Get allowed domains: prefer the pre-warmed cache on WorkflowData to avoid
		// re-running the expensive map+sort operation.
		var allowedDomains string
		if workflowData.CachedAllowedDomainsComputed {
			allowedDomains = workflowData.CachedAllowedDomainsStr
		} else {
			allowedDomains = GetAllowedDomainsForEngine(
				constants.EngineName(e.GetID()),
				workflowData.NetworkPermissions,
				workflowData.Tools,
				workflowData.Runtimes,
			)
		}
		// Add GHES/custom API target domains to the firewall allow-list when
		// engine.api-target is set.
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
		}

		npmPathSetup := GetNpmBinPathSetup()
		cliCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, cliCommand)
		// Add MCP CLI bin directory to PATH when cli-proxy is enabled.
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			cliCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, cliCommandWithPath)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     e.GetID(),
			EngineCommand:  cliCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			AllowedDomains: allowedDomains,
			// Create the agent step summary file before AWF starts so it is accessible
			// inside the sandbox.
			PathSetup: "touch " + AgentStepSummaryPath,
			// Exclude every env var whose step-env value is a secret so the agent
			// cannot read raw token values via bash tools (env / printenv).
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, e.cfg.excludeAPIKeys),
		})
	} else {
		command = fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`,
			AgentCLIStartMsPath, AgentStepSummaryPath, logFile, cliCommand, logFile)
	}

	// Build environment variables.
	env := map[string]string{
		e.cfg.apiKeySecretName: fmt.Sprintf("${{ secrets.%s }}", e.cfg.apiKeySecretName),
		"GH_AW_PROMPT":         constants.AwPromptsFile,
		// Tag the step as a GitHub AW agentic execution for discoverability by agents.
		"GITHUB_AW":        "true",
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
		"RUNNER_TEMP":      "${{ runner.temp }}",
		// Override GITHUB_STEP_SUMMARY with a path accessible inside the sandbox.
		// We create this file before the agent starts and append it to the real
		// $GITHUB_STEP_SUMMARY after secret redaction.
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
		// Enable verbose debug logging from the CLI.
		"DEBUG": e.cfg.debugEnvValue,
		// Trust the workspace to prevent CLI v1.x from overriding --yolo to default
		// approval mode when the workspace is untrusted (exit code 55).
		e.cfg.trustWorkspaceEnvVar: "true",
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
	// Indicate the phase: "agent" for the main run, "detection" for threat detection.
	// Include the compiler version so agents can identify which gh-aw version generated the workflow.
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

	// Add MCP config env var if needed.
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = fmt.Sprintf("${{ github.workspace }}/%s/settings.json", e.cfg.configDir)
	}

	// When the firewall (AWF) is enabled with --enable-api-proxy, point the CLI at
	// the LLM gateway sidecar instead of the real googleapis.com endpoint.
	if firewallEnabled {
		env[e.cfg.apiBaseURLEnvVar] = fmt.Sprintf("http://host.docker.internal:%d", e.getDedicatedLLMGatewayPort())

		// Set git identity environment variables so the first git commit succeeds
		// inside the container.
		maps.Copy(env, getGitIdentityEnvVars())
	}

	// Add safe outputs env.
	applySafeOutputEnvToMap(env, workflowData)

	// Propagate W3C trace context so engine spans nest under the gh-aw.agent.setup span.
	applyTraceContextEnvToMap(env)

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	} else {
		env["GH_AW_MAX_TURNS"] = compilerenv.BuildDefaultMaxTurnsExpression()
	}

	// Set the model environment variable only when explicitly configured.
	if modelConfigured {
		e.cfg.log.Printf("Setting %s env var for model: %s", e.cfg.modelEnvVar, workflowData.EngineConfig.Model)
		env[e.cfg.modelEnvVar] = workflowData.EngineConfig.Model
	}

	// Add custom environment variables from engine config.
	applyEngineCwdEnv(env, workflowData)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Add custom environment variables from agent config.
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		e.cfg.log.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}

	// Mirror the primary API key into a secondary env var if configured.
	// This runs after all env overrides so the mirror tracks the effective key value.
	// Antigravity uses this to copy ANTIGRAVITY_API_KEY → GEMINI_API_KEY so the
	// Gemini proxy sidecar can authenticate without requiring users to duplicate secrets.
	if e.cfg.mirrorAPIKeyAs != "" {
		if _, alreadySet := env[e.cfg.mirrorAPIKeyAs]; !alreadySet {
			env[e.cfg.mirrorAPIKeyAs] = env[e.cfg.apiKeySecretName]
		}
	}

	// Generate the execution step.
	stepLines := []string{
		"      - name: " + e.cfg.executionStepName,
		"        id: agentic_execution",
	}

	// Build the allowed-secrets list. extraAllowedSecrets are prepended so that
	// FilterEnvForSecrets keeps those secrets even if GetRequiredSecretNames does
	// not include them (e.g. Antigravity includes GEMINI_API_KEY here).
	requiredSecrets := e.GetRequiredSecretNames(workflowData)
	allowedSecrets := make([]string, 0, len(e.cfg.extraAllowedSecrets)+len(requiredSecrets))
	allowedSecrets = append(allowedSecrets, e.cfg.extraAllowedSecrets...)
	allowedSecrets = append(allowedSecrets, requiredSecrets...)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Inject GH_TOKEN for CLI proxy (added after filtering since it uses a special
	// fallback expression that is always allowed when cli-proxy is enabled).
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)

	// Format step with command and env.
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))
	return steps
}
