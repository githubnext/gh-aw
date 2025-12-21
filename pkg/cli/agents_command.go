package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var agentsLog = logger.New("cli:agents_command")

// NewAgentsCommand creates the agents command group for managing agentic workflows
func NewAgentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage agentic workflows in your repository",
		Long: `Manage the lifecycle of agentic workflows in your repository.

This command provides an easy-to-use interface for discovering, installing,
updating, and removing agentic workflows from repositories like githubnext/agentics.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents list          # List installed agents
  ` + constants.CLIExtensionPrefix + ` agents browse        # Browse and install agents interactively
  ` + constants.CLIExtensionPrefix + ` agents install       # Install agents interactively
  ` + constants.CLIExtensionPrefix + ` agents uninstall     # Remove agents interactively
  ` + constants.CLIExtensionPrefix + ` agents update        # Update installed agents
  ` + constants.CLIExtensionPrefix + ` agents info ci-doctor  # Show details about an agent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newAgentsListSubcommand())
	cmd.AddCommand(newAgentsBrowseSubcommand())
	cmd.AddCommand(newAgentsInstallSubcommand())
	cmd.AddCommand(newAgentsUninstallSubcommand())
	cmd.AddCommand(newAgentsUpdateSubcommand())
	cmd.AddCommand(newAgentsInfoSubcommand())

	return cmd
}

// newAgentsListSubcommand creates the agents list subcommand
func newAgentsListSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed agentic workflows",
		Long: `List all agentic workflows currently installed in your repository.

Shows workflow name, status (enabled/disabled), trigger type, and source repository.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents list
  ` + constants.CLIExtensionPrefix + ` agents list --json
  ` + constants.CLIExtensionPrefix + ` agents list --repo githubnext/agentics`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoFilter, _ := cmd.Flags().GetString("repo")

			return listInstalledAgents(verbose, jsonOutput, repoFilter)
		},
	}

	addJSONFlag(cmd)
	cmd.Flags().String("repo", "", "Filter by source repository (e.g., githubnext/agentics)")

	return cmd
}

// newAgentsBrowseSubcommand creates the agents browse subcommand
func newAgentsBrowseSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse [repository]",
		Short: "Browse and install agents interactively",
		Long: `Launch an interactive browser to discover and install agentic workflows.

Browse workflows from the specified repository (defaults to githubnext/agentics).
Select multiple workflows to install at once with a visual interface.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents browse
  ` + constants.CLIExtensionPrefix + ` agents browse githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` agents browse githubnext/agentics@v1.0.0`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if running in CI environment
			if IsRunningInCI() {
				return fmt.Errorf("interactive mode cannot be used in CI environments")
			}

			repository := "githubnext/agentics"
			if len(args) > 0 {
				repository = args[0]
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			force, _ := cmd.Flags().GetBool("force")

			return browseAndInstallAgents(repository, verbose, force)
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite existing workflow files")

	return cmd
}

// newAgentsInstallSubcommand creates the agents install subcommand
func newAgentsInstallSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [workflow]...",
		Short: "Install agentic workflows",
		Long: `Install one or more agentic workflows from a repository.

When called without arguments, launches an interactive selection interface.
When called with workflow names, installs them directly.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents install                    # Interactive mode
  ` + constants.CLIExtensionPrefix + ` agents install ci-doctor          # Install specific agent
  ` + constants.CLIExtensionPrefix + ` agents install ci-doctor daily-plan  # Install multiple
  ` + constants.CLIExtensionPrefix + ` agents install ci-doctor --repo githubnext/agentics`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			force, _ := cmd.Flags().GetBool("force")
			repository, _ := cmd.Flags().GetString("repo")

			if len(args) == 0 {
				// Check if running in CI environment
				if IsRunningInCI() {
					return fmt.Errorf("interactive mode cannot be used in CI environments")
				}

				// Interactive mode
				if repository == "" {
					repository = "githubnext/agentics"
				}
				return browseAndInstallAgents(repository, verbose, force)
			}

			// Direct installation mode
			if repository == "" {
				repository = "githubnext/agentics"
			}

			return installAgentsDirect(repository, args, verbose, force)
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite existing workflow files")
	cmd.Flags().String("repo", "", "Source repository (default: githubnext/agentics)")

	return cmd
}

// newAgentsUninstallSubcommand creates the agents uninstall subcommand
func newAgentsUninstallSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall [workflow]...",
		Short: "Uninstall agentic workflows",
		Long: `Uninstall one or more agentic workflows from your repository.

When called without arguments, launches an interactive selection interface.
When called with workflow names, removes them directly.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents uninstall                  # Interactive mode
  ` + constants.CLIExtensionPrefix + ` agents uninstall ci-doctor        # Remove specific agent
  ` + constants.CLIExtensionPrefix + ` agents uninstall ci-doctor daily-plan  # Remove multiple`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			keepOrphans, _ := cmd.Flags().GetBool("keep-orphans")

			if len(args) == 0 {
				// Check if running in CI environment
				if IsRunningInCI() {
					return fmt.Errorf("interactive mode cannot be used in CI environments")
				}

				// Interactive mode
				return uninstallAgentsInteractive(verbose, keepOrphans)
			}

			// Direct removal mode
			return uninstallAgentsDirect(args, verbose, keepOrphans)
		},
	}

	cmd.Flags().Bool("keep-orphans", false, "Keep orphaned include files")

	return cmd
}

// newAgentsUpdateSubcommand creates the agents update subcommand
func newAgentsUpdateSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [workflow]...",
		Short: "Update installed agentic workflows",
		Long: `Update one or more installed agentic workflows to their latest versions.

When called without arguments, shows available updates and prompts for confirmation.
When called with workflow names, updates only those workflows.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents update                     # Interactive mode
  ` + constants.CLIExtensionPrefix + ` agents update ci-doctor           # Update specific agent
  ` + constants.CLIExtensionPrefix + ` agents update ci-doctor daily-plan  # Update multiple
  ` + constants.CLIExtensionPrefix + ` agents update --all                # Update all agents`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			updateAll, _ := cmd.Flags().GetBool("all")
			force, _ := cmd.Flags().GetBool("force")

			if updateAll || len(args) == 0 {
				// Check if running in CI environment for interactive mode
				if !updateAll && IsRunningInCI() {
					return fmt.Errorf("interactive mode cannot be used in CI environments. Use --all flag")
				}

				// Interactive or update-all mode
				return updateAgentsInteractive(verbose, updateAll, force)
			}

			// Direct update mode
			return updateAgentsDirect(args, verbose, force)
		},
	}

	cmd.Flags().Bool("all", false, "Update all installed agents without prompting")
	cmd.Flags().Bool("force", false, "Force update even if no changes detected")

	return cmd
}

// newAgentsInfoSubcommand creates the agents info subcommand
func newAgentsInfoSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <workflow>",
		Short: "Show detailed information about an agent",
		Long: `Display detailed information about a specific agentic workflow.

Shows metadata including description, category, triggers, permissions,
safe outputs, source repository, and installation status.

Examples:
  ` + constants.CLIExtensionPrefix + ` agents info ci-doctor
  ` + constants.CLIExtensionPrefix + ` agents info daily-plan --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName := args[0]
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			return showAgentInfo(workflowName, verbose, jsonOutput)
		},
	}

	addJSONFlag(cmd)

	return cmd
}

// listInstalledAgents lists all installed agentic workflows
func listInstalledAgents(verbose bool, jsonOutput bool, repoFilter string) error {
	agentsLog.Printf("Listing installed agents: jsonOutput=%v, repoFilter=%s", jsonOutput, repoFilter)

	// Get all workflows in .github/workflows
	workflows, err := scanInstalledWorkflows(verbose)
	if err != nil {
		return fmt.Errorf("failed to scan installed workflows: %w", err)
	}

	// Filter by source repository if specified
	if repoFilter != "" {
		workflows = filterWorkflowsBySource(workflows, repoFilter)
	}

	if len(workflows) == 0 {
		if repoFilter != "" {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No agents found from repository: %s", repoFilter)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No agents installed"))
		}
		return nil
	}

	// Display workflows
	if jsonOutput {
		return displayWorkflowsJSON(workflows)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installed Agents (%d):", len(workflows))))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, console.RenderStruct(workflows))

	return nil
}
