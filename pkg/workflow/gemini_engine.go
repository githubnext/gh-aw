package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var geminiLog = logger.New("workflow:gemini_engine")

// GeminiEngine represents the Google Gemini CLI agentic engine
type GeminiEngine struct {
	BaseEngine
}

func NewGeminiEngine() *GeminiEngine {
	return &GeminiEngine{
		BaseEngine: BaseEngine{
			id:                     "gemini",
			displayName:            "Google Gemini CLI",
			description:            "Google Gemini CLI with headless mode and LLM gateway support",
			experimental:           false,
			supportsToolsAllowlist: true,
			supportsMaxTurns:       false,
			supportsWebFetch:       false,
			supportsWebSearch:      false,
			supportsFirewall:       true, // Gemini supports network firewalling via AWF
			supportsPlugins:        false,
			supportsLLMGateway:     true, // Gemini supports LLM gateway on port 10003
		},
	}
}

// SupportsLLMGateway returns the LLM gateway port for Gemini engine
func (e *GeminiEngine) SupportsLLMGateway() int {
	return constants.GeminiLLMGatewayPort
}

// GetModelEnvVarName returns the native environment variable name that the Gemini CLI uses
// for model selection. Setting GEMINI_MODEL is equivalent to passing --model to the CLI.
func (e *GeminiEngine) GetModelEnvVarName() string {
	return constants.GeminiCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the Gemini engine
// This includes GEMINI_API_KEY and optionally MCP_GATEWAY_API_KEY
func (e *GeminiEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	geminiLog.Print("Collecting required secrets for Gemini engine")
	secrets := []string{"GEMINI_API_KEY"}

	// Add MCP gateway API key if MCP servers are present (gateway is always started with MCP servers)
	if HasMCPServers(workflowData) {
		geminiLog.Print("Adding MCP_GATEWAY_API_KEY secret")
		secrets = append(secrets, "MCP_GATEWAY_API_KEY")
	}

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

	// Add safe-inputs secret names
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		for varName := range safeInputsSecrets {
			secrets = append(secrets, varName)
		}
		if len(safeInputsSecrets) > 0 {
			geminiLog.Printf("Added %d safe-inputs secrets", len(safeInputsSecrets))
		}
	}

	return secrets
}

// GetSecretValidationStep returns the secret validation step for the Gemini engine.
// Returns an empty step if custom command is specified.
func (e *GeminiEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		geminiLog.Printf("Skipping secret validation step: custom command specified (%s)", workflowData.EngineConfig.Command)
		return GitHubActionStep{}
	}
	return GenerateMultiSecretValidationStep(
		[]string{"GEMINI_API_KEY"},
		"Gemini CLI",
		"https://geminicli.com/docs/get-started/authentication/",
		getEngineEnvOverrides(workflowData),
	)
}

func (e *GeminiEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	geminiLog.Printf("Generating installation steps for Gemini engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		geminiLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	var steps []GitHubActionStep

	// Define engine configuration for shared validation
	config := EngineInstallConfig{
		Secrets:         []string{"GEMINI_API_KEY"},
		DocsURL:         "https://geminicli.com/docs/get-started/authentication/",
		NpmPackage:      "@google/gemini-cli",
		Version:         string(constants.DefaultGeminiVersion),
		Name:            "Gemini CLI",
		CliName:         "gemini",
		InstallStepName: "Install Gemini CLI",
	}

	// Secret validation step is now generated in the activation job (GetSecretValidationStep).

	// Determine Gemini version
	geminiVersion := config.Version
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		geminiVersion = workflowData.EngineConfig.Version
	}

	// Add Node.js setup step first (before sandbox installation)
	npmSteps := GenerateNpmInstallSteps(
		config.NpmPackage,
		geminiVersion,
		config.InstallStepName,
		config.CliName,
		true, // Include Node.js setup
	)

	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Setup Node.js step
	}

	// Add AWF installation if firewall is enabled
	if isFirewallEnabled(workflowData) {
		// Install AWF after Node.js setup but before Gemini CLI installation
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

	// Add Gemini CLI installation step after sandbox installation
	if len(npmSteps) > 1 {
		steps = append(steps, npmSteps[1:]...) // Install Gemini CLI and subsequent steps
	}

	return steps
}

// GetDeclaredOutputFiles returns the output files that Gemini may produce.
// Gemini CLI writes structured error reports to /tmp/gemini-client-error-*.json
// with a timestamp in the filename (e.g. gemini-client-error-Turn.run-sendMessageStream-2026-02-21T20-45-59-824Z.json).
// These files provide detailed diagnostics when the Gemini API call fails.
func (e *GeminiEngine) GetDeclaredOutputFiles() []string {
	return []string{
		"/tmp/gemini-client-error-*.json",
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
	var geminiArgs []string

	// Model is passed via the native GEMINI_MODEL environment variable only when explicitly
	// configured. When not configured, the Gemini CLI uses its built-in default model.
	// This avoids embedding the value directly in the shell command (which fails template injection
	// validation for GitHub Actions expressions like ${{ inputs.model }}).
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Gemini CLI reads MCP config from .gemini/settings.json (project-level)
	// The conversion script (convert_gateway_config_gemini.sh) writes settings.json
	// during the MCP setup step, so no --mcp-config flag is needed here.

	// Auto-approve all tool executions (equivalent to Codex's --dangerously-bypass-approvals-and-sandbox)
	// Without this, Gemini CLI's default approval mode rejects tool calls with "Tool execution denied by policy"
	geminiArgs = append(geminiArgs, "--yolo")

	// Add streaming JSON output (JSONL format, compatible with the log parser)
	geminiArgs = append(geminiArgs, "--output-format", "stream-json")

	// Add prompt argument
	geminiArgs = append(geminiArgs, "--prompt", "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\"")

	// Build the command
	commandName := "gemini"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}

	geminiCommand := fmt.Sprintf("%s %s", commandName, shellJoinArgs(geminiArgs))

	// Build the full command with AWF wrapping if enabled
	var command string
	firewallEnabled := isFirewallEnabled(workflowData)
	if firewallEnabled {
		allowedDomains := GetGeminiAllowedDomainsWithToolsAndRuntimes(
			workflowData.NetworkPermissions,
			workflowData.Tools,
			workflowData.Runtimes,
		)

		npmPathSetup := GetNpmBinPathSetup()
		geminiCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, geminiCommand)

		// Enable API proxy sidecar if this engine supports LLM gateway
		llmGatewayPort := e.SupportsLLMGateway()
		usesAPIProxy := llmGatewayPort > 0

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "gemini",
			EngineCommand:  geminiCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			UsesAPIProxy:   usesAPIProxy,
			AllowedDomains: allowedDomains,
		})
	} else {
		command = fmt.Sprintf(`set -o pipefail
%s 2>&1 | tee -a %s`, geminiCommand, logFile)
	}

	// Build environment variables
	env := map[string]string{
		"GEMINI_API_KEY":   "${{ secrets.GEMINI_API_KEY }}",
		"GH_AW_PROMPT":     "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
		// Enable verbose debug logging from Gemini CLI for better diagnostics.
		// Gemini CLI uses the npm 'debug' package, and 'gemini-cli:*' enables all
		// internal Gemini CLI debug channels (see: https://gemini-cli-docs.pages.dev/cli/configuration).
		// Non-JSON debug lines are gracefully skipped by ParseLogMetrics.
		"DEBUG": "gemini-cli:*",
	}

	// Add MCP config env var if needed (points to .gemini/settings.json for Gemini)
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/.gemini/settings.json"
	}

	// When the firewall (AWF) is enabled with --enable-api-proxy, point Gemini CLI at the
	// LLM gateway sidecar instead of the real googleapis.com endpoint.
	if firewallEnabled {
		env["GEMINI_API_BASE_URL"] = fmt.Sprintf("http://host.docker.internal:%d", constants.GeminiLLMGatewayPort)
	}

	// Add safe outputs env
	applySafeOutputEnvToMap(env, workflowData)

	// Set the model environment variable only when explicitly configured.
	// When model is configured, use the native GEMINI_MODEL env var - the Gemini CLI reads it
	// directly, avoiding the need to embed the value in the shell command (which would fail
	// template injection validation for GitHub Actions expressions like ${{ inputs.model }}).
	// When model is not configured, let the Gemini CLI use its built-in default model.
	if modelConfigured {
		geminiLog.Printf("Setting %s env var for model: %s", constants.GeminiCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.GeminiCLIModelEnvVar] = workflowData.EngineConfig.Model
	}

	// Add custom environment variables from engine config.
	// This allows users to override the default engine token expression (e.g.
	// GEMINI_API_KEY: ${{ secrets.MY_ORG_GEMINI_KEY }}) via engine.env.
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Add custom environment variables from agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		geminiLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}

	// Generate the execution step
	stepLines := []string{
		"      - name: Execute Gemini CLI",
		"        id: agentic_execution",
	}

	// Filter environment variables for security
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Format step with command and env
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))
	return steps
}
