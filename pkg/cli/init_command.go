package cli

import (
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var initCommandLog = logger.New("cli:init_command")

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize repository for agentic workflows",
		Long: `Initialize the repository for agentic workflows by configuring .gitattributes and creating GitHub Copilot instruction files.

This command:
- Configures .gitattributes to mark .lock.yml files as generated
- Creates GitHub Copilot custom instructions at .github/aw/github-agentic-workflows.md
- Creates the agent for workflow creation at .github/agents/create-agentic-workflow.agent.md
- Creates the setup agentic workflows agent at .github/agents/setup-agentic-workflows.agent.md
- Creates the debug agentic workflow agent at .github/agents/debug-agentic-workflow.agent.md
- Removes old prompt files from .github/prompts/ if they exist

With --mcp flag:
- Creates .github/workflows/copilot-setup-steps.yml with gh-aw installation steps
- Creates .vscode/mcp.json with gh-aw MCP server configuration

With --codespace flag:
- Detects or creates .devcontainer/devcontainer.json
- Adds GitHub Codespace token permissions for actions, contents, workflows, issues, pull requests, and discussions
- Adds read permissions for githubnext/gh-aw repository to enable future extension downloads

After running this command, you can:
- Use GitHub Copilot Chat: type /create-agentic-workflow to create workflows interactively
- Use GitHub Copilot Chat: type /setup-agentic-workflows for setup guidance
- Use GitHub Copilot Chat: type /debug-agentic-workflow to debug existing workflows
- Add workflows from the catalog with: ` + constants.CLIExtensionPrefix + ` add <workflow-name>
- Create new workflows from scratch with: ` + constants.CLIExtensionPrefix + ` new <workflow-name>

Examples:
  ` + constants.CLIExtensionPrefix + ` init
  ` + constants.CLIExtensionPrefix + ` init -v
  ` + constants.CLIExtensionPrefix + ` init --mcp
  ` + constants.CLIExtensionPrefix + ` init --codespace`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			mcp, _ := cmd.Flags().GetBool("mcp")
			codespace, _ := cmd.Flags().GetBool("codespace")
			initCommandLog.Printf("Executing init command: verbose=%v, mcp=%v, codespace=%v", verbose, mcp, codespace)
			if err := InitRepository(verbose, mcp, codespace); err != nil {
				initCommandLog.Printf("Init command failed: %v", err)
				return err
			}
			initCommandLog.Print("Init command completed successfully")
			return nil
		},
	}

	cmd.Flags().Bool("mcp", false, "Configure GitHub Copilot Agent MCP server integration")
	cmd.Flags().Bool("codespace", false, "Optimize setup for GitHub Codespaces with token permissions")

	return cmd
}
