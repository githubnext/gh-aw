package main

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <workflow>...",
	Short: "Add one or more workflows from the components to .github/workflows",
	Long: `Add one or more workflows from the components to .github/workflows.

Examples:
  ` + constants.CLIExtensionPrefix + ` add weekly-research
  ` + constants.CLIExtensionPrefix + ` add ci-doctor daily-perf-improver
  ` + constants.CLIExtensionPrefix + ` add weekly-research -n my-custom-name
  ` + constants.CLIExtensionPrefix + ` add weekly-research -r githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` add weekly-research --pr
  ` + constants.CLIExtensionPrefix + ` add weekly-research daily-plan --force

The -r flag allows you to install and use workflows from a specific repository.
The -n flag allows you to specify a custom name for the workflow file (only applies to the first workflow when adding multiple).
The --pr flag automatically creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.
It's a shortcut for:
  ` + constants.CLIExtensionPrefix + ` install githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` add weekly-research`,
	Args: func(cmd *cobra.Command, args []string) error {
		// If no arguments provided and not in CI, automatically use interactive mode
		if len(args) == 0 && !isRunningInCI() {
			// Auto-enable interactive mode
			var workflowName = "my-workflow" // Default name
			if err := cli.CreateWorkflowInteractively(workflowName, verbose, false); err != nil {
				return fmt.Errorf("failed to create workflow interactively: %w", err)
			}
			// Exit successfully after interactive creation
			os.Exit(0)
		}

		// Normal mode requires at least one workflow
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 arg(s), received %d", len(args))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		workflows := args
		numberFlag, _ := cmd.Flags().GetInt("number")
		engineOverride, _ := cmd.Flags().GetString("engine")
		repoFlag, _ := cmd.Flags().GetString("repo")
		nameFlag, _ := cmd.Flags().GetString("name")
		prFlag, _ := cmd.Flags().GetBool("pr")
		forceFlag, _ := cmd.Flags().GetBool("force")

		if err := validateEngine(engineOverride); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}

		// Handle normal mode
		if prFlag {
			if err := cli.AddWorkflows(workflows, numberFlag, verbose, engineOverride, repoFlag, nameFlag, forceFlag, true); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		} else {
			if err := cli.AddWorkflows(workflows, numberFlag, verbose, engineOverride, repoFlag, nameFlag, forceFlag, false); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		}
	},
}

func init() {
	// Add the command to root
	rootCmd.AddCommand(addCmd)

	// Add number flag to add command
	addCmd.Flags().IntP("number", "c", 1, "Create multiple numbered copies")

	// Add name flag to add command
	addCmd.Flags().StringP("name", "n", "", "Specify name for the added workflow (without .md extension)")

	// Add AI flag to add command
	addCmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex)")

	// Add repository flag to add command
	addCmd.Flags().StringP("repo", "r", "", "Install and use workflows from specified repository (org/repo)")

	// Add PR flag to add command
	addCmd.Flags().Bool("pr", false, "Create a pull request with the workflow changes")

	// Add force flag to add command
	addCmd.Flags().Bool("force", false, "Overwrite existing workflow files")
}
