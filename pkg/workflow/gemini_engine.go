package workflow

import (
	"strings"
)

// GeminiEngine represents the Google Gemini CLI agentic engine
type GeminiEngine struct {
	BaseEngine
}

func NewGeminiEngine() *GeminiEngine {
	return &GeminiEngine{
		BaseEngine: BaseEngine{
			id:                     "gemini",
			displayName:            "Gemini CLI",
			description:            "Uses Google Gemini CLI with GitHub integration and tool support",
			experimental:           false,
			supportsToolsWhitelist: true,
			supportsHTTPTransport:  false, // Gemini CLI does not support custom MCP HTTP servers
		},
	}
}

func (e *GeminiEngine) GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep {
	// Gemini CLI doesn't require installation as it uses the Google GitHub Action
	return []GitHubActionStep{}
}

func (e *GeminiEngine) GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig) ExecutionConfig {
	config := ExecutionConfig{
		StepName: "Execute Gemini CLI Action",
		Action:   "google-github-actions/run-gemini-cli@v1",
		Inputs: map[string]string{
			"prompt":         "$(cat /tmp/aw-prompts/prompt.txt)", // Read from the prompt file
			"gemini_api_key": "${{ secrets.GEMINI_API_KEY }}",
		},
		Environment: map[string]string{
			"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	}

	// Add model configuration via settings if specified
	if engineConfig != nil && engineConfig.Model != "" {
		// Gemini CLI uses settings JSON for model configuration
		settingsJSON := `{"model": "` + engineConfig.Model + `"}`
		config.Inputs["settings"] = settingsJSON
	}

	return config
}

func (e *GeminiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	// Gemini CLI has built-in GitHub integration, so we don't need external MCP configuration
	// The GitHub tools are handled natively by the Gemini CLI when it has access to GITHUB_TOKEN

	yaml.WriteString("          # Gemini CLI handles GitHub integration natively when GITHUB_TOKEN is available\n")
	yaml.WriteString("          # No additional MCP configuration required for GitHub tools\n")

	// Check if there are custom MCP tools beyond GitHub
	hasCustomMCP := false
	for _, toolName := range mcpTools {
		if toolName != "github" {
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					hasCustomMCP = true
					break
				}
			}
		}
	}

	if hasCustomMCP {
		yaml.WriteString("          # Note: Custom MCP tools are not currently supported by Gemini CLI engine\n")
		yaml.WriteString("          # Consider using claude or opencode engines for custom MCP integrations\n")
	}
}
