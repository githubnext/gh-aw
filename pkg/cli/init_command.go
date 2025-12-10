package cli

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var initCommandLog = logger.New("cli:init_command")

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize repository for workflows",
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

With --codespaces flag:
- Creates .devcontainer/gh-aw/devcontainer.json with universal image (in subfolder to avoid conflicts)
- Configures permissions for current repo: actions:write, contents:write, discussions:read, issues:read, pull-requests:write, workflows:write
- Configures permissions for additional repos (in same org): actions:read, contents:read, discussions:read, issues:read, pull-requests:read, workflows:read
- Pre-installs gh aw extension CLI
- Pre-installs @github/copilot
- Use without value (--codespaces) for current repo only, or with comma-separated repos (--codespaces repo1,repo2)

After running this command, you can:
- Use GitHub Copilot Chat: type /agent and select create-agentic-workflow to create workflows interactively
- Use GitHub Copilot Chat: type /agent and select debug-agentic-workflow to debug existing workflows
- Add workflows from the catalog with: ` + constants.CLIExtensionPrefix + ` add <workflow-name>
- Create new workflows from scratch with: ` + constants.CLIExtensionPrefix + ` new <workflow-name>

Examples:
  ` + constants.CLIExtensionPrefix + ` init
  ` + constants.CLIExtensionPrefix + ` init -v
  ` + constants.CLIExtensionPrefix + ` init --mcp
  ` + constants.CLIExtensionPrefix + ` init --codespaces
  ` + constants.CLIExtensionPrefix + ` init --codespaces repo1,repo2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			mcp, _ := cmd.Flags().GetBool("mcp")
			codespaceReposStr, _ := cmd.Flags().GetString("codespaces")
			codespaceEnabled := cmd.Flags().Changed("codespaces")

			// Trim the codespace repos string (NoOptDefVal uses a space)
			codespaceReposStr = strings.TrimSpace(codespaceReposStr)

			// Parse codespace repos from comma-separated string
			var codespaceRepos []string
			if codespaceReposStr != "" {
				codespaceRepos = strings.Split(codespaceReposStr, ",")
				// Trim spaces from each repo name
				for i, repo := range codespaceRepos {
					codespaceRepos[i] = strings.TrimSpace(repo)
				}
			}

			initCommandLog.Printf("Executing init command: verbose=%v, mcp=%v, codespaces=%v, codespaceEnabled=%v", verbose, mcp, codespaceRepos, codespaceEnabled)
			if err := InitRepository(verbose, mcp, codespaceRepos, codespaceEnabled); err != nil {
				initCommandLog.Printf("Init command failed: %v", err)
				return err
			}
			initCommandLog.Print("Init command completed successfully")
			return nil
		},
	}

	cmd.Flags().Bool("mcp", false, "Configure GitHub Copilot Agent MCP server integration")
	cmd.Flags().String("codespaces", "", "Create devcontainer.json for GitHub Codespaces with agentic workflows support. Specify comma-separated repository names in the same organization (e.g., repo1,repo2), or use without value for current repo only")
	// NoOptDefVal allows using --codespaces without a value (returns empty string when no value provided)
	cmd.Flags().Lookup("codespaces").NoOptDefVal = " "

	return cmd
}
