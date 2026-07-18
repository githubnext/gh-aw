package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

// GeminiEngine represents the Google Gemini CLI agentic engine.
// It embeds googleCLIEngine for all shared behavior; only GetInstallationSteps
// is defined here since Gemini installs via npm (@google/gemini-cli).
type GeminiEngine struct {
	googleCLIEngine
}

var _ CodingAgentEngine = (*GeminiEngine)(nil)

func NewGeminiEngine() *GeminiEngine {
	return &GeminiEngine{
		googleCLIEngine: googleCLIEngine{
			BaseEngine: BaseEngine{
				id:               "gemini",
				displayName:      "Google Gemini CLI",
				description:      "Google Gemini CLI with headless mode and LLM gateway support",
				experimental:     false,
				ghSkillAgentName: "gemini-cli",
				capabilities: EngineCapabilities{
					ToolsAllowlist:   true,
					MaxTurns:         true,
					MaxContinuations: false, // Gemini CLI does not support --max-autopilot-continues-style continuation mode
					WebSearch:        false,
					NativeAgentFile:  false, // Gemini does not support agent file natively; the compiler prepends the agent file content to prompt.txt
				},
				dedicatedLLMGatewayPort: constants.GeminiLLMGatewayPort,
			},
			cfg: googleCLIEngineConfig{
				log:                  logger.New("workflow:gemini_engine"),
				apiKeySecretName:     constants.GeminiAPIKey,
				apiBaseURLEnvVar:     "GEMINI_API_BASE_URL",
				modelEnvVar:          constants.GeminiCLIModelEnvVar,
				trustWorkspaceEnvVar: "GEMINI_CLI_TRUST_WORKSPACE",
				debugEnvValue:        "gemini-cli:*",
				defaultCLIBinary:     "gemini",
				// Auto-approve tool executions; skip workspace trust check so --yolo
				// is not overridden to "default" approval mode by CLI v1.x.
				// Stream-JSON output is required for log parsing.
				cliArgs:               []string{"--yolo", "--skip-trust", "--output-format", "stream-json"},
				configDir:             ".gemini",
				baseConfigEnvVar:      "GH_AW_GEMINI_BASE_CONFIG",
				secretValidationURL:   "https://geminicli.com/docs/get-started/authentication/",
				secretValidationLabel: "Gemini CLI",
				configStepName:        "Write Gemini Config",
				executionStepName:     "Execute Gemini CLI",
				errorMoveStepName:     "Move Gemini error files to artifact directory",
				errorFileSrcGlob:      "/tmp/gemini-client-error-*.json",
				errorFileDstGlob:      constants.TmpGeminiClientErrorGlob,
				agentManifestFiles:    []string{"GEMINI.md", "AGENTS.md"},
				agentManifestPrefixes: []string{".gemini/"},
				logParserScriptID:     "parse_gemini_log",
				logParserEngineName:   "Gemini",
				excludeAPIKeys:        []string{constants.GeminiAPIKey},
				extraAllowedSecrets:   []string{},
				mirrorAPIKeyAs:        "",
			},
		},
	}
}

// GetInstallationSteps returns the GitHub Actions steps needed to install Gemini CLI.
// Gemini installs from npm (@google/gemini-cli); Antigravity uses a different installer.
func (e *GeminiEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	e.cfg.log.Printf("Generating installation steps for Gemini engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		e.cfg.log.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	npmSteps := BuildStandardNpmEngineInstallStepsNoCooldown(
		"@google/gemini-cli",
		string(constants.DefaultGeminiVersion),
		"Install Gemini CLI",
		"gemini",
		workflowData,
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}
