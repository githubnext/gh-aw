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
var verbose bool

// validateEngine validates the engine flag value
func validateEngine(engine string) error {
	if engine != "" && engine != "claude" && engine != "codex" && engine != "gemini" {
		return fmt.Errorf("invalid engine value '%s'. Must be 'claude', 'codex', or 'gemini'", engine)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   constants.CLIExtensionPrefix,
	Short: "GitHub Agentic Workflows CLI from GitHub Next",
	Long: ` = GitHub Agentic Workflows from GitHub Next

A natural language GitHub Action is a markdown file checked into the .github/workflows directory of a repository.
The file contains a natural language description of the workflow, which is then compiled into a GitHub Actions workflow file.
The workflow file is then executed by GitHub Actions in response to events in the repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available engines, workflows and installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		packages, _ := cmd.Flags().GetBool("packages")
		local, _ := cmd.Flags().GetBool("local")
		if packages {
			if err := cli.ListPackages(local, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		} else {
			if err := cli.ListWorkflows(verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		}
	},
}

var addCmd = &cobra.Command{
	Use:   "add <workflow>",
	Short: "Add a workflow from the components to .github/workflows",
	Long: `Add a workflow from the components to .github/workflows.

Examples:
  ` + constants.CLIExtensionPrefix + ` add weekly-research
  ` + constants.CLIExtensionPrefix + ` add weekly-research -n my-custom-name
  ` + constants.CLIExtensionPrefix + ` add weekly-research -r githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` add weekly-research --pr
  ` + constants.CLIExtensionPrefix + ` add weekly-research --force

The -r flag allows you to install and use workflows from a specific repository.
The -n flag allows you to specify a custom name for the workflow file.
The --pr flag automatically creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.
It's a shortcut for:
  ` + constants.CLIExtensionPrefix + ` install githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` add weekly-research`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflow := args[0]
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

		if prFlag {
			if err := cli.AddWorkflowWithRepoAndPR(workflow, numberFlag, verbose, engineOverride, repoFlag, nameFlag, forceFlag); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		} else {
			if err := cli.AddWorkflowWithRepo(workflow, numberFlag, verbose, engineOverride, repoFlag, nameFlag, forceFlag); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		}
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
		if err := cli.StatusWorkflows(pattern, verbose); err != nil {
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
	Use:   "compile [markdown-file]",
	Short: "Compile markdown to YAML workflows",
	Run: func(cmd *cobra.Command, args []string) {
		var file string
		if len(args) > 0 {
			file = args[0]
		}
		engineOverride, _ := cmd.Flags().GetString("engine")
		validate, _ := cmd.Flags().GetBool("validate")
		autoCompile, _ := cmd.Flags().GetBool("auto-compile")
		watch, _ := cmd.Flags().GetBool("watch")
		instructions, _ := cmd.Flags().GetBool("instructions")
		if err := validateEngine(engineOverride); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
		if err := cli.CompileWorkflows(file, verbose, engineOverride, validate, autoCompile, watch, instructions); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			os.Exit(1)
		}
	},
}

var runCmd = &cobra.Command{
	Use:   "run <workflow-id-or-name>",
	Short: "Run an agentic workflow on GitHub Actions",
	Long: `Run an agentic workflow on GitHub Actions using the workflow_dispatch trigger.

This command accepts either a workflow ID or an agentic workflow name.
The workflow must have been added as an action and compiled.

This command only works with workflows that have workflow_dispatch triggers.
It executes 'gh workflow run <workflow-lock-file>' to trigger the workflow on GitHub Actions.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflowIdOrName := args[0]
		if err := cli.RunWorkflowOnGitHub(workflowIdOrName, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
				Type:    "error",
				Message: fmt.Sprintf("running workflow on GitHub Actions: %v", err),
			}))
			os.Exit(1)
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install <org/repo>[@version]",
	Short: "Install agent workflows from a GitHub repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoSpec := args[0]
		local, _ := cmd.Flags().GetBool("local")
		if err := cli.InstallPackage(repoSpec, local, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
				Type:    "error",
				Message: fmt.Sprintf("installing package: %v", err),
			}))
			os.Exit(1)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <org/repo>",
	Short: "Uninstall agent workflows package",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoSpec := args[0]
		local, _ := cmd.Flags().GetBool("local")
		if err := cli.UninstallPackage(repoSpec, local, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
				Type:    "error",
				Message: fmt.Sprintf("uninstalling package: %v", err),
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output showing detailed information")

	// Add number flag to add command
	addCmd.Flags().IntP("number", "c", 1, "Create multiple numbered copies")

	// Add name flag to add command
	addCmd.Flags().StringP("name", "n", "", "Specify name for the added workflow (without .md extension)")

	// Add AI flag to add command
	addCmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, gemini)")

	// Add repository flag to add command
	addCmd.Flags().StringP("repo", "r", "", "Install and use workflows from specified repository (org/repo)")

	// Add PR flag to add command
	addCmd.Flags().Bool("pr", false, "Create a pull request with the workflow changes")

	// Add force flag to add command
	addCmd.Flags().Bool("force", false, "Overwrite existing workflow files")

	// Add force flag to new command
	newCmd.Flags().Bool("force", false, "Overwrite existing workflow files")

	// Add packages flag to list command
	listCmd.Flags().BoolP("packages", "p", false, "List installed packages instead of available workflows")
	listCmd.Flags().BoolP("local", "l", false, "List local packages instead of global packages (requires --packages)")

	// Add local flag to install command
	installCmd.Flags().BoolP("local", "l", false, "Install packages locally in .aw/packages instead of globally in ~/.aw/packages")

	// Add local flag to uninstall command
	uninstallCmd.Flags().BoolP("local", "l", false, "Uninstall packages from local .aw/packages instead of global ~/.aw/packages")

	// Add AI flag to compile and add commands
	compileCmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, gemini)")
	compileCmd.Flags().Bool("validate", false, "Enable GitHub Actions workflow schema validation")
	compileCmd.Flags().Bool("auto-compile", false, "Generate auto-compile workflow file for automatic compilation")
	compileCmd.Flags().BoolP("watch", "w", false, "Watch for changes to workflow files and recompile automatically")
	compileCmd.Flags().Bool("instructions", false, "Generate or update GitHub Copilot instructions file")

	// Add flags to remove command
	removeCmd.Flags().Bool("keep-orphans", false, "Skip removal of orphaned include files that are no longer referenced by any workflow")

	// Add all commands to root
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(cli.NewLogsCommand())
	rootCmd.AddCommand(cli.NewInspectCommand())
	rootCmd.AddCommand(versionCmd)
}

func main() {
	// Set version information in the CLI package
	cli.SetVersionInfo(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(console.FormatErrorMessage(err.Error()))
		os.Exit(1)
	}
}
