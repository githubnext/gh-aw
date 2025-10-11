package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize repository for agentic workflows",
		Long: `Initialize the repository for agentic workflows by configuring .gitattributes and creating GitHub Copilot instruction files.

This command:
- Configures .gitattributes to mark .lock.yml files as generated
- Creates GitHub Copilot custom instructions at .github/instructions/github-agentic-workflows.instructions.md
- Creates the /create-agentic-workflow prompt at .github/prompts/create-agentic-workflow.prompt.md

After running this command, you can:
- Use GitHub Copilot Chat with /create-agentic-workflow to create workflows interactively
- Add workflows from the catalog with: ` + constants.CLIExtensionPrefix + ` add <workflow-name>
- Create new workflows from scratch with: ` + constants.CLIExtensionPrefix + ` new <workflow-name>

Examples:
  ` + constants.CLIExtensionPrefix + ` init
  ` + constants.CLIExtensionPrefix + ` init -v`,
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if err := InitRepository(verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}
}
