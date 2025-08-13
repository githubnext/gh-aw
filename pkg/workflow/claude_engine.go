package workflow

import (
	"fmt"
	"strings"
)

// ClaudeEngine represents the Claude Code agentic engine
type ClaudeEngine struct {
	BaseEngine
}

func NewClaudeEngine() *ClaudeEngine {
	return &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:                     "claude",
			displayName:            "Claude Code",
			description:            "Uses Claude Code with full MCP tool support and allow-listing",
			experimental:           false,
			supportsToolsWhitelist: true,
			supportsHTTPTransport:  true, // Claude supports both stdio and HTTP transport
		},
	}
}

func (e *ClaudeEngine) GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep {
	// Claude Code doesn't require installation as it uses claude-base-action
	return []GitHubActionStep{}
}

func (e *ClaudeEngine) GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig) ExecutionConfig {
	config := ExecutionConfig{
		StepName: "Execute Claude Code Action",
		Action:   "anthropics/claude-code-base-action@beta",
		Inputs: map[string]string{
			"prompt_file":       "/tmp/aw-prompts/prompt.txt",
			"anthropic_api_key": "${{ secrets.ANTHROPIC_API_KEY }}",
			"mcp_config":        "/tmp/mcp-config/mcp-servers.json",
			"claude_env":        "|\n            GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}",
			"allowed_tools":     "", // Will be filled in during generation
			"timeout_minutes":   "", // Will be filled in during generation
		},
		Environment: map[string]string{
			"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	}

	// Add model configuration if specified
	if engineConfig != nil && engineConfig.Model != "" {
		config.Inputs["model"] = engineConfig.Model
	}

	return config
}

func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	yaml.WriteString("          cat > /tmp/mcp-config/mcp-servers.json << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Generate configuration for each MCP tool
	for i, toolName := range mcpTools {
		isLast := i == len(mcpTools)-1

		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubClaudeMCPConfig(yaml, githubTool, isLast)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderClaudeMCPConfig(yaml, toolName, toolConfig, isLast); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
			}
		}
	}

	yaml.WriteString("            }\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF\n")
}

// renderGitHubClaudeMCPConfig generates the GitHub MCP server configuration
// Always uses Docker MCP as the default
func (e *ClaudeEngine) renderGitHubClaudeMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Always use Docker-based GitHub MCP server (services mode has been removed)
	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"${{ secrets.GITHUB_TOKEN }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderClaudeMCPConfig generates custom MCP server configuration for a single tool in Claude workflow mcp-servers.json
func (e *ClaudeEngine) renderClaudeMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	yaml.WriteString(fmt.Sprintf("              \"%s\": {\n", toolName))

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel: "                ",
		Format:      "json",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, isLast, renderer)
	if err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}
