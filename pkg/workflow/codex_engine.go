package workflow

import (
	"fmt"
	"strings"
)

// CodexEngine represents the Codex agentic engine (experimental)
type CodexEngine struct {
	BaseEngine
}

func NewCodexEngine() *CodexEngine {
	return &CodexEngine{
		BaseEngine: BaseEngine{
			id:                     "codex",
			displayName:            "Codex",
			description:            "Uses OpenAI Codex CLI (experimental)",
			experimental:           true,
			supportsToolsWhitelist: false,
		},
	}
}

func (e *CodexEngine) GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep {
	// Build the npm install command, optionally with version
	installCmd := "npm install -g @openai/codex"
	if engineConfig != nil && engineConfig.Version != "" {
		installCmd = fmt.Sprintf("npm install -g @openai/codex@%s", engineConfig.Version)
	}

	return []GitHubActionStep{
		{
			"      - name: Setup Node.js",
			"        uses: actions/setup-node@v4",
			"        with:",
			"          node-version: '24'",
		},
		{
			"      - name: Install Codex",
			fmt.Sprintf("        run: %s", installCmd),
		},
	}
}

func (e *CodexEngine) GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig) ExecutionConfig {
	// Use model from engineConfig if available, otherwise default to gpt-4o
	model := "gpt-4o"
	if engineConfig != nil && engineConfig.Model != "" {
		model = engineConfig.Model
	}

	command := fmt.Sprintf(`INSTRUCTION=$(cat /tmp/aw-prompts/prompt.txt)
export CODEX_HOME=/tmp/mcp-config

# Create log directory outside git repo
mkdir -p /tmp/aw-logs

# Run codex with log capture
codex exec \
  -c model=%s \
  --full-auto "$INSTRUCTION" 2>&1 | tee /tmp/aw-logs/%s.log`, model, logFile)

	return ExecutionConfig{
		StepName: "Run Codex",
		Command:  command,
		Environment: map[string]string{
			"OPENAI_API_KEY": "${{ secrets.OPENAI_API_KEY }}",
		},
	}
}

func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	yaml.WriteString("          cat > /tmp/mcp-config/config.toml << EOF\n")

	// Generate [mcp_servers] section
	for _, toolName := range mcpTools {
		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubCodexMCPConfig(yaml, githubTool)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderCodexMCPConfig(yaml, toolName, toolConfig); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
			}
		}
	}

	yaml.WriteString("          EOF\n")
}

// renderGitHubCodexMCPConfig generates GitHub MCP server configuration for codex config.toml
// Always uses Docker MCP as the default
func (e *CodexEngine) renderGitHubCodexMCPConfig(yaml *strings.Builder, githubTool any) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.github]\n")

	// Always use Docker-based GitHub MCP server (services mode has been removed)
	yaml.WriteString("          command = \"docker\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"run\",\n")
	yaml.WriteString("            \"-i\",\n")
	yaml.WriteString("            \"--rm\",\n")
	yaml.WriteString("            \"-e\",\n")
	yaml.WriteString("            \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	yaml.WriteString("            \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          env = { \"GITHUB_PERSONAL_ACCESS_TOKEN\" = \"${{ secrets.GITHUB_TOKEN }}\" }\n")
}

// renderCodexMCPConfig generates custom MCP server configuration for a single tool in codex workflow config.toml
func (e *CodexEngine) renderCodexMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any) error {
	yaml.WriteString("          \n")
	yaml.WriteString(fmt.Sprintf("          [mcp_servers.%s]\n", toolName))

	// Use the shared MCP config renderer with TOML format
	renderer := MCPConfigRenderer{
		IndentLevel: "          ",
		Format:      "toml",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, false, renderer)
	if err != nil {
		return err
	}

	return nil
}
