package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// Build-time variables set by GoReleaser
var (
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     constants.CLIExtensionPrefix,
	Short:   "GitHub Agentic Workflows CLI from GitHub Next",
	Version: version,
	Long: `GitHub Agentic Workflows from GitHub Next

Common Tasks:
  gh aw init                  # Set up a new repository
  gh aw new my-workflow       # Create your first workflow
  gh aw compile               # Compile all workflows
  gh aw run my-workflow       # Execute a workflow
  gh aw logs my-workflow      # View execution logs
  gh aw audit <run-id>        # Debug a failed run

For detailed help on any command, use:
  gh aw [command] --help`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	// Add command groups to root command
	rootCmd.AddGroup(&cobra.Group{
		ID:    "setup",
		Title: "Setup Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "development",
		Title: "Development Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "execution",
		Title: "Execution Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "analysis",
		Title: "Analysis Commands:",
	})

	// Add global verbose flag to root command
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output showing detailed information")

	// Set output to stderr for consistency with CLI logging guidelines
	rootCmd.SetOut(os.Stderr)

	// Set version template to match the version subcommand format
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n%s\n",
		console.FormatInfoMessage(fmt.Sprintf("%s version {{.Version}}", constants.CLIExtensionPrefix)),
		console.FormatInfoMessage("GitHub Agentic Workflows CLI from GitHub Next")))

	// Override the help function to hide completion command
	originalHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Hide completion command before displaying help
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "completion" {
				subCmd.Hidden = true
			}
		}
		originalHelpFunc(cmd, args)
	})

	// Create custom help command that supports "all" subcommand
	customHelpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type ` + constants.CLIExtensionPrefix + ` help [path to command] for full details.

Use "` + constants.CLIExtensionPrefix + ` help all" to show help for all commands.`,
		Run: func(c *cobra.Command, args []string) {
			// Check if the argument is "all"
			if len(args) == 1 && args[0] == "all" {
				// Print header
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Agentic Workflows CLI - Complete Command Reference"))
				fmt.Fprintln(os.Stderr, "")

				// Iterate through all commands and print their help
				for _, subCmd := range rootCmd.Commands() {
					// Skip hidden commands (like completion) and help itself
					if subCmd.Hidden || subCmd.Name() == "help" {
						continue
					}

					// Print command separator
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
					fmt.Fprintf(os.Stderr, "\n%s\n\n", console.FormatInfoMessage(fmt.Sprintf("Command: %s %s", constants.CLIExtensionPrefix, subCmd.Name())))

					// Print the command's help
					_ = subCmd.Help()
					fmt.Fprintln(os.Stderr, "")
				}

				// Print footer
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For more information, visit: https://githubnext.github.io/gh-aw/"))
				return
			}

			// Otherwise, use the default help behavior
			cmd, _, e := rootCmd.Find(args)
			if cmd == nil || e != nil {
				fmt.Fprintf(os.Stderr, "Unknown help topic [%#q]\n", args)
				_ = rootCmd.Usage()
			} else {
				cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
				_ = cmd.Help()
			}
		},
	}

	// Replace the default help command
	rootCmd.SetHelpCommand(customHelpCmd)

	// Create command instances
	addCmd := cli.NewAddCommand(cli.ValidateEngine)
	updateCmd := cli.NewUpdateCommand(cli.ValidateEngine)
	trialCmd := cli.NewTrialCommand(cli.ValidateEngine)
	initCmd := cli.NewInitCommand()
	newCmd := cli.NewNewCommand()
	removeCmd := cli.NewRemoveCommand()
	enableCmd := cli.NewEnableCommand()
	disableCmd := cli.NewDisableCommand()
	compileCmd := cli.NewCompileCommand()
	runCmd := cli.NewRunCommand()
	statusCmd := cli.NewStatusCommand()
	mcpCmd := cli.NewMCPCommand()
	logsCmd := cli.NewLogsCommand()
	auditCmd := cli.NewAuditCommand()
	versionCmd := cli.NewVersionCommand()

	// Assign commands to groups
	// Setup Commands
	initCmd.GroupID = "setup"
	newCmd.GroupID = "setup"
	addCmd.GroupID = "setup"

	// Development Commands
	compileCmd.GroupID = "development"
	mcpCmd.GroupID = "development"
	statusCmd.GroupID = "development"

	// Execution Commands
	runCmd.GroupID = "execution"
	enableCmd.GroupID = "execution"
	disableCmd.GroupID = "execution"

	// Analysis Commands
	logsCmd.GroupID = "analysis"
	auditCmd.GroupID = "analysis"

	// Add all commands to root
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(trialCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(cli.NewMCPServerCommand())
	rootCmd.AddCommand(cli.NewPRCommand())
	rootCmd.AddCommand(versionCmd)
}

func main() {
	// Set version information in the CLI package
	cli.SetVersionInfo(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		os.Exit(1)
	}
}
		ID:    "analysis",
		Title: "Analysis Commands:",
	})

	// Add global verbose flag to root command
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output showing detailed information")

	// Set output to stderr for consistency with CLI logging guidelines
	rootCmd.SetOut(os.Stderr)

	// Set version template to match the version subcommand format
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n%s\n",
		console.FormatInfoMessage(fmt.Sprintf("%s version {{.Version}}", constants.CLIExtensionPrefix)),
		console.FormatInfoMessage("GitHub Agentic Workflows CLI from GitHub Next")))

	// Override the help function to hide completion command
	originalHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Hide completion command before displaying help
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "completion" {
				subCmd.Hidden = true
			}
		}
		originalHelpFunc(cmd, args)
	})

	// Create custom help command that supports "all" subcommand
	customHelpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type ` + constants.CLIExtensionPrefix + ` help [path to command] for full details.

Use "` + constants.CLIExtensionPrefix + ` help all" to show help for all commands.`,
		Run: func(c *cobra.Command, args []string) {
			// Check if the argument is "all"
			if len(args) == 1 && args[0] == "all" {
				// Print header
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Agentic Workflows CLI - Complete Command Reference"))
				fmt.Fprintln(os.Stderr, "")

				// Iterate through all commands and print their help
				for _, subCmd := range rootCmd.Commands() {
					// Skip hidden commands (like completion) and help itself
					if subCmd.Hidden || subCmd.Name() == "help" {
						continue
					}

					// Print command separator
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
					fmt.Fprintf(os.Stderr, "\n%s\n\n", console.FormatInfoMessage(fmt.Sprintf("Command: %s %s", constants.CLIExtensionPrefix, subCmd.Name())))

					// Print the command's help
					_ = subCmd.Help()
					fmt.Fprintln(os.Stderr, "")
				}

				// Print footer
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For more information, visit: https://githubnext.github.io/gh-aw/"))
				return
			}

			// Otherwise, use the default help behavior
			cmd, _, e := rootCmd.Find(args)
			if cmd == nil || e != nil {
				fmt.Fprintf(os.Stderr, "Unknown help topic [%#q]\n", args)
				_ = rootCmd.Usage()
			} else {
				cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
				_ = cmd.Help()
			}
		},
	}

	// Replace the default help command
	rootCmd.SetHelpCommand(customHelpCmd)

	// Create and setup add command
	addCmd := cli.NewAddCommand(validateEngine)

	// Create and setup update command
	updateCmd := cli.NewUpdateCommand(validateEngine)

	// Create and setup trial command
	trialCmd := cli.NewTrialCommand(validateEngine)

	// Create and setup init command
	initCmd := cli.NewInitCommand()

	// Add flags to new command
	newCmd.Flags().Bool("force", false, "Overwrite existing workflow files")
	newCmd.Flags().BoolP("interactive", "i", false, "Launch interactive workflow creation wizard")

	// Add AI flag to compile and add commands
	compileCmd.Flags().StringP("engine", "e", "", "Override AI engine (claude, codex, copilot, custom)")
	compileCmd.Flags().Bool("validate", false, "Enable GitHub Actions workflow schema validation, container image validation, and action SHA validation")
	compileCmd.Flags().BoolP("watch", "w", false, "Watch for changes to workflow files and recompile automatically")
	compileCmd.Flags().String("dir", "", "Relative directory containing workflows (default: .github/workflows)")
	compileCmd.Flags().String("workflows-dir", "", "Deprecated: use --dir instead")
	_ = compileCmd.Flags().MarkDeprecated("workflows-dir", "use --dir instead")
	compileCmd.Flags().Bool("no-emit", false, "Validate workflow without generating lock files")
	compileCmd.Flags().Bool("purge", false, "Delete .lock.yml files that were not regenerated during compilation (only when no specific files are specified)")
	compileCmd.Flags().Bool("strict", false, "Enable strict mode: require timeout, refuse write permissions, require network configuration")
	compileCmd.Flags().Bool("trial", false, "Enable trial mode compilation (modifies workflows for trial execution)")
	compileCmd.Flags().String("logical-repo", "", "Repository to simulate workflow execution against (for trial mode)")
	compileCmd.Flags().Bool("dependabot", false, "Generate dependency manifests (package.json, requirements.txt, go.mod) and Dependabot config when dependencies are detected")
	compileCmd.Flags().Bool("force", false, "Force overwrite of existing files (e.g., dependabot.yml)")
	compileCmd.Flags().Bool("zizmor", false, "Run zizmor security scanner on generated .lock.yml files")
	compileCmd.Flags().Bool("poutine", false, "Run poutine security scanner on generated .lock.yml files")
	compileCmd.Flags().Bool("actionlint", false, "Run actionlint linter on generated .lock.yml files")
	compileCmd.Flags().Bool("json", false, "Output validation results as JSON")
	rootCmd.AddCommand(compileCmd)

	// Add flags to remove command
	removeCmd.Flags().Bool("keep-orphans", false, "Skip removal of orphaned include files that are no longer referenced by any workflow")

	// Add flags to run command
	runCmd.Flags().Int("repeat", 0, "Number of times to repeat running workflows (0 = run once)")
	runCmd.Flags().Bool("enable-if-needed", false, "Enable the workflow before running if needed, and restore state afterward")
	runCmd.Flags().StringP("engine", "e", "", "Override AI engine (claude, codex, copilot, custom)")
	runCmd.Flags().StringP("repo", "r", "", "Repository to run the workflow in (owner/repo format)")
	runCmd.Flags().Bool("auto-merge-prs", false, "Auto-merge any pull requests created during the workflow execution")
	runCmd.Flags().Bool("use-local-secrets", false, "Use local environment API key secrets for workflow execution (pushes and cleans up secrets in repository)")

	// Create and setup status command
	statusCmd := cli.NewStatusCommand()

	// Create commands that need group assignment
	mcpCmd := cli.NewMCPCommand()
	logsCmd := cli.NewLogsCommand()
	auditCmd := cli.NewAuditCommand()

	// Assign commands to groups
	// Setup Commands
	initCmd.GroupID = "setup"
	newCmd.GroupID = "setup"
	addCmd.GroupID = "setup"

	// Development Commands
	compileCmd.GroupID = "development"
	mcpCmd.GroupID = "development"
	statusCmd.GroupID = "development"

	// Execution Commands
	runCmd.GroupID = "execution"
	enableCmd.GroupID = "execution"
	disableCmd.GroupID = "execution"

	// Analysis Commands
	logsCmd.GroupID = "analysis"
	auditCmd.GroupID = "analysis"

	// Add all commands to root
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(trialCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(initCmd)

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(cli.NewMCPServerCommand())
	rootCmd.AddCommand(cli.NewPRCommand())
	rootCmd.AddCommand(versionCmd)
}

func main() {
	// Set version information in the CLI package
	cli.SetVersionInfo(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		os.Exit(1)
	}
}
