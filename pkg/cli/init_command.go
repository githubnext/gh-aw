package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
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
- Creates GitHub Copilot custom instructions at .github/instructions/github-agentic-workflows.instructions.md
- Creates the custom agent for workflow creation at .github/agents/create-agentic-workflow.md
- Creates the setup agentic workflows agent at .github/agents/setup-agentic-workflows.md
- Removes the old /create-agentic-workflow prompt if it exists

With --mcp flag:
- Creates .github/workflows/copilot-setup-steps.yml with gh-aw installation steps
- Creates .vscode/mcp.json with gh-aw MCP server configuration

After running this command, you can:
- Use GitHub Copilot Chat with @.github/agents/create-agentic-workflow.md to create workflows interactively
- Use GitHub Copilot Chat with @.github/agents/setup-agentic-workflows.md for setup guidance
- Add workflows from the catalog with: ` + constants.CLIExtensionPrefix + ` add <workflow-name>
- Create new workflows from scratch with: ` + constants.CLIExtensionPrefix + ` new <workflow-name>

Examples:
  ` + constants.CLIExtensionPrefix + ` init
  ` + constants.CLIExtensionPrefix + ` init -v
  ` + constants.CLIExtensionPrefix + ` init --mcp`,
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			mcp, _ := cmd.Flags().GetBool("mcp")
			initCommandLog.Printf("Executing init command: verbose=%v, mcp=%v", verbose, mcp)
			if err := InitRepository(verbose, mcp); err != nil {
				initCommandLog.Printf("Init command failed: %v", err)
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
			initCommandLog.Print("Init command completed successfully")
		},
	}

	cmd.Flags().Bool("mcp", false, "Configure GitHub Copilot Agent MCP server integration")

	return cmd
}
