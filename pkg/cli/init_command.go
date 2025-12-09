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
- Creates .github/aw/logs/.gitignore to ignore downloaded workflow logs
- Creates GitHub Copilot custom instructions at .github/aw/github-agentic-workflows.md
- Creates the agent for workflow creation at .github/agents/create-agentic-workflow.agent.md
- Creates the debug agentic workflow agent at .github/agents/debug-agentic-workflow.agent.md
- Removes old prompt files from .github/prompts/ if they exist

With --mcp flag:
- Creates .github/workflows/copilot-setup-steps.yml with gh-aw installation steps
- Creates .vscode/mcp.json with gh-aw MCP server configuration

With --codespace flag:
- Creates .devcontainer/devcontainer.json with universal image
- Configures workflows: write permission for triggering actions
- Adds read access to githubnext/gh-aw releases
- Adds read access to additional repositories specified (e.g., --codespace owner/repo1,owner/repo2)
- Pre-installs gh aw extension CLI
- Pre-installs @github/copilot

After running this command, you can:
- Use GitHub Copilot Chat: type /agent and select create-agentic-workflow to create workflows interactively
- Use GitHub Copilot Chat: type /agent and select debug-agentic-workflow to debug existing workflows
- Add workflows from the catalog with: ` + constants.CLIExtensionPrefix + ` add <workflow-name>
- Create new workflows from scratch with: ` + constants.CLIExtensionPrefix + ` new <workflow-name>

Examples:
  ` + constants.CLIExtensionPrefix + ` init
  ` + constants.CLIExtensionPrefix + ` init -v
  ` + constants.CLIExtensionPrefix + ` init --mcp
  ` + constants.CLIExtensionPrefix + ` init --codespace owner/repo1,owner/repo2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			mcp, _ := cmd.Flags().GetBool("mcp")
			codespaceRepos, _ := cmd.Flags().GetStringSlice("codespace")
			initCommandLog.Printf("Executing init command: verbose=%v, mcp=%v, codespace=%v", verbose, mcp, codespaceRepos)
			if err := InitRepository(verbose, mcp, codespaceRepos); err != nil {
				initCommandLog.Printf("Init command failed: %v", err)
				return err
			}
			initCommandLog.Print("Init command completed successfully")
			return nil
		},
	}

	cmd.Flags().Bool("mcp", false, "Configure GitHub Copilot Agent MCP server integration")
	cmd.Flags().StringSlice("codespace", []string{}, "Create devcontainer.json for GitHub Codespaces with agentic workflows support. Optionally specify additional repositories for read access (e.g., owner/repo1,owner/repo2)")

	return cmd
}
