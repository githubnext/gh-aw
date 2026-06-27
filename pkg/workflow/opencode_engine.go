package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var openCodeLog = logger.New("workflow:opencode_engine")

// OpenCodeEngine represents the OpenCode CLI agentic engine.
// OpenCode is a provider-agnostic, open-source AI coding agent that supports
// multiple models via BYOK (Bring Your Own Key).
type OpenCodeEngine struct {
	UniversalLLMConsumerEngine
}

func NewOpenCodeEngine() *OpenCodeEngine {
	return &OpenCodeEngine{
		UniversalLLMConsumerEngine: UniversalLLMConsumerEngine{
			BaseEngine: BaseEngine{
				id:           "opencode",
				displayName:  "OpenCode",
				description:  "OpenCode CLI with headless mode and multi-provider LLM support",
				experimental: true,
				capabilities: EngineCapabilities{
					ToolsAllowlist: false,
					MaxTurns:       true,
					WebSearch:      false,
				},
			},
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the OpenCode CLI uses
// for model selection. Setting OPENCODE_MODEL is equivalent to passing --model to the CLI.
func (e *OpenCodeEngine) GetModelEnvVarName() string {
	return constants.OpenCodeCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the OpenCode engine.
// By default, OpenCode routes through the Copilot API using COPILOT_GITHUB_TOKEN
// (or ${{ github.token }} when permissions.copilot-requests is set to write).
// Additional provider API keys can be added via engine.env overrides.
func (e *OpenCodeEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	openCodeLog.Print("Collecting required secrets for OpenCode engine")
	return e.GetUniversalRequiredSecretNames(workflowData)
}

// GetSupportedEnvVarKeys returns the engine.env variable names that the OpenCode engine
// supports as defined in the AWF specification. OpenCode is a multi-provider engine so all
// provider API keys are valid engine.env overrides.
func (e *OpenCodeEngine) GetSupportedEnvVarKeys() []string {
	return []string{
		constants.CopilotGitHubToken,
		constants.AnthropicAPIKey,
		constants.CodexAPIKey,
		constants.OpenAIAPIKey,
	}
}

// GetInstallationSteps returns the GitHub Actions steps needed to install OpenCode CLI
func (e *OpenCodeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	openCodeLog.Printf("Generating installation steps for OpenCode engine: workflow=%s", workflowData.Name)

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		openCodeLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	npmSteps := BuildStandardNpmEngineInstallSteps(
		"opencode-ai",
		string(constants.DefaultOpenCodeVersion),
		"Install OpenCode CLI",
		"opencode",
		workflowData,
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetSecretValidationStep returns the secret validation step for the OpenCode engine.
// Returns an empty step if permissions.copilot-requests is write (uses GitHub Actions token).
func (e *OpenCodeEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return e.GetUniversalSecretValidationStep(
		workflowData,
		"OpenCode CLI",
		"https://github.github.com/gh-aw/reference/engines/#opencode",
	)
}

func (e *OpenCodeEngine) GetAgentManifestFiles() []string {
	return []string{"opencode.jsonc", "AGENTS.md"}
}

func (e *OpenCodeEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".opencode/"}
}

// GetDeclaredOutputFiles returns the output files that OpenCode may produce.
func (e *OpenCodeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing OpenCode
func (e *OpenCodeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	openCodeLog.Printf("Generating execution steps for OpenCode engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	return e.BuildCLIEngineExecutionSteps(workflowData, logFile, UniversalCLIEngineExecutionConfig{
		EngineConstant:     constants.OpenCodeEngine,
		DefaultCommandName: "opencode",
		ExtraCLIArgs:       []string{"--print-logs", "--log-level", "DEBUG"},
		MCPConfigFile:      "opencode.jsonc",
		StepName:           "Execute OpenCode CLI",
		ConfigStep:         e.generateOpenCodeConfigStep(workflowData),
		ModelEnvVarName:    constants.OpenCodeCLIModelEnvVar,
		WriteTimestamp:     false,
	})
}

// generateOpenCodeConfigStep writes opencode.jsonc with all permissions set to allow
// to prevent CI hanging on permission prompts.
func (e *OpenCodeEngine) generateOpenCodeConfigStep(_ *WorkflowData) GitHubActionStep {
	configJSON := `{"agent":{"build":{"permissions":{"bash":"allow","edit":"allow","read":"allow","glob":"allow","grep":"allow","write":"allow","webfetch":"allow","websearch":"allow"}}}}`

	command := fmt.Sprintf(`umask 077
mkdir -p "$GITHUB_WORKSPACE"
CONFIG="$GITHUB_WORKSPACE/opencode.jsonc"
BASE_CONFIG='%s'
if [ -f "$CONFIG" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$CONFIG")" '$existing * $base')
  echo "$MERGED" > "$CONFIG"
else
  echo "$BASE_CONFIG" > "$CONFIG"
fi
chmod 600 "$CONFIG"`, configJSON)

	stepLines := []string{"      - name: Write OpenCode Config"}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, nil)
	return GitHubActionStep(stepLines)
}
