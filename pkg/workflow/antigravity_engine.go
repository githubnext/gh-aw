package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

// AntigravityEngine represents the Google Antigravity CLI agentic engine.
// It embeds googleCLIEngine for all shared behavior; only GetInstallationSteps
// is defined here since Antigravity installs from a GCS binary (not npm).
type AntigravityEngine struct {
	googleCLIEngine
}

var _ CodingAgentEngine = (*AntigravityEngine)(nil)

func NewAntigravityEngine() *AntigravityEngine {
	return &AntigravityEngine{
		googleCLIEngine: googleCLIEngine{
			BaseEngine: BaseEngine{
				id:               "antigravity",
				displayName:      "Antigravity CLI",
				description:      "Antigravity CLI with headless mode and LLM gateway support",
				experimental:     true,
				ghSkillAgentName: "antigravity",
				capabilities: EngineCapabilities{
					ToolsAllowlist:   true,
					MaxTurns:         true,
					MaxContinuations: false, // Antigravity CLI does not support --max-autopilot-continues-style continuation mode
					WebSearch:        false,
					NativeAgentFile:  false, // Antigravity does not support agent file natively; the compiler prepends the agent file content to prompt.txt
				},
				dedicatedLLMGatewayPort: constants.AntigravityLLMGatewayPort,
			},
			cfg: googleCLIEngineConfig{
				log:                  logger.New("workflow:antigravity_engine"),
				apiKeySecretName:     constants.AntigravityAPIKey,
				apiBaseURLEnvVar:     "ANTIGRAVITY_API_BASE_URL",
				modelEnvVar:          constants.AntigravityCLIModelEnvVar,
				trustWorkspaceEnvVar: "ANTIGRAVITY_CLI_TRUST_WORKSPACE",
				debugEnvValue:        "antigravity-cli:*",
				defaultCLIBinary:     "agy",
				// Grant broad tool permission inside the workflow sandbox without blocking on
				// permission prompts. agy does not support the Gemini-style --yolo/--skip-trust
				// flags; --dangerously-skip-permissions is the equivalent for Antigravity.
				cliArgs:               []string{"--dangerously-skip-permissions"},
				configDir:             ".antigravity",
				baseConfigEnvVar:      "GH_AW_ANTIGRAVITY_BASE_CONFIG",
				secretValidationURL:   "https://antigravity.google/docs/cli-overview",
				secretValidationLabel: "Antigravity CLI",
				configStepName:        "Write Antigravity Config",
				executionStepName:     "Execute Antigravity CLI",
				errorMoveStepName:     "Move Antigravity error files to artifact directory",
				errorFileSrcGlob:      "/tmp/antigravity-client-error-*.json",
				errorFileDstGlob:      constants.TmpAntigravityClientErrorGlob,
				agentManifestFiles:    []string{"ANTIGRAVITY.md", "AGENTS.md"},
				agentManifestPrefixes: []string{".antigravity/"},
				logParserScriptID:     "parse_antigravity_log",
				logParserEngineName:   "Antigravity",
				// Exclude both the Antigravity and mirrored Gemini API keys from the sandbox env.
				excludeAPIKeys: []string{constants.AntigravityAPIKey, constants.GeminiAPIKey},
				// GEMINI_API_KEY must be allowed through FilterEnvForSecrets because it is
				// mirrored from ANTIGRAVITY_API_KEY and is required by the Gemini proxy sidecar.
				extraAllowedSecrets: []string{constants.GeminiAPIKey},
				// Mirror ANTIGRAVITY_API_KEY → GEMINI_API_KEY so the Gemini proxy sidecar can
				// authenticate without requiring users to duplicate secrets.
				mirrorAPIKeyAs: constants.GeminiAPIKey,
			},
		},
	}
}

// GetInstallationSteps returns the GitHub Actions steps needed to install Antigravity CLI.
// Antigravity installs from a GCS binary via install_antigravity_cli.sh; Gemini uses npm.
func (e *AntigravityEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	e.cfg.log.Printf("Generating installation steps for Antigravity engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		e.cfg.log.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	version := string(constants.DefaultAntigravityVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}
	installSteps := GenerateAntigravityInstallerSteps(version, "Install Antigravity CLI")
	return BuildNpmEngineInstallStepsWithAWF(installSteps, workflowData)
}
