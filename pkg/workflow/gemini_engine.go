package workflow

import (
	"fmt"

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
			experimental:           true, // Marked as experimental as requested
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

	// Add secret validation step
	secretValidation := GenerateMultiSecretValidationStep(
		config.Secrets,
		config.Name,
		config.DocsURL,
	)
	steps = append(steps, secretValidation)

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

// GetDeclaredOutputFiles returns the output files that Gemini may produce
func (e *GeminiEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Gemini
func (e *GeminiEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	geminiLog.Printf("Generating execution steps for Gemini engine: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	// Handle custom steps if they exist in engine config
	steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)

	// Build gemini CLI arguments based on configuration
	var geminiArgs []string

	// Add model if specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		geminiArgs = append(geminiArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add MCP config if servers are present
	if HasMCPServers(workflowData) {
		geminiArgs = append(geminiArgs, "--mcp-config", "/tmp/gh-aw/mcp-config/mcp-servers.json")
	}

	// Add headless mode with JSON output
	geminiArgs = append(geminiArgs, "--output-format", "json")

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

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "gemini",
			EngineCommand:  geminiCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			UsesAPIProxy:   false,
			AllowedDomains: allowedDomains,
		})
	} else {
		command = fmt.Sprintf(`set -o pipefail
%s 2>&1 | tee %s`, geminiCommand, logFile)
	}

	// Build environment variables
	env := map[string]string{
		"GEMINI_API_KEY":   "${{ secrets.GEMINI_API_KEY }}",
		"GH_AW_PROMPT":     "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
	}

	// Add MCP config env var if needed
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "/tmp/gh-aw/mcp-config/mcp-servers.json"
	}

	// Add safe outputs env
	applySafeOutputEnvToMap(env, workflowData)

	// Add model env var if not explicitly configured
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
	if !modelConfigured {
		isDetectionJob := workflowData.SafeOutputs == nil
		if isDetectionJob {
			env[constants.EnvVarModelDetectionGemini] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelDetectionGemini)
		} else {
			env[constants.EnvVarModelAgentGemini] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelAgentGemini)
		}
	}

	// Generate the execution step
	stepLines := []string{
		"      - name: Run Gemini",
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

// convertStepToYAML converts a single step to YAML format
func (e *GeminiEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}
