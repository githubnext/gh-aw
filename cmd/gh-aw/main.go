package main

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// Build-time variables set by GoReleaser
var (
	version = "dev"
)

// Global flags
var verboseFlag bool

// validateEngine validates the engine flag value
func validateEngine(engine string) error {
	if engine != "" && engine != "claude" && engine != "codex" && engine != "copilot" {
		return fmt.Errorf("invalid engine value '%s'. Must be 'claude', 'codex', or 'copilot'", engine)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:     constants.CLIExtensionPrefix,
	Short:   "GitHub Agentic Workflows CLI from GitHub Next",
	Version: version,
	Long: `GitHub Agentic Workflows from GitHub Next

A natural language GitHub Action is a markdown file checked into the .github/workflows directory of a repository.
The file contains a natural language description of the workflow, which is then compiled into a GitHub Actions workflow file.
The workflow file is then executed by GitHub Actions in response to events in the repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var newCmd = &cobra.Command{
	Use:   "new <workflow-base-name>",
	Short: "Create a new workflow markdown file with example configuration",
	Long: `Create a new workflow markdown file with commented examples and explanations of all available options.

The created file will include comprehensive examples of:
- All trigger types (on: events)
- Permissions configuration
- AI processor settings
- Tools configuration (github, claude, mcps)
- All frontmatter options with explanations

Examples:
  ` + constants.CLIExtensionPrefix + ` new my-workflow
  ` + constants.CLIExtensionPrefix + ` new issue-handler --force`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflowName := args[0]
		forceFlag, _ := cmd.Flags().GetBool("force")
		verbose, _ := cmd.Flags().GetBool("verbose")
		if err := cli.NewWorkflow(workflowName, verbose, forceFlag); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [pattern]",
	Short: "Remove workflow files matching the given name prefix",
	Run: func(cmd *cobra.Command, args []string) {
		var pattern string
		if len(args) > 0 {
			pattern = args[0]
		}
		keepOrphans, _ := cmd.Flags().GetBool("keep-orphans")
		if err := cli.RemoveWorkflows(pattern, keepOrphans); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [pattern]",
	Short: "Show status of natural language action files and workflows",
	Run: func(cmd *cobra.Command, args []string) {
		var pattern string
		if len(args) > 0 {
			pattern = args[0]
		}
		if err := cli.StatusWorkflows(pattern, verboseFlag); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable [pattern]",
	Short: "Enable natural language action workflows",
	Run: func(cmd *cobra.Command, args []string) {
		var pattern string
		if len(args) > 0 {
			pattern = args[0]
		}
		if err := cli.EnableWorkflows(pattern); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable [pattern]",
	Short: "Disable natural language action workflows and cancel any in-progress runs",
	Run: func(cmd *cobra.Command, args []string) {
		var pattern string
		if len(args) > 0 {
			pattern = args[0]
		}
		if err := cli.DisableWorkflows(pattern); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var compileCmd = &cobra.Command{
	Use:   "compile [markdown-file]...",
	Short: "Compile markdown to YAML workflows",
	Long: `Compile one or more markdown workflow files to YAML workflows.

If no files are specified, all markdown files in .github/workflows will be compiled.

Examples:
  ` + constants.CLIExtensionPrefix + ` compile                    # Compile all markdown files
  ` + constants.CLIExtensionPrefix + ` compile ci-doctor    # Compile a specific workflow
  ` + constants.CLIExtensionPrefix + ` compile ci-doctor daily-plan  # Compile multiple workflows
  ` + constants.CLIExtensionPrefix + ` compile workflow.md        # Compile by file path
  ` + constants.CLIExtensionPrefix + ` compile --workflows-dir custom/workflows  # Compile from custom directory
  ` + constants.CLIExtensionPrefix + ` compile --watch ci-doctor     # Watch and auto-compile`,
	Run: func(cmd *cobra.Command, args []string) {
		engineOverride, _ := cmd.Flags().GetString("engine")
		validate, _ := cmd.Flags().GetBool("validate")
		watch, _ := cmd.Flags().GetBool("watch")
		workflowDir, _ := cmd.Flags().GetString("workflows-dir")
		noEmit, _ := cmd.Flags().GetBool("no-emit")
		purge, _ := cmd.Flags().GetBool("purge")
		strict, _ := cmd.Flags().GetBool("strict")
		verbose, _ := cmd.Flags().GetBool("verbose")
		if err := validateEngine(engineOverride); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
		config := cli.CompileConfig{
			MarkdownFiles:     args,
			Verbose:           verbose,
			EngineOverride:    engineOverride,
			Validate:          validate,
			Watch:             watch,
			WorkflowDir:       workflowDir,
			SkipInstructions:  false, // Deprecated field, kept for backward compatibility
			NoEmit:            noEmit,
			Purge:             purge,
			TrialMode:         false,
			SimulatedRepoSlug: "",
			Strict:            strict,
		}
		if _, err := cli.CompileWorkflows(config); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var runCmd = &cobra.Command{
	Use:   "run <workflow-id-or-name>...",
	Short: "Run one or more agentic workflows on GitHub Actions",
	Long: `Run one or more agentic workflows on GitHub Actions using the workflow_dispatch trigger.

This command accepts one or more workflow IDs or agentic workflow names.
The workflows must have been added as actions and compiled.

This command only works with workflows that have workflow_dispatch triggers.
It executes 'gh workflow run <workflow-lock-file>' to trigger each workflow on GitHub Actions.

Examples:
  gh aw run daily-perf-improver
  gh aw run daily-perf-improver --repeat 3600  # Run every hour
  gh aw run daily-perf-improver --enable-if-needed # Enable if disabled, run, then restore state`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repeatSeconds, _ := cmd.Flags().GetInt("repeat")
		enable, _ := cmd.Flags().GetBool("enable-if-needed")
		if err := cli.RunWorkflowsOnGitHub(args, repeatSeconds, enable, verboseFlag); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
				Type:    "error",
				Message: fmt.Sprintf("running workflows on GitHub Actions: %v", err),
			}))
			os.Exit(1)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("%s version %s", constants.CLIExtensionPrefix, version)))
		fmt.Println(console.FormatInfoMessage("GitHub Agentic Workflows CLI from GitHub Next"))
	},
}

func init() {
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

	// Create and setup add command
	addCmd := cli.NewAddCommand(validateEngine)

	// Create and setup update command
	updateCmd := cli.NewUpdateCommand(validateEngine)

	// Create and setup trial command
	trialCmd := cli.NewTrialCommand(validateEngine)

	// Create and setup init command
	initCmd := NewInitCommand()

	// Add force flag to new command
	newCmd.Flags().Bool("force", false, "Overwrite existing workflow files")

	// Add AI flag to compile and add commands
	compileCmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, copilot)")
	compileCmd.Flags().Bool("validate", true, "Enable GitHub Actions workflow schema validation (default: true)")
	compileCmd.Flags().BoolP("watch", "w", false, "Watch for changes to workflow files and recompile automatically")
	compileCmd.Flags().String("workflows-dir", "", "Relative directory containing workflows (default: .github/workflows)")
	compileCmd.Flags().Bool("no-emit", false, "Validate workflow without generating lock files")
	compileCmd.Flags().Bool("purge", false, "Delete .lock.yml files that were not regenerated during compilation (only when no specific files are specified)")
	compileCmd.Flags().Bool("strict", false, "Enable strict mode: require timeout, refuse write permissions, require network configuration")

	// Add flags to remove command
	removeCmd.Flags().Bool("keep-orphans", false, "Skip removal of orphaned include files that are no longer referenced by any workflow")

	// Add flags to run command
	runCmd.Flags().Int("repeat", 0, "Repeat running workflows every SECONDS (0 = run once)")
	runCmd.Flags().Bool("enable-if-needed", false, "Enable the workflow before running if needed, and restore state afterward")

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
	rootCmd.AddCommand(cli.NewLogsCommand())
	rootCmd.AddCommand(cli.NewAuditCommand())
	rootCmd.AddCommand(cli.NewMCPCommand())
	rootCmd.AddCommand(cli.NewMCPServerCommand())
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
