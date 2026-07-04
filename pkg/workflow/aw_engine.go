package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var awLog = logger.New("workflow:aw_engine")

// AWHarnessScriptName is the filename of the AW Harness script in the setup actions directory.
const AWHarnessScriptName = "aw_harness.cjs"

// AWStreamingLogFile is the path where AW Harness emits structured JSONL events.
const AWStreamingLogFile = "/tmp/gh-aw/aw-streaming.jsonl"

// AWConfigFile is the path where the compiler writes the harness config.json.
const AWConfigFile = "/tmp/gh-aw/aw-config/config.json"

// AWEngine represents the AW Harness engine — a Pi-SDK-based coding agent backed
// by aw_harness.cjs (see specs/aw-harness.md).
type AWEngine struct {
	BaseEngine
}

// NewAWEngine creates a new AWEngine with its default capabilities.
func NewAWEngine() *AWEngine {
	return &AWEngine{
		BaseEngine: BaseEngine{
			id:           "aw",
			displayName:  "AW Harness",
			description:  "AW Harness — Pi-SDK-based coding agent (experimental, see specs/aw-harness.md)",
			experimental: true,
			capabilities: EngineCapabilities{
				ToolsAllowlist:   false,
				MaxTurns:         false,
				MaxContinuations: false,
				WebSearch:        false,
				NativeAgentFile:  false,
			},
		},
	}
}

// GetHarnessScriptName returns the filename of the AW Harness JavaScript script.
// Implementing HarnessProvider ensures node is set up before execution.
func (e *AWEngine) GetHarnessScriptName() string {
	return AWHarnessScriptName
}

// GetInstallationSteps returns the GitHub Actions steps needed to install the
// @earendil-works/pi-coding-agent SDK required by aw_harness.cjs.
func (e *AWEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	awLog.Printf("Generating installation steps for AW engine: workflow=%s", workflowData.Name)

	version := string(constants.DefaultPiVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	npmSteps := GenerateNpmInstallSteps(
		"@earendil-works/pi-coding-agent",
		version,
		"Install Pi SDK for AW Harness",
		"aw",
		true,  // Include Node.js setup
		false, // Must not run install scripts
		false, // No npm release-age cooldown for SDK installs
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetExecutionSteps returns the GitHub Actions steps for executing aw_harness.cjs.
// The harness receives a generated config.json and the workflow prompt as arguments.
func (e *AWEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	awLog.Printf("Generating execution steps for AW engine: workflow=%s", workflowData.Name)

	harnessPath := fmt.Sprintf(`"%s/%s"`, SetupActionDestinationShell, AWHarnessScriptName)
	awCommand := fmt.Sprintf(
		`%s %s --config %s --prompt %s`,
		nodeRuntimeResolutionCommand,
		harnessPath,
		shellEscapeArg(AWConfigFile),
		shellEscapeArg(constants.AwPromptsFile),
	)

	awCommand = getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + awCommand

	var command string
	if isFirewallEnabled(workflowData) {
		npmPathSetup := GetNpmBinPathSetup()
		awCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, awCommand)
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			awCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, awCommandWithPath)
		}

		var allowedDomains string
		if workflowData.CachedAllowedDomainsComputed {
			allowedDomains = workflowData.CachedAllowedDomainsStr
		} else {
			allowedDomains = GetAllowedDomainsForEngine(
				constants.AWEngine,
				workflowData.NetworkPermissions,
				workflowData.Tools,
				workflowData.Runtimes,
			)
		}
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
		}

		pathSetup := "touch " + AgentStepSummaryPath + "\n" +
			"GH_AW_NODE_BIN=$(command -v node 2>/dev/null || true)\n" +
			"export GH_AW_NODE_BIN"

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "aw",
			EngineCommand:  awCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			AllowedDomains: allowedDomains,
			PathSetup:      pathSetup,
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{
				"ANTHROPIC_API_KEY",
				"OPENAI_API_KEY",
				"COPILOT_GITHUB_TOKEN",
				"GEMINI_API_KEY",
			}),
		})
	} else {
		command = fmt.Sprintf(
			"set -o pipefail\nprintf '%%s' \"$(date +%%s%%3N)\" > %s\ntouch %s\n(umask 177 && touch %s)\n%s 2>&1 | tee -a %s",
			AgentCLIStartMsPath, AgentStepSummaryPath, logFile, awCommand, logFile)
	}

	env := map[string]string{
		"GH_AW_PROMPT":        constants.AwPromptsFile,
		"GITHUB_AW":           "true",
		"GITHUB_WORKSPACE":    "${{ github.workspace }}",
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
		"RUNNER_TEMP":         "${{ runner.temp }}",
		// Pass the config path so the harness can locate it without relying on argv.
		"GH_AW_HARNESS_CONFIG": AWConfigFile,
	}
	// Inject provider-specific credentials into the harness env so buildProviderConfigs()
	// can auto-detect the active provider at runtime.
	providerEnv := map[string]string{
		"ANTHROPIC_API_KEY":    "${{ secrets.ANTHROPIC_API_KEY }}",
		"OPENAI_API_KEY":       "${{ secrets.OPENAI_API_KEY }}",
		"COPILOT_GITHUB_TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}",
		"GEMINI_API_KEY":       "${{ secrets.GEMINI_API_KEY }}",
	}
	maps.Copy(env, providerEnv)
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)

	// Write a minimal config.json for the harness. Provider credentials are passed
	// via env vars, so the config only needs to set harness-level options.
	configJSON := buildAWHarnessConfigJSON(workflowData)
	writeConfigStep := GitHubActionStep{
		"      - name: Write AW Harness config",
		"        run: |",
		fmt.Sprintf("          mkdir -p \"$(dirname %s)\"", shellEscapeArg(AWConfigFile)),
		fmt.Sprintf("          printf '%%s\\n' %s > %s", shellEscapeArg(configJSON), shellEscapeArg(AWConfigFile)),
	}

	// Build the execution step using the standard pattern
	stepLines := []string{
		"      - name: Run AW Harness",
		"        id: agentic_execution",
	}
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, append([]string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"COPILOT_GITHUB_TOKEN",
		"GEMINI_API_KEY",
		"GH_AW_HARNESS_CONFIG",
	}, allowedSecrets...))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)
	execStep := GitHubActionStep(stepLines)
	return []GitHubActionStep{writeConfigStep, execStep}
}

// GetDeclaredOutputFiles returns the JSONL streaming log produced by aw_harness.cjs.
func (e *AWEngine) GetDeclaredOutputFiles() []string {
	return []string{AWStreamingLogFile}
}

// GetAgentManifestFiles returns the instruction files recognised by the AW Harness engine.
func (e *AWEngine) GetAgentManifestFiles() []string {
	return []string{"AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns config directory prefixes for the AW engine.
func (e *AWEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".aw/"}
}

// GetRequiredSecretNames returns the provider secrets that may be needed by the harness.
// The harness auto-detects the active provider from env vars; all common provider
// secrets are listed so the activation job can validate availability.
func (e *AWEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	return collectCommonMCPSecrets(workflowData)
}

// buildAWHarnessConfigJSON generates the JSON config string passed to aw_harness.cjs.
// Provider credentials are injected via env vars; only harness-level options are emitted here.
func buildAWHarnessConfigJSON(workflowData *WorkflowData) string {
	// Minimal config: the harness auto-detects providers from env vars.
	// Future expansion: emit harness.maxTokens, harness.extensions, etc.
	_ = workflowData
	return `{}`
}
