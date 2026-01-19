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
		Short: "Initialize repository for agentic workflows",
		Long: `Initialize the repository for agentic workflows by configuring .gitattributes and creating GitHub Copilot instruction files.

This command:
- Configures .gitattributes to mark .lock.yml files as generated
- Creates .github/aw/logs/.gitignore to ignore downloaded workflow logs
- Creates GitHub Copilot custom instructions at .github/aw/github-agentic-workflows.md
- Creates the dispatcher agent at .github/agents/agentic-workflows.agent.md
- Creates workflow creation prompt at .github/aw/create-agentic-workflow.md (for new workflows)
- Creates workflow update prompt at .github/aw/update-agentic-workflow.md (for updating existing workflows)
- Creates shared workflow creation prompt at .github/aw/create-shared-agentic-workflow.md
- Creates debug workflow prompt at .github/aw/debug-agentic-workflow.md
- Creates upgrade workflow prompt at .github/aw/upgrade-agentic-workflows.md
- Removes old prompt files from .github/prompts/ if they exist
- Configures VSCode settings (.vscode/settings.json)

By default (without --no-mcp):
- Creates .github/workflows/copilot-setup-steps.yml with gh-aw installation steps
- Creates .vscode/mcp.json with gh-aw MCP server configuration

With --no-mcp flag:
- Skips creating GitHub Copilot Agent MCP server configuration files

With --tokens flag:
- Validates which required and optional secrets are configured
- Provides commands to set up missing secrets for the specified engine
- Use with --engine flag to check engine-specific tokens (copilot, claude, codex)

With --codespaces flag:
- Updates existing .devcontainer/devcontainer.json if present, otherwise creates new file at default location
- Configures permissions for current repo: actions:write, contents:write, discussions:read, issues:read, pull-requests:write, workflows:write
- Configures permissions for additional repos (in same org): actions:read, contents:read, discussions:read, issues:read, pull-requests:read, workflows:read
- Adds GitHub Copilot extensions and gh aw CLI installation
- Use without value (--codespaces) for current repo only, or with comma-separated repos (--codespaces repo1,repo2)

With --campaign flag:
- Creates .github/agents/agentic-campaigns.agent.md with the Campaigns dispatcher agent
- Adds (or reuses) .github/aw/agentic-campaign-generator.md source and compiles .github/workflows/agentic-campaign-generator.lock.yml for creating campaigns from issues
- Enables campaign-related prompts and functionality for multi-workflow coordination

With --completions flag:
- Automatically detects your shell (bash, zsh, fish, or PowerShell)
- Installs shell completion configuration for the CLI
- Provides instructions for enabling completions in your shell

After running this command, you can:
- Use GitHub Copilot Chat: type /agent and select agentic-workflows to get started with workflow tasks
- The dispatcher will route your request to the appropriate specialized prompt
- Add workflows from the catalog with: ` + string(constants.CLIExtensionPrefix) + ` add <workflow-name>
- Create new workflows from scratch with: ` + string(constants.CLIExtensionPrefix) + ` new <workflow-name>

To create, update or debug automated agentic actions using github, playwright, and other tools, load the .github/agents/agentic-workflows.agent.md (applies to .github/workflows/*.md)

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` init
  ` + string(constants.CLIExtensionPrefix) + ` init -v
  ` + string(constants.CLIExtensionPrefix) + ` init --no-mcp
  ` + string(constants.CLIExtensionPrefix) + ` init --tokens --engine copilot
  ` + string(constants.CLIExtensionPrefix) + ` init --codespaces
  ` + string(constants.CLIExtensionPrefix) + ` init --codespaces repo1,repo2
  ` + string(constants.CLIExtensionPrefix) + ` init --completions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			mcpFlag, _ := cmd.Flags().GetBool("mcp")
			noMcp, _ := cmd.Flags().GetBool("no-mcp")
			campaign, _ := cmd.Flags().GetBool("campaign")
			tokens, _ := cmd.Flags().GetBool("tokens")
			engine, _ := cmd.Flags().GetString("engine")
			codespaceReposStr, _ := cmd.Flags().GetString("codespaces")
			codespaceEnabled := cmd.Flags().Changed("codespaces")
			completions, _ := cmd.Flags().GetBool("completions")

			// Determine MCP state: default true, unless --no-mcp is specified
			// --mcp flag is kept for backward compatibility (hidden from help)
			mcp := !noMcp
			if cmd.Flags().Changed("mcp") {
				// If --mcp is explicitly set, use it (backward compatibility)
				mcp = mcpFlag
			}

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

			initCommandLog.Printf("Executing init command: verbose=%v, mcp=%v, campaign=%v, tokens=%v, engine=%v, codespaces=%v, codespaceEnabled=%v, completions=%v", verbose, mcp, campaign, tokens, engine, codespaceRepos, codespaceEnabled, completions)
			if err := InitRepository(verbose, mcp, campaign, tokens, engine, codespaceRepos, codespaceEnabled, completions, cmd.Root()); err != nil {
				initCommandLog.Printf("Init command failed: %v", err)
				return err
			}
			initCommandLog.Print("Init command completed successfully")
			return nil
		},
	}

	cmd.Flags().Bool("no-mcp", false, "Skip configuring GitHub Copilot Agent MCP server integration")
	cmd.Flags().Bool("mcp", false, "Configure GitHub Copilot Agent MCP server integration (deprecated, MCP is enabled by default)")
	cmd.Flags().Bool("campaign", false, "Install the Campaign Designer agent for gh-aw campaigns in this repository")
	cmd.Flags().Bool("tokens", false, "Validate required secrets for agentic workflows")
	cmd.Flags().String("engine", "", "AI engine to check tokens for (copilot, claude, codex) - requires --tokens flag")
	cmd.Flags().String("codespaces", "", "Create devcontainer.json for GitHub Codespaces with agentic workflows support. Specify comma-separated repository names in the same organization (e.g., repo1,repo2), or use without value for current repo only")
	// NoOptDefVal allows using --codespaces without a value (returns empty string when no value provided)
	cmd.Flags().Lookup("codespaces").NoOptDefVal = " "
	cmd.Flags().Bool("completions", false, "Install shell completion for the detected shell (bash, zsh, fish, or PowerShell)")

	// Hide the deprecated --mcp flag from help (kept for backward compatibility)
	_ = cmd.Flags().MarkHidden("mcp")

	return cmd
}
