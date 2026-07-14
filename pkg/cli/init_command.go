package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var initCommandLog = logger.New("cli:init_command")

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the repository for agentic workflows",
		Long: `Initialize the repository for agentic workflows by configuring .gitattributes and creating the dispatcher skill file.

This command performs non-interactive repository setup and does not prompt for
engine selection or secret configuration.

This command:
- Configures .gitattributes to mark .lock.yml files as generated
- Creates the dispatcher skill at .github/skills/agentic-workflows/SKILL.md
- Creates the custom agent at .github/agents/agentic-workflows.md
- Removes old prompt files from .github/prompts/ if they exist
- Configures VSCode settings (.vscode/settings.json)
- Generates/updates .github/workflows/agentics-maintenance.yml if any workflows use the expires field for discussions or issues

By default (without --no-mcp):
- Creates .github/workflows/copilot-setup-steps.yml with gh-aw installation steps
- Creates .github/mcp.json with gh-aw MCP server configuration

With --no-mcp flag:
- Skips creating GitHub Copilot Agent MCP server configuration files

With --no-skill flag:
- Skips creating the dispatcher skill

With --no-agent flag:
- Skips creating the custom agent

With --codespaces flag:
- Updates existing .devcontainer/devcontainer.json if present, otherwise creates new file at default location
- Configures permissions for current repo: actions:write, contents:write, discussions:read, issues:read, pull-requests:write, workflows:write
- Configures permissions for additional repos (in same org): actions:read, contents:read, discussions:read, issues:read, pull-requests:read, workflows:read
- Adds GitHub Copilot extensions and gh aw CLI installation
- Use with an empty value (--codespaces "") for current repo only, or with comma-separated repos (--codespaces repo1,repo2)

With --completions flag:
- Automatically detects your shell (bash, zsh, fish, or PowerShell)
- Installs shell completion configuration for the CLI
- Provides instructions for enabling completions in your shell

With --repo flag:
- Attaches to an existing repository or creates a new one (with --create)
- Clones into --dir (default: ./<repo-name>) when the checkout does not exist
- Validates that the local checkout matches the requested repository
- Detects and re-applies initialization markers when missing
- Supports --plan to preview actions, --yes to skip confirmation, and --json for structured output

After running this command, you can:
- Use GitHub Copilot Chat or coding agent tools with the agentic-workflows skill to get started with workflow tasks
- The dispatcher skill will route your request to the appropriate specialized prompt
- Add workflows from the catalog with: ` + string(constants.CLIExtensionPrefix) + ` add <workflow-name>
- Create new workflows from scratch with: ` + string(constants.CLIExtensionPrefix) + ` new <workflow-name>`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` init                                     # Initialize repository with defaults
  ` + string(constants.CLIExtensionPrefix) + ` init -v                                  # Initialize with verbose output
  ` + string(constants.CLIExtensionPrefix) + ` init --engine claude                     # Skip Copilot-specific artifacts
  ` + string(constants.CLIExtensionPrefix) + ` init --no-mcp                            # Skip MCP configuration
  ` + string(constants.CLIExtensionPrefix) + ` init --no-skill                          # Skip dispatcher skill creation
  ` + string(constants.CLIExtensionPrefix) + ` init --no-agent                          # Skip custom agent creation
  ` + string(constants.CLIExtensionPrefix) + ` init --codespaces ""                     # Configure Codespaces for current repo only
  ` + string(constants.CLIExtensionPrefix) + ` init --codespaces repo1,repo2            # Codespaces with additional repos
  ` + string(constants.CLIExtensionPrefix) + ` init --completions                       # Install shell completions
  ` + string(constants.CLIExtensionPrefix) + ` init --create-pull-request               # Initialize and create a pull request
  ` + string(constants.CLIExtensionPrefix) + ` init --repo myorg/myrepo                 # Attach to existing repo and run init
  ` + string(constants.CLIExtensionPrefix) + ` init --repo myorg/myrepo --create        # Create repo and run init
  ` + string(constants.CLIExtensionPrefix) + ` init --repo myorg/myrepo --create --private --dir ./myrepo  # Create private repo, clone into ./myrepo
  ` + string(constants.CLIExtensionPrefix) + ` init --repo myorg/myrepo --plan          # Preview actions without mutating
  ` + string(constants.CLIExtensionPrefix) + ` init --repo myorg/myrepo --yes           # Skip confirmation prompt
  ` + string(constants.CLIExtensionPrefix) + ` init --repo myorg/myrepo --json          # Machine-readable JSON output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			mcpFlag, _ := cmd.Flags().GetBool("mcp")
			noMcp, _ := cmd.Flags().GetBool("no-mcp")
			noSkill, _ := cmd.Flags().GetBool("no-skill")
			noAgent, _ := cmd.Flags().GetBool("no-agent")
			engineOverride, _ := cmd.Flags().GetString("engine")
			codespaceReposStr, _ := cmd.Flags().GetString("codespaces")
			codespaceEnabled := cmd.Flags().Changed("codespaces")
			completions, _ := cmd.Flags().GetBool("completions")
			createPRFlag, _ := cmd.Flags().GetBool("create-pull-request")
			prFlagAlias, _ := cmd.Flags().GetBool("pr")
			createPR := createPRFlag || prFlagAlias // Support both --create-pull-request and --pr

			// Repo setup flags
			repoFlag, _ := cmd.Flags().GetString("repo")
			dirFlag, _ := cmd.Flags().GetString("dir")
			createFlag, _ := cmd.Flags().GetBool("create")
			privateFlag, _ := cmd.Flags().GetBool("private")
			planFlag, _ := cmd.Flags().GetBool("plan")
			yesFlag, _ := cmd.Flags().GetBool("yes")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			requireOwnerTypeStr, _ := cmd.Flags().GetString("require-owner-type")

			if engineOverride != "" {
				registry := workflow.GetGlobalEngineRegistry()
				if !registry.IsValidEngine(engineOverride) {
					supportedEngines := registry.GetSupportedEngines()
					sort.Strings(supportedEngines)
					return fmt.Errorf("invalid engine value '%s'. Must be one of: %s", engineOverride, strings.Join(supportedEngines, ", "))
				}
			}

			// Validate --require-owner-type
			if requireOwnerTypeStr != "" {
				switch OwnerType(requireOwnerTypeStr) {
				case OwnerTypeOrg, OwnerTypeUser, OwnerTypeAny:
					// valid
				default:
					return fmt.Errorf("invalid --require-owner-type value %q: must be one of: org, user, any", requireOwnerTypeStr)
				}
			}

			// --plan, --yes, --json, --dir, --create, --private, --require-owner-type
			// only make sense together with --repo.
			repoSetupFlagsUsed := planFlag || yesFlag || jsonFlag ||
				cmd.Flags().Changed("dir") || createFlag || privateFlag ||
				requireOwnerTypeStr != ""

			if repoSetupFlagsUsed && repoFlag == "" {
				return errors.New("--repo OWNER/REPO is required when using --plan, --yes, --json, --dir, --create, --private, or --require-owner-type")
			}

			// When --repo is provided, run the repo setup flow first.
			if repoFlag != "" {
				initCommandLog.Printf("Running repo setup for %s", repoFlag)
				setupOpts := RepoSetupOptions{
					Repo:             repoFlag,
					Dir:              dirFlag,
					Create:           createFlag,
					Private:          privateFlag,
					RequireOwnerType: OwnerType(requireOwnerTypeStr),
					Plan:             planFlag,
					Yes:              yesFlag,
					JSON:             jsonFlag,
					Verbose:          verbose,
				}
				result, err := PlanAndExecuteRepoSetup(setupOpts)
				if err != nil {
					initCommandLog.Printf("Repo setup failed: %v", err)
					return err
				}
				// When --plan was requested we stop here — no further init steps.
				if planFlag {
					return nil
				}
				// If the result says the repo setup already handled initialization
				// (including running InitRepository internally), we are done.
				if result != nil && result.Status == RepoSetupStatusBlocked {
					return nil
				}
				// Otherwise fall through to the normal InitRepository call below,
				// which handles the in-directory initialization steps.
			}

			// Determine MCP state: default true, unless --no-mcp is specified
			// --mcp flag is kept for backward compatibility (hidden from help)
			mcp := !noMcp
			if cmd.Flags().Changed("mcp") {
				// If --mcp is explicitly set, use it (backward compatibility)
				mcp = mcpFlag
			}

			// Trim the codespace repos string (explicit value required; use --codespaces "" for current repo only)
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

			initCommandLog.Printf("Executing init command: verbose=%v, skill=%v, agent=%v, mcp=%v, codespaces=%v, codespaceEnabled=%v, completions=%v, createPR=%v", verbose, !noSkill, !noAgent, mcp, codespaceRepos, codespaceEnabled, completions, createPR)
			opts := InitOptions{
				Ctx:              cmd.Context(),
				Verbose:          verbose,
				Engine:           engineOverride,
				Skill:            !noSkill,
				Agent:            !noAgent,
				MCP:              mcp,
				CodespaceRepos:   codespaceRepos,
				CodespaceEnabled: codespaceEnabled,
				Completions:      completions,
				CreatePR:         createPR,
				RootCmd:          cmd.Root(),
			}
			if err := InitRepository(opts); err != nil {
				initCommandLog.Printf("Init command failed: %v", err)
				return err
			}
			initCommandLog.Print("Init command completed successfully")
			return nil
		},
	}

	addEngineFlag(cmd)
	cmd.Flags().Bool("no-mcp", false, "Skip configuring gh-aw MCP server integration for GitHub Copilot Agent")
	cmd.Flags().Bool("no-skill", false, "Skip creating the agentic-workflows dispatcher skill")
	cmd.Flags().Bool("no-agent", false, "Skip creating the Agentic Workflows custom agent")
	cmd.Flags().String("codespaces", "", "Create devcontainer.json for GitHub Codespaces with agentic workflow support. Specify comma-separated repository names in the same organization (e.g., repo1,repo2), or use with an empty value for the current repo only")
	cmd.Flags().Bool("completions", false, "Install shell completion for the detected shell (bash, zsh, fish, or PowerShell)")
	cmd.Flags().Bool("create-pull-request", false, "Create a pull request with the initialization changes")
	cmd.Flags().Bool("pr", false, "Alias for --create-pull-request")
	_ = cmd.Flags().MarkHidden("pr") // Hide the short alias from help output
	cmd.Flags().Bool("mcp", false, "Configure GitHub Copilot Agent MCP server integration (deprecated, MCP is enabled by default)")
	// Hide the deprecated --mcp flag from help (kept for backward compatibility)
	_ = cmd.Flags().MarkHidden("mcp")

	// Repo setup flags
	cmd.Flags().String("repo", "", "Repository to attach to or create (format: OWNER/REPO)")
	cmd.Flags().String("dir", "", "Local directory for the checkout (default: ./<repo-name>)")
	cmd.Flags().Bool("create", false, "Create the remote repository when it does not exist")
	cmd.Flags().Bool("private", false, "Create the repository as private (only used with --create)")
	cmd.Flags().Bool("plan", false, "Print planned actions without mutating anything")
	cmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt before mutations")
	cmd.Flags().Bool("json", false, "Emit machine-readable JSON result to stdout")
	cmd.Flags().String("require-owner-type", "", "Require owner to be a specific account type: org, user, or any (default: any)")

	return cmd
}
