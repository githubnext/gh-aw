package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var auggieLog = logger.New("workflow:auggie_engine")

// AuggieEngine represents the Augment Code Auggie CLI agentic engine.
type AuggieEngine struct {
	BaseEngine
}

func NewAuggieEngine() *AuggieEngine {
	return &AuggieEngine{
		BaseEngine: BaseEngine{
			id:           "auggie",
			displayName:  "Auggie CLI",
			description:  "Augment Code Auggie CLI (experimental)",
			experimental: true,
			capabilities: EngineCapabilities{
				ToolsAllowlist:   false,
				MaxTurns:         false,
				WebSearch:        true,
				MaxContinuations: false,
				NativeAgentFile:  false,
			},
		},
	}
}

// GetRequiredSecretNames returns the list of secrets required by the Auggie engine.
func (e *AuggieEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	secrets := []string{constants.AuggieSessionAuthEnvVar}
	secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)

	parsedTools, tools := extractToolsConfig(workflowData)
	if hasGitHubTool(parsedTools) {
		secrets = append(secrets, "GITHUB_MCP_SERVER_TOKEN")
	}
	for varName := range collectHTTPMCPHeaderSecrets(tools) {
		secrets = append(secrets, varName)
	}

	return secrets
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the Auggie engine supports.
func (e *AuggieEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.AuggieSessionAuthEnvVar,
	}
}

// GetSecretValidationStep returns the secret validation step for the Auggie engine.
func (e *AuggieEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{constants.AuggieSessionAuthEnvVar},
		"Auggie CLI",
		"https://docs.augmentcode.com/cli/overview",
	)
}

func (e *AuggieEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	auggieLog.Printf("Generating installation steps for Auggie engine: workflow=%s", workflowData.Name)

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		auggieLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	version := string(constants.DefaultAuggieVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	npmSteps := GenerateNpmInstallSteps(
		"@augmentcode/auggie",
		version,
		"Install Auggie CLI",
		"auggie",
		true,
		false,
		false,
	)
	npmSteps = append(npmSteps, GitHubActionStep{
		"      - name: Verify Auggie CLI installation",
		"        run: auggie --version",
	})
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

func (e *AuggieEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

func (e *AuggieEngine) GetAgentManifestFiles() []string {
	return []string{"AGENTS.md"}
}

func (e *AuggieEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".augment/"}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Auggie.
func (e *AuggieEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	auggieLog.Printf("Generating execution steps for Auggie engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	commandName := "auggie"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}

	modelEnvVar := constants.EnvVarModelAgentAuggie
	if workflowData.IsDetectionRun {
		modelEnvVar = constants.EnvVarModelDetectionAuggie
	}
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	baseArgs := shellJoinArgs([]string{"--print", "--quiet"})
	extraArgs := ""
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		extraArgs = " " + shellJoinArgs(workflowData.EngineConfig.Args)
	}
	modelArg := fmt.Sprintf(` ${%s:+--model "$%s"}`, modelEnvVar, modelEnvVar)
	mcpConfigArg := ""
	if HasMCPServers(workflowData) {
		mcpConfigArg = ` --mcp-config "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json"`
	}

	engineCommand := fmt.Sprintf(`%s %s%s%s%s "$(cat %s)"`,
		commandName,
		baseArgs,
		modelArg,
		extraArgs,
		mcpConfigArg,
		constants.AwPromptsFile,
	)
	engineCommand = getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + engineCommand

	firewallEnabled := isFirewallEnabled(workflowData)
	var command string
	if firewallEnabled {
		allowedDomains := GetAllowedDomainsForEngine(constants.AuggieEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
		engineCommandWithPath := fmt.Sprintf("%s && %s", GetNpmBinPathSetup(), engineCommand)
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			engineCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, engineCommandWithPath)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "auggie",
			EngineCommand:  engineCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			AllowedDomains: allowedDomains,
			PathSetup:      "touch " + AgentStepSummaryPath,
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(
				workflowData,
				[]string{constants.AuggieSessionAuthEnvVar},
			),
		})
	} else {
		command = fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s 2>&1 | tee -a %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, engineCommand, logFile)
	}

	env := map[string]string{
		constants.AuggieSessionAuthEnvVar: "${{ secrets." + constants.AuggieSessionAuthEnvVar + " }}",
		"GH_AW_PROMPT":                    constants.AwPromptsFile,
		"GITHUB_AW":                       "true",
		"GITHUB_STEP_SUMMARY":             AgentStepSummaryPath,
		"GITHUB_WORKSPACE":                "${{ github.workspace }}",
		"RUNNER_TEMP":                     "${{ runner.temp }}",
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)
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
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = constants.McpServersJsonPathExpr
	}
	if firewallEnabled {
		maps.Copy(env, getGitIdentityEnvVars())
	}

	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)
	applyOptionalEngineToolTimeouts(env, workflowData)
	applyEngineMaxTurnsEnv(env, workflowData)

	if modelConfigured {
		env[modelEnvVar] = workflowData.EngineConfig.Model
	} else {
		env[modelEnvVar] = fmt.Sprintf("${{ vars.%s || '' }}", modelEnvVar)
	}

	applyEngineCwdEnv(env, workflowData)
	applyEngineAndAgentEnv(env, workflowData, auggieLog)
	applyMCPScriptsSecretEnv(env, workflowData)

	stepLines := []string{
		"      - name: Execute Auggie CLI",
		"        id: agentic_execution",
	}
	filteredEnv := FilterEnvForSecrets(env, e.GetRequiredSecretNames(workflowData))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)
	return []GitHubActionStep{stepLines}
}

var _ CodingAgentEngine = (*AuggieEngine)(nil)
