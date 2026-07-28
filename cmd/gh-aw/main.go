package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/github/gh-aw/pkg/cli"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// Build-time variables set by GoReleaser
var (
	version   = "dev"
	isRelease = "false" // Set to "true" during release builds
)

// Global flags
var verboseFlag bool
var bannerFlag bool

// formatListWithOr formats a list of strings with commas and "or" before the last item
// Example: ["a", "b", "c"] -> "a, b, or c"
func formatListWithOr(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " or " + items[1]
	}
	// For 3+ items: "a, b, or c"
	return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
}

// validateEngine validates the engine flag value
func validateEngine(engine string) error {
	// Get the global engine registry
	registry := workflow.GetGlobalEngineRegistry()
	validEngines := registry.GetSupportedEngines()

	if engine != "" && !registry.IsValidEngine(engine) {
		// Sort engines for deterministic output
		sortedEngines := make([]string, len(validEngines))
		copy(sortedEngines, validEngines)
		sort.Strings(sortedEngines)

		// Format engines with quotes and "or" conjunction
		quotedEngines := make([]string, len(sortedEngines))
		for i, e := range sortedEngines {
			quotedEngines[i] = "'" + e + "'"
		}
		formattedList := formatListWithOr(quotedEngines)

		// Try to find close matches for "did you mean" suggestion
		suggestions := parser.FindClosestMatches(engine, validEngines, 1)

		errMsg := fmt.Sprintf("invalid engine value '%s'. Must be %s", engine, formattedList)

		if len(suggestions) > 0 {
			errMsg = fmt.Sprintf("invalid engine value '%s'. Must be %s.\n\nDid you mean: %s?",
				engine, formattedList, suggestions[0])
		}

		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:     string(constants.CLIExtensionPrefix),
	Short:   "GitHub Agentic Workflows CLI",
	Version: version,
	Long: `GitHub Agentic Workflows CLI

Common Tasks:
  ` + string(constants.CLIExtensionPrefix) + ` init                  		# Set up a new repository
  ` + string(constants.CLIExtensionPrefix) + ` doctor --repo owner/repo 		# Run diagnostics for authentication and repository setup
  ` + string(constants.CLIExtensionPrefix) + ` add-wizard            		# Add workflows with interactive guided setup
  ` + string(constants.CLIExtensionPrefix) + ` new my-workflow       		# Create your first workflow
  ` + string(constants.CLIExtensionPrefix) + ` compile               		# Compile all workflows
  ` + string(constants.CLIExtensionPrefix) + ` run my-workflow       		# Execute a workflow
  ` + string(constants.CLIExtensionPrefix) + ` status                		# Check workflow status
  ` + string(constants.CLIExtensionPrefix) + ` logs my-workflow      		# Download and analyze execution logs
  ` + string(constants.CLIExtensionPrefix) + ` audit <run-id-or-url> 		# Audit and compare workflow runs

For detailed help on any command, use:
  ` + string(constants.CLIExtensionPrefix) + ` [command] --help`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cli.ConfigureProjectTimezone()
		if bannerFlag {
			console.PrintBanner()
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var newCmd = &cobra.Command{
	Use:   "new [workflow]",
	Short: "Create a new agentic workflow file with example configuration",
	Long: `Create a new agentic workflow file with example configuration and explanations of all available options.

When called without a workflow name (or with --interactive flag), launches an interactive wizard
to guide you through creating a workflow with custom settings.

When called with a workflow name, creates a template file with comprehensive examples of:
- All trigger types (on: events)
- Permissions configuration
- AI engine settings
- Tools configuration (GitHub, Claude, MCPs)
- All frontmatter options with explanations

` + cli.WorkflowIDExplanation,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` new                      # Interactive mode
  ` + string(constants.CLIExtensionPrefix) + ` new my-workflow          # Create template file
  ` + string(constants.CLIExtensionPrefix) + ` new my-workflow.md       # Same as above (.md extension stripped)
  ` + string(constants.CLIExtensionPrefix) + ` new my-workflow --force  # Overwrite if exists
  ` + string(constants.CLIExtensionPrefix) + ` new my-workflow --engine copilot  # Create template with specific engine`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		forceFlag, _ := cmd.Flags().GetBool("force")
		verbose, _ := cmd.Flags().GetBool("verbose")
		interactiveFlag, _ := cmd.Flags().GetBool("interactive")
		engineOverride, _ := cmd.Flags().GetString("engine")

		if engineOverride != "" {
			if err := validateEngine(engineOverride); err != nil {
				return err
			}
		}

		// If no arguments provided or interactive flag is set, use interactive mode
		if len(args) == 0 || interactiveFlag {
			// Check if running in CI environment
			if cli.IsRunningInCI() {
				return errors.New("interactive mode cannot be used in CI environments. Please provide a workflow name")
			}

			// Use default workflow name for interactive mode
			workflowName := "my-workflow"
			if len(args) > 0 {
				workflowName = args[0]
			}

			return cli.CreateWorkflowInteractively(cmd.Context(), workflowName, verbose, forceFlag)
		}

		// Template mode with workflow name
		workflowName := args[0]
		return cli.CreateWorkflowMarkdownFile(workflowName, verbose, forceFlag, engineOverride)
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [filter]",
	Short: "Remove agentic workflow files matching the given filter",
	Long: `Remove agentic workflow files matching the given filter.

The workflow-id is the basename of the Markdown file without the .md extension.
You can provide a substring to match multiple workflows, or a specific workflow-id.

By default, this command also removes orphaned include files that are no longer referenced
by any workflow. Use --no-remove-orphans to skip this cleanup.`,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` remove my-workflow                    # Remove specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` remove test-                          # Remove all workflows containing 'test-' in name
  ` + string(constants.CLIExtensionPrefix) + ` remove old- --no-remove-orphans       # Remove workflows but keep orphaned includes
  ` + string(constants.CLIExtensionPrefix) + ` remove my-workflow --dir .github/workflows/shared  # Remove from custom directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pattern string
		if len(args) > 0 {
			pattern = args[0]
		}
		keepOrphans, _ := cmd.Flags().GetBool("keep-orphans")
		noRemoveOrphans, _ := cmd.Flags().GetBool("no-remove-orphans")
		keepOrphans = keepOrphans || noRemoveOrphans
		workflowDir, _ := cmd.Flags().GetString("dir")
		return cli.RemoveWorkflows(pattern, keepOrphans, workflowDir)
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable [workflow]...",
	Short: "Enable agentic workflows",
	Long: `Enable one or more workflows by ID, or all workflows if no IDs are provided.

` + cli.WorkflowIDExplanation,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` enable                   # Enable all workflows
  ` + string(constants.CLIExtensionPrefix) + ` enable ci-doctor         # Enable specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` enable ci-doctor.md      # Enable specific workflow (alternative format)
  ` + string(constants.CLIExtensionPrefix) + ` enable ci-doctor daily   # Enable multiple workflows
  ` + string(constants.CLIExtensionPrefix) + ` enable ci-doctor --repo owner/repo  # Enable workflow in specific repository`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoOverride, _ := cmd.Flags().GetString("repo")
		return cli.EnableWorkflowsByNames(cmd.Context(), args, repoOverride)
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable [workflow]...",
	Short: "Disable agentic workflows",
	Long: `Disable one or more workflows by ID, or all workflows if no IDs are provided.

Any in-progress runs will be canceled before disabling.

` + cli.WorkflowIDExplanation,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` disable                   # Disable all workflows
  ` + string(constants.CLIExtensionPrefix) + ` disable ci-doctor         # Disable specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` disable ci-doctor.md      # Disable specific workflow (alternative format)
  ` + string(constants.CLIExtensionPrefix) + ` disable ci-doctor daily   # Disable multiple workflows
  ` + string(constants.CLIExtensionPrefix) + ` disable ci-doctor --repo owner/repo  # Disable workflow in specific repository`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoOverride, _ := cmd.Flags().GetString("repo")
		return cli.DisableWorkflowsByNames(cmd.Context(), args, repoOverride)
	},
}

var compileCmd = &cobra.Command{
	Use:   "compile [workflow]...",
	Short: "Compile agentic workflow Markdown files into GitHub Actions YAML",
	Long: `Compile agentic workflow Markdown files into GitHub Actions YAML.

If no workflows are specified, all Markdown files in .github/workflows will be compiled.

` + cli.WorkflowIDExplanation + `

The --dependabot flag generates dependency manifests when dependencies are detected:
  - For npm: Creates package.json and package-lock.json (requires npm in PATH)
  - For Python: Creates requirements.txt for pip packages
  - For Go: Creates go.mod for go install/get packages
  - For all detected ecosystems: Generates .github/dependabot.yml
  - Use --force to overwrite existing dependabot.yml
  - Cannot be used with specific workflow files or custom --dir
  - Only processes workflows in the default .github/workflows directory

Action mode controls how gh-aw action scripts are referenced in compiled workflows.
Three flags govern this. --gh-aw-ref is mutually exclusive with the other two;
--action-tag and --action-mode may be combined (e.g. --action-mode action --action-tag v1.2.3):

Unlike ` + "`gh aw upgrade`" + `, ` + "`gh aw compile`" + ` only applies codemods when you opt in with ` + "`--fix`" + `.

  --action-mode <mode>
    Explicit mode selection. Values:
      dev      Local paths (./actions/...). For developing inside the gh-aw repo.
      release  SHA-pinned refs from github/gh-aw (e.g. github/gh-aw/actions/setup@<sha>).
               The SHA is derived from the binary version or from --action-tag.
      action   SHA-pinned refs from the github/gh-aw-actions repository.
               Used by release binaries. Can be combined with --action-tag to pin a version.
    Auto-detected from the binary build type when not set.

  --action-tag <sha-or-tag>
    Pin to a specific SHA or version tag (e.g. v1, v1.2.3, <full-sha>).
    Implies --action-mode release unless --action-mode action is also specified.
    The value is used as-is without SHA resolution. Use --gh-aw-ref to resolve
    branches or tags at compile time.

  --gh-aw-ref <branch-tag-or-sha>
    Resolve a branch name, tag, or SHA from github/gh-aw to its full commit SHA
    at compile time and pin the compiled workflow to that immutable SHA.
    Equivalent to --action-mode release --action-tag <resolved-sha>.
    Branch and tag names are resolved via the GitHub API.
    Cannot be combined with --action-tag or --action-mode.
    Use this when E2E-testing compiled workflows against a specific gh-aw revision.`,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` compile                    # Compile all Markdown files
  ` + string(constants.CLIExtensionPrefix) + ` compile ci-doctor          # Compile a specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` compile ci-doctor daily-plan  # Compile multiple workflows
  ` + string(constants.CLIExtensionPrefix) + ` compile workflow.md        # Compile by file path
  ` + string(constants.CLIExtensionPrefix) + ` compile .github/workflows  # Compile all workflows in a directory
  ` + string(constants.CLIExtensionPrefix) + ` compile --dir custom/workflows  # Compile from custom directory
  ` + string(constants.CLIExtensionPrefix) + ` compile ci-doctor --watch     # Watch and auto-compile
  ` + string(constants.CLIExtensionPrefix) + ` compile --trial --logical-repo owner/repo  # Compile for trial mode
  ` + string(constants.CLIExtensionPrefix) + ` compile --dependabot        # Generate Dependabot manifests
  ` + string(constants.CLIExtensionPrefix) + ` compile --dependabot --force  # Force overwrite existing dependabot.yml
  ` + string(constants.CLIExtensionPrefix) + ` compile --gh-aw-ref main       # Pin workflows to the SHA of github/gh-aw main at compile time
  ` + string(constants.CLIExtensionPrefix) + ` compile --action-tag v1.2.3    # Pin workflows to a specific release tag`,
	RunE: runCompileCommand,
}

func runCompileCommand(cmd *cobra.Command, args []string) error {
	flags, err := readCompileFlags(cmd)
	if err != nil {
		return err
	}
	if err := validateEngine(flags.EngineOverride); err != nil {
		return err
	}

	finishCompileUpdateCheck := cli.StartCompileUpdateCheck(cmd.Context(), flags.NoCheckUpdate, flags.Verbose)
	defer finishCompileUpdateCheck()

	if err := runCompileFixIfRequested(args, flags); err != nil {
		return err
	}
	if _, err := cli.CompileWorkflows(cmd.Context(), buildCompileConfig(args, flags)); err != nil {
		// Return error as-is without additional formatting
		// Errors from CompileWorkflows are already formatted with console.FormatError
		// which provides IDE-parseable location information (file:line:column)
		return err
	}
	return nil
}

func runCompileFixIfRequested(args []string, flags compileFlagValues) error {
	if !flags.Fix {
		return nil
	}
	return cli.RunFix(cli.FixConfig{
		WorkflowIDs: args,
		Write:       true,
		Verbose:     flags.Verbose,
		WorkflowDir: flags.Dir,
	})
}

type compileFlagValues struct {
	EngineOverride, ActionMode, ActionTag, ActionsRepo, Dir, WorkflowsDir, LogicalRepo string
	ScheduleSeed, PriorManifestFile                                                    string
	Validate, Watch, NoEmit, Purge, Strict, Trial, Dependabot, ForceOverwrite          bool
	RefreshStopTime, ForceRefreshActionPins, AllowActionRefs                           bool
	Zizmor, Poutine, Actionlint, RunnerGuard, Syft, Grype, Grant, Yamllint             bool
	JSONOutput, ShowAllErrors, Fix, Stats, FailFast, NoCheckUpdate                     bool
	Staged, Approve, ValidateImages, DisableModelsDevLookup, GHESCompat, Verbose       bool
	UseSamples                                                                         bool
}

func readCompileFlags(cmd *cobra.Command) (compileFlagValues, error) {
	flags := compileFlagValues{
		EngineOverride:         flagString(cmd, "engine"),
		ActionMode:             flagString(cmd, "action-mode"),
		ActionTag:              flagString(cmd, "action-tag"),
		ActionsRepo:            flagString(cmd, "actions-repo"),
		Validate:               flagBool(cmd, "validate"),
		Watch:                  flagBool(cmd, "watch"),
		Dir:                    flagString(cmd, "dir"),
		WorkflowsDir:           flagString(cmd, "workflows-dir"),
		NoEmit:                 flagBool(cmd, "no-emit"),
		Purge:                  flagBool(cmd, "purge"),
		Strict:                 flagBool(cmd, "strict"),
		Trial:                  flagBool(cmd, "trial"),
		LogicalRepo:            flagString(cmd, "logical-repo"),
		Dependabot:             flagBool(cmd, "dependabot"),
		ForceOverwrite:         flagBool(cmd, "force"),
		RefreshStopTime:        flagBool(cmd, "refresh-stop-time"),
		ForceRefreshActionPins: flagBool(cmd, "force-refresh-action-pins"),
		AllowActionRefs:        flagBool(cmd, "allow-action-refs"),
		Zizmor:                 flagBool(cmd, "zizmor"),
		Poutine:                flagBool(cmd, "poutine"),
		Actionlint:             flagBool(cmd, "actionlint"),
		RunnerGuard:            flagBool(cmd, "runner-guard"),
		Syft:                   flagBool(cmd, "syft"),
		Grype:                  flagBool(cmd, "grype"),
		Grant:                  flagBool(cmd, "grant"),
		Yamllint:               flagBool(cmd, "yamllint"),
		JSONOutput:             flagBool(cmd, "json"),
		ShowAllErrors:          flagBool(cmd, "show-all"),
		Fix:                    flagBool(cmd, "fix"),
		Stats:                  flagBool(cmd, "stats"),
		FailFast:               flagBool(cmd, "fail-fast"),
		NoCheckUpdate:          flagBool(cmd, "no-check-update"),
		ScheduleSeed:           flagString(cmd, "schedule-seed"),
		Staged:                 flagBool(cmd, "staged"),
		Approve:                flagBool(cmd, "approve"),
		ValidateImages:         flagBool(cmd, "validate-images"),
		DisableModelsDevLookup: flagBool(cmd, "no-models-dev-lookup"),
		PriorManifestFile:      flagString(cmd, "prior-manifest-file"),
		GHESCompat:             flagBool(cmd, "ghes"),
		Verbose:                flagBool(cmd, "verbose"),
		UseSamples:             flagBool(cmd, "use-samples"),
	}
	return resolveCompileGhAwRef(cmd, flags)
}

func resolveCompileGhAwRef(cmd *cobra.Command, flags compileFlagValues) (compileFlagValues, error) {
	ghAwRef := flagString(cmd, "gh-aw-ref")
	if ghAwRef == "" {
		return flags, nil
	}
	resolvedRef, err := workflow.ResolveGhAwRef(cmd.Context(), ghAwRef)
	if err != nil {
		return compileFlagValues{}, fmt.Errorf("--gh-aw-ref: %w", err)
	}
	flags.ActionMode = string(workflow.ActionModeRelease)
	flags.ActionTag = resolvedRef
	return flags, nil
}

func buildCompileConfig(args []string, flags compileFlagValues) cli.CompileConfig {
	workflowDir := flags.Dir
	if flags.WorkflowsDir != "" {
		workflowDir = flags.WorkflowsDir
	}
	return cli.CompileConfig{
		MarkdownFiles:          args,
		Verbose:                flags.Verbose,
		EngineOverride:         flags.EngineOverride,
		ActionMode:             flags.ActionMode,
		ActionTag:              flags.ActionTag,
		ActionsRepo:            flags.ActionsRepo,
		Validate:               flags.Validate,
		Watch:                  flags.Watch,
		WorkflowDir:            workflowDir,
		SkipInstructions:       false, // Deprecated field, kept for backward compatibility
		NoEmit:                 flags.NoEmit,
		Purge:                  flags.Purge,
		TrialMode:              flags.Trial,
		TrialLogicalRepoSlug:   flags.LogicalRepo,
		Strict:                 flags.Strict,
		Dependabot:             flags.Dependabot,
		ForceOverwrite:         flags.ForceOverwrite,
		RefreshStopTime:        flags.RefreshStopTime,
		ForceRefreshActionPins: flags.ForceRefreshActionPins,
		AllowActionRefs:        flags.AllowActionRefs,
		Zizmor:                 flags.Zizmor,
		Poutine:                flags.Poutine,
		Actionlint:             flags.Actionlint,
		RunnerGuard:            flags.RunnerGuard,
		Syft:                   flags.Syft,
		Grype:                  flags.Grype,
		Grant:                  flags.Grant,
		Yamllint:               flags.Yamllint,
		JSONOutput:             flags.JSONOutput,
		ShowAllErrors:          flags.ShowAllErrors,
		Stats:                  flags.Stats,
		FailFast:               flags.FailFast,
		ScheduleSeed:           flags.ScheduleSeed,
		Staged:                 flags.Staged,
		Approve:                flags.Approve,
		ValidateImages:         flags.ValidateImages,
		DisableModelsDevLookup: flags.DisableModelsDevLookup,
		PriorManifestFile:      flags.PriorManifestFile,
		GHESCompat:             flags.GHESCompat,
		UseSamples:             flags.UseSamples,
	}
}

func flagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func flagBool(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}

var runCmd = &cobra.Command{
	Use:   "run [workflow]...",
	Short: "Run one or more agentic workflows on GitHub Actions",
	Long: `Run one or more agentic workflows on GitHub Actions using the workflow_dispatch trigger.

When called without workflow arguments, this command enters interactive mode and shows:
- List of workflows that support workflow_dispatch
- Display of required and optional inputs
- Input collection with validation
- Command display for future reference

This command accepts one or more workflow IDs.
The workflows must have been compiled into GitHub Actions YAML files.

This command only works with workflows that have workflow_dispatch triggers.

` + cli.WorkflowIDExplanation,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` run                          # Interactive mode
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver.md   # Alternative format
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --ref main  # Run on specific branch
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --repeat 3  # Run 4 times total (1 initial + 3 repeats)
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --enable-if-needed  # Enable if disabled, run, then restore state
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --auto-merge-prs  # Auto-merge any PRs created during execution
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --raw-field name=value --raw-field env=prod  # Pass workflow inputs
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --push  # Commit, push, and dispatch the workflow
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --dry-run  # Preview without triggering workflow runs
  ` + string(constants.CLIExtensionPrefix) + ` run daily-perf-improver --json  # Output results in JSON format`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repeatCount, _ := cmd.Flags().GetInt("repeat")
		enable, _ := cmd.Flags().GetBool("enable-if-needed")
		engineOverride, _ := cmd.Flags().GetString("engine")
		repoOverride, _ := cmd.Flags().GetString("repo")
		refOverride, _ := cmd.Flags().GetString("ref")
		autoMergePRs, _ := cmd.Flags().GetBool("auto-merge-prs")
		inputs, _ := cmd.Flags().GetStringArray("raw-field")
		push, _ := cmd.Flags().GetBool("push")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		approveRun, _ := cmd.Flags().GetBool("approve")

		if err := validateEngine(engineOverride); err != nil {
			return err
		}

		// If no arguments provided, enter interactive mode
		if len(args) == 0 {
			// Check if running in CI environment
			if cli.IsRunningInCI() {
				return errors.New("interactive mode cannot be used in CI environments. Please provide a workflow name")
			}

			// Interactive mode doesn't support repeat or enable flags
			if repeatCount > 0 {
				return errors.New("--repeat flag is not supported in interactive mode")
			}
			if enable {
				return errors.New("--enable-if-needed flag is not supported in interactive mode")
			}
			if len(inputs) > 0 {
				return errors.New("workflow inputs cannot be specified in interactive mode (they will be collected interactively)")
			}

			return cli.RunWorkflowInteractively(cmd.Context(), verboseFlag, repoOverride, refOverride, autoMergePRs, push, engineOverride, dryRun, approveRun)
		}

		return cli.RunWorkflowsOnGitHub(cmd.Context(), args, cli.RunOptions{
			RepeatCount:    repeatCount,
			Enable:         enable,
			EngineOverride: engineOverride,
			RepoOverride:   repoOverride,
			RefOverride:    refOverride,
			AutoMergePRs:   autoMergePRs,
			Push:           push,
			Inputs:         inputs,
			Verbose:        verboseFlag,
			DryRun:         dryRun,
			JSON:           jsonOutput,
			Approve:        approveRun,
		})
	},
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print the current version",
	Long:    `Print the current version and build information for the gh aw CLI extension.`,
	Example: `  ` + string(constants.CLIExtensionPrefix) + ` version   # Print the current version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(os.Stderr, "%s version %s\n", string(constants.CLIExtensionPrefix), version)
		return nil
	},
}

func init() {
	configureRootCommand()
	configureRootUsage()
	rootCmd.SetHelpCommand(newCustomHelpCommand())

	commands := createCommandBundle()
	configureCommandFlags()
	assignCommandGroups(commands)
	addCommandsToRoot(commands)
	normalizeSubcommandHelpFlags()
}

func configureRootCommand() {
	rootCmd.AddGroup(&cobra.Group{ID: "setup", Title: "Setup Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: "development", Title: "Development Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: "execution", Title: "Execution Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: "analysis", Title: "Analysis Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: "utilities", Title: "Utilities:"})
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output showing detailed information")
	rootCmd.PersistentFlags().BoolVar(&bannerFlag, "banner", false, "Display ASCII logo banner with purple GitHub color theme")
	rootCmd.SetOut(os.Stderr)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.SetVersionTemplate(string(constants.CLIExtensionPrefix) + " version {{.Version}}\n")
	rootCmd.InitDefaultHelpFlag()
	if f := rootCmd.Flags().Lookup("help"); f != nil {
		f.Usage = "Show help for " + string(constants.CLIExtensionPrefix)
	}
	rootCmd.InitDefaultVersionFlag()
	if f := rootCmd.Flags().Lookup("version"); f != nil {
		f.Usage = "Print the current version"
	}
}

func configureRootUsage() {
	rootCmd.SetUsageFunc(printUsageWithGhAwPath)
}

func printUsageWithGhAwPath(cmd *cobra.Command) error {
	out := cmd.OutOrStderr()
	fmt.Fprint(out, "Usage:")
	if cmd.Runnable() {
		fmt.Fprintf(out, "\n  %s", fixUsagePath(cmd.UseLine()))
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(out, "\n  %s [command]", fixUsagePath(cmd.CommandPath()))
	}
	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(out, "\n\nAliases:\n  %s", cmd.NameAndAliases())
	}
	if cmd.HasExample() {
		fmt.Fprintf(out, "\n\nExamples:\n%s", cmd.Example)
	}
	printUsageCommandList(out, cmd)
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(out, "\n\nFlags:\n%s", strings.TrimRight(cmd.LocalFlags().FlagUsages(), " \t\n"))
	}
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(out, "\n\nGlobal Flags:\n%s", strings.TrimRight(cmd.InheritedFlags().FlagUsages(), " \t\n"))
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(out, "\n\nUse \"%s [command] --help\" for more information about a command.\n", fixUsagePath(cmd.CommandPath()))
	} else {
		fmt.Fprintln(out)
	}
	return nil
}

func printUsageCommandList(out io.Writer, cmd *cobra.Command) {
	if !cmd.HasAvailableSubCommands() {
		return
	}
	cmds := cmd.Commands()
	colFmt := fmt.Sprintf("\n  %%-%ds %%s", usageCommandColumnWidth(cmds))
	if len(cmd.Groups()) == 0 {
		fmt.Fprint(out, "\n\nAvailable Commands:")
		printUsageCommandsForGroup(out, cmds, colFmt, "")
		return
	}
	for _, group := range cmd.Groups() {
		fmt.Fprintf(out, "\n\n%s", group.Title)
		printUsageCommandsForGroup(out, cmds, colFmt, group.ID)
	}
	if !cmd.AllChildCommandsHaveGroup() {
		fmt.Fprint(out, "\n\nAdditional Commands:")
		printUsageCommandsForGroup(out, cmds, colFmt, "")
	}
}

func usageCommandColumnWidth(cmds []*cobra.Command) int {
	colWidth := 0
	for _, sub := range cmds {
		if (sub.IsAvailableCommand() || sub.Name() == "help") && len(sub.Name()) > colWidth {
			colWidth = len(sub.Name())
		}
	}
	return colWidth
}

func printUsageCommandsForGroup(out io.Writer, cmds []*cobra.Command, colFmt, groupID string) {
	for _, sub := range cmds {
		if sub.GroupID != groupID || (!sub.IsAvailableCommand() && sub.Name() != "help") {
			continue
		}
		fmt.Fprintf(out, colFmt, sub.Name(), sub.Short)
	}
}

func fixUsagePath(s string) string {
	if s == "gh" {
		return "gh aw"
	}
	if strings.HasPrefix(s, "gh ") && !strings.HasPrefix(s, "gh aw") {
		return "gh aw " + s[3:]
	}
	return s
}

func newCustomHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type ` + string(constants.CLIExtensionPrefix) + ` help [path to command] for full details.

Use "` + string(constants.CLIExtensionPrefix) + ` help all" to show help for all commands.`,
		RunE: runCustomHelpCommand,
	}
}

func runCustomHelpCommand(_ *cobra.Command, args []string) error {
	if len(args) == 1 && args[0] == "all" {
		printAllCommandsHelp()
		return nil
	}
	cmd, _, err := rootCmd.Find(args)
	if cmd == nil || err != nil {
		return fmt.Errorf("unknown help topic [%#q]", args)
	}
	cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
	return cmd.Help()
}

func printAllCommandsHelp() {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Agentic Workflows CLI - Complete Command Reference"))
	fmt.Fprintln(os.Stderr, "")
	for _, subCmd := range rootCmd.Commands() {
		if subCmd.Hidden || subCmd.Name() == "help" {
			continue
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
		fmt.Fprintf(os.Stderr, "\n%s\n\n", console.FormatInfoMessage(fmt.Sprintf("Command: %s %s", string(constants.CLIExtensionPrefix), subCmd.Name())))
		_ = subCmd.Help()
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For more information, visit: https://github.github.com/gh-aw/"))
}

type commandBundle struct {
	addCmd, addWizardCmd, updateCmd, deployCmd, trialCmd, initCmd   *cobra.Command
	statusCmd, listCmd, mcpCmd, logsCmd, auditCmd, viewCmd          *cobra.Command
	healthCmd, outcomesCmd, mcpServerCmd, prCmd, secretsCmd, fixCmd *cobra.Command
	upgradeCmd, completionCmd, hashCmd, projectCmd, doctorCmd       *cobra.Command
	checksCmd, validateCmd, lintCmd, domainsCmd, experimentsCmd     *cobra.Command
	forecastCmd, envCmd                                             *cobra.Command
}

func createCommandBundle() commandBundle {
	initCmd := cli.NewInitCommand()
	cli.RegisterEngineFlagCompletion(initCmd)
	return commandBundle{
		addCmd:         cli.NewAddCommand(validateEngine),
		addWizardCmd:   cli.NewAddWizardCommand(validateEngine),
		updateCmd:      cli.NewUpdateCommand(validateEngine),
		deployCmd:      cli.NewDeployCommand(validateEngine),
		trialCmd:       cli.NewTrialCommand(validateEngine),
		initCmd:        initCmd,
		statusCmd:      cli.NewStatusCommand(),
		listCmd:        cli.NewListCommand(),
		mcpCmd:         cli.NewMCPCommand(),
		logsCmd:        cli.NewLogsCommand(),
		auditCmd:       cli.NewAuditCommand(),
		viewCmd:        cli.NewViewCommand(),
		healthCmd:      cli.NewHealthCommand(),
		outcomesCmd:    cli.NewOutcomesCommand(),
		mcpServerCmd:   cli.NewMCPServerCommand(),
		prCmd:          cli.NewPRCommand(),
		secretsCmd:     cli.NewSecretsCommand(),
		fixCmd:         cli.NewFixCommand(),
		upgradeCmd:     cli.NewUpgradeCommand(validateEngine),
		completionCmd:  cli.NewCompletionCommand(),
		hashCmd:        cli.NewHashCommand(),
		projectCmd:     cli.NewProjectCommand(),
		doctorCmd:      cli.NewDoctorCommand(),
		checksCmd:      cli.NewChecksCommand(),
		validateCmd:    cli.NewValidateCommand(validateEngine),
		lintCmd:        cli.NewLintCommand(),
		domainsCmd:     cli.NewDomainsCommand(),
		experimentsCmd: cli.NewExperimentsCommand(),
		forecastCmd:    cli.NewForecastCommand(),
		envCmd:         cli.NewEnvCommand(),
	}
}

func configureCommandFlags() {
	configureNewCommandFlags()
	configureCompileCommandFlags()
	configureRemoveAndToggleFlags()
	configureRunCommandFlags()
	cli.RegisterEngineFlagCompletion(newCmd)
	cli.RegisterEngineFlagCompletion(runCmd)
	compileCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	removeCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	enableCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	disableCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	runCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	cli.RegisterEngineFlagCompletion(compileCmd)
	cli.RegisterDirFlagCompletion(compileCmd, "dir")
	cli.RegisterDirFlagCompletion(removeCmd, "dir")
}

func configureNewCommandFlags() {
	newCmd.Flags().BoolP("force", "f", false, "Overwrite existing workflow files without confirmation")
	newCmd.Flags().BoolP("interactive", "i", false, "Launch interactive workflow creation wizard")
	newCmd.Flags().StringP("engine", "e", "", cli.EngineFlagOverrideUsage)
}

func configureCompileCommandFlags() {
	configureCompileCoreFlags()
	configureCompileScannerFlags()
	configureCompileAdvancedFlags()
	if err := compileCmd.Flags().MarkHidden("prior-manifest-file"); err != nil {
		_ = err
	}
	compileCmd.MarkFlagsMutuallyExclusive("dir", "workflows-dir")
	compileCmd.MarkFlagsMutuallyExclusive("gh-aw-ref", "action-tag")
	compileCmd.MarkFlagsMutuallyExclusive("gh-aw-ref", "action-mode")
}

func configureCompileCoreFlags() {
	compileCmd.Flags().StringP("engine", "e", "", cli.EngineFlagOverrideUsage)
	compileCmd.Flags().String("action-mode", "", "How gh-aw action scripts are referenced in compiled workflows: 'dev' uses local paths (for developing gh-aw itself), 'release' emits SHA-pinned remote refs from github/gh-aw, 'action' uses the github/gh-aw-actions repository. Auto-detected from the binary build type if not specified")
	compileCmd.Flags().String("action-tag", "", "Pin compiled workflows to a specific version of gh-aw actions. Accepts a full commit SHA or a version tag (e.g. v1, v1.2.3). Sets --action-mode to 'release' unless --action-mode action is also specified. Cannot be combined with --gh-aw-ref; use --gh-aw-ref when you want to resolve a branch or tag name to its current SHA")
	compileCmd.Flags().String("actions-repo", "", "Override the external actions repository used in action mode (default: github/gh-aw-actions)")
	compileCmd.Flags().String("gh-aw-ref", "", "Pin compiled workflows to a specific branch, tag, or commit SHA of github/gh-aw (e.g. main, my-feature, abc123). Branch and tag names are resolved to their full commit SHA at compile time so the baked-in ref is immutable. Equivalent to --action-mode release --action-tag <resolved-sha>. Cannot be combined with --action-tag or --action-mode. Use this to E2E-test workflows against a specific gh-aw revision")
	compileCmd.Flags().Bool("validate", false, "Enable GitHub Actions workflow schema validation, container image validation, and action SHA validation")
	compileCmd.Flags().BoolP("watch", "w", false, "Watch for changes to workflow files and recompile automatically")
	compileCmd.Flags().StringP("dir", "d", "", "Workflow directory (default: $GH_AW_WORKFLOWS_DIR or .github/workflows)")
	compileCmd.Flags().String("workflows-dir", "", "Deprecated: use --dir instead")
	_ = compileCmd.Flags().MarkDeprecated("workflows-dir", "use --dir instead")
	compileCmd.Flags().Bool("no-emit", false, "Validate workflow without generating lock files")
	compileCmd.Flags().Bool("purge", false, "Delete .lock.yml files that were not regenerated during compilation (only when no specific files are provided)")
	compileCmd.Flags().Bool("strict", false, "Override frontmatter to enforce strict mode validation for all workflows (enforces action pinning, network config, safe-outputs, disallows write permissions and deprecated fields). Note: Workflows default to strict mode unless frontmatter sets strict: false")
	compileCmd.Flags().Bool("trial", false, "Enable trial mode compilation (modifies workflows for trial execution)")
	compileCmd.Flags().StringP("logical-repo", "l", "", "Repository to simulate workflow execution against (for trial mode)")
	compileCmd.Flags().Bool("use-samples", false, "Hidden: replace the agentic 'Execute coding agent' step with a deterministic driver that replays the workflow's safe-outputs `samples` frontmatter entries through the safe-outputs MCP server. Used to make end-to-end tests deterministic.")
	_ = compileCmd.Flags().MarkHidden("use-samples")
	compileCmd.Flags().Bool("dependabot", false, "Generate dependency manifests (package.json, requirements.txt, go.mod) and Dependabot config when dependencies are detected")
	compileCmd.Flags().BoolP("force", "f", false, "Force overwrite of existing dependency files (only applies when --dependabot is set; e.g., dependabot.yml)")
	compileCmd.Flags().Bool("refresh-stop-time", false, "Force regeneration of stop-after times instead of preserving existing values from lock files")
	compileCmd.Flags().Bool("force-refresh-action-pins", false, "Force refresh of action pins by clearing the cache and resolving all action SHAs from GitHub API")
	compileCmd.Flags().Bool("allow-action-refs", false, "Allow unresolved action refs and emit warnings instead of failing validation")
}

func configureCompileScannerFlags() {
	compileCmd.Flags().Bool("zizmor", false, "Run zizmor security scanner on generated .lock.yml files")
	compileCmd.Flags().Bool("poutine", false, "Run poutine security scanner on generated .lock.yml files")
	compileCmd.Flags().Bool("actionlint", false, "Run actionlint linter on generated .lock.yml files")
	compileCmd.Flags().Bool("runner-guard", false, "Run runner-guard taint analysis scanner on generated .lock.yml files (uses Docker image "+cli.RunnerGuardImage+")")
	compileCmd.Flags().Bool("syft", false, "Run syft SBOM scanner on container images referenced in compiled .lock.yml files (uses Docker image "+cli.SyftImage+")")
	compileCmd.Flags().Bool("grype", false, "Run grype vulnerability scanner on container images referenced in compiled .lock.yml files (uses Docker image "+cli.GrypeImage+")")
	compileCmd.Flags().Bool("grant", false, "Run grant license scanner on container images referenced in compiled .lock.yml files (uses Docker image "+cli.GrantImage+")")
	compileCmd.Flags().Bool("yamllint", false, "Run yamllint YAML linter on generated .lock.yml files (uses Docker image "+cli.YamllintImage+")")
}

func configureCompileAdvancedFlags() {
	compileCmd.Flags().Bool("fix", false, "Apply automatic codemod fixes to workflows before compiling")
	compileCmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	compileCmd.Flags().Bool("show-all", false, "Display all compilation errors instead of only the highest-priority subset (default: top 5)")
	compileCmd.Flags().Bool("stats", false, "Display statistics table sorted by workflow file size (shows jobs, steps, scripts, and shells)")
	compileCmd.Flags().Bool("fail-fast", false, "Stop at the first validation error instead of collecting all errors")
	compileCmd.Flags().Bool("no-check-update", false, "Skip checking for gh-aw updates")
	compileCmd.Flags().String("schedule-seed", "", "Override the repository slug (owner/repo) used as seed for fuzzy schedule scattering (e.g., \"github/gh-aw\"). Bypasses git remote detection entirely. Use this when your git remote is not named \"origin\" and you have multiple remotes configured")
	compileCmd.Flags().Bool("staged", false, "Force all safe-outputs into staged mode")
	compileCmd.Flags().Bool("approve", false, "Approve all safe update changes. When strict mode is active (the default), the compiler emits warnings for new restricted secrets or unapproved action additions/removals not present in the existing gh-aw-manifest. Use this flag to approve and skip safe update enforcement")
	compileCmd.Flags().Bool("validate-images", false, "Require Docker to be available for container image validation. Without this flag, container image validation is silently skipped when Docker is not installed or the daemon is not running")
	compileCmd.Flags().Bool("no-models-dev-lookup", false, "Disable compile-time models.dev pricing lookup for models missing from the embedded catalog")
	compileCmd.Flags().String("prior-manifest-file", "", "Path to a JSON file containing pre-cached gh-aw-manifests (map[lockFile]*GHAWManifest); used by the MCP server to supply a tamper-proof manifest baseline captured at startup")
	compileCmd.Flags().Bool("ghes", false, "Enable GitHub Enterprise Server (GHES) compatibility mode. Artifact actions continue using latest non-v3 pins (v3 is deprecated). Overrides the aw.json ghes field")
}

func configureRemoveAndToggleFlags() {
	removeCmd.Flags().Bool("no-remove-orphans", false, "Skip removal of orphaned include files that are no longer referenced by any workflow")
	removeCmd.Flags().Bool("keep-orphans", false, "Skip removal of orphaned include files that are no longer referenced by any workflow")
	_ = removeCmd.Flags().MarkDeprecated("keep-orphans", "use --no-remove-orphans instead")
	removeCmd.Flags().StringP("dir", "d", "", "Workflow directory (default: $GH_AW_WORKFLOWS_DIR or .github/workflows)")
	enableCmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
	disableCmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
}

func configureRunCommandFlags() {
	runCmd.Flags().Int("repeat", 0, "Number of additional times to run after the initial execution (e.g., --repeat 3 runs 4 times total)")
	runCmd.Flags().Bool("enable-if-needed", false, "Enable the workflow before running if needed, and restore state afterward")
	runCmd.Flags().StringP("engine", "e", "", cli.EngineFlagOverrideUsage)
	runCmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
	runCmd.Flags().String("ref", "", "Branch or tag name to run the workflow on (default: current branch)")
	runCmd.Flags().Bool("auto-merge-prs", false, "Auto-merge any pull requests created during the workflow execution")
	runCmd.Flags().StringArrayP("raw-field", "F", []string{}, "Pass a workflow dispatch input in key=value format (can be specified multiple times)")
	_ = runCmd.Flags().MarkShorthandDeprecated("raw-field", "use the long form --raw-field instead")
	runCmd.Flags().Bool("push", false, "Commit and push workflow files (including transitive imports) before running. Refuses to proceed when unrelated files are already staged.")
	runCmd.Flags().Bool("dry-run", false, "Preview workflow execution without triggering runs on GitHub Actions")
	runCmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	runCmd.Flags().Bool("approve", false, "Approve all safe update changes. When strict mode is active (the default), the compiler emits warnings for new restricted secrets or unapproved action additions/removals not present in the existing gh-aw-manifest. Use this flag to approve and skip safe update enforcement")
}

func assignCommandGroups(commands commandBundle) {
	commands.initCmd.GroupID, newCmd.GroupID, commands.addCmd.GroupID, commands.addWizardCmd.GroupID = "setup", "setup", "setup", "setup"
	removeCmd.GroupID, commands.updateCmd.GroupID, commands.deployCmd.GroupID, commands.upgradeCmd.GroupID = "setup", "setup", "setup", "setup"
	commands.secretsCmd.GroupID, commands.envCmd.GroupID, commands.doctorCmd.GroupID = "setup", "setup", "setup"
	compileCmd.GroupID, commands.validateCmd.GroupID, commands.lintCmd.GroupID = "development", "development", "development"
	commands.mcpCmd.GroupID, commands.fixCmd.GroupID, commands.domainsCmd.GroupID = "development", "development", "development"
	runCmd.GroupID, enableCmd.GroupID, disableCmd.GroupID, commands.trialCmd.GroupID = "execution", "execution", "execution", "execution"
	commands.logsCmd.GroupID, commands.auditCmd.GroupID, commands.viewCmd.GroupID = "analysis", "analysis", "analysis"
	commands.healthCmd.GroupID, commands.outcomesCmd.GroupID, commands.checksCmd.GroupID = "analysis", "analysis", "analysis"
	commands.statusCmd.GroupID, commands.listCmd.GroupID, commands.experimentsCmd.GroupID, commands.forecastCmd.GroupID = "analysis", "analysis", "analysis", "analysis"
	commands.mcpServerCmd.GroupID, commands.prCmd.GroupID, commands.completionCmd.GroupID = "utilities", "utilities", "utilities"
	commands.hashCmd.GroupID, commands.projectCmd.GroupID = "utilities", "utilities"
}

func addCommandsToRoot(commands commandBundle) {
	rootCmd.AddCommand(commands.addCmd, commands.addWizardCmd, commands.updateCmd, commands.deployCmd, commands.upgradeCmd, commands.trialCmd, newCmd, commands.initCmd)
	rootCmd.AddCommand(runCmd, removeCmd, commands.statusCmd, commands.listCmd, enableCmd, disableCmd, commands.logsCmd, commands.auditCmd)
	rootCmd.AddCommand(commands.viewCmd, commands.healthCmd, commands.outcomesCmd, commands.checksCmd, commands.mcpCmd, commands.mcpServerCmd, commands.prCmd, versionCmd)
	rootCmd.AddCommand(commands.secretsCmd, commands.fixCmd, compileCmd, commands.validateCmd, commands.lintCmd, commands.completionCmd, commands.hashCmd, commands.projectCmd)
	rootCmd.AddCommand(commands.doctorCmd, commands.domainsCmd, commands.experimentsCmd, commands.forecastCmd, commands.envCmd)
}

func normalizeSubcommandHelpFlags() {
	var fixSubCmdHelpFlags func(cmd *cobra.Command)
	fixSubCmdHelpFlags = func(cmd *cobra.Command) {
		cmd.InitDefaultHelpFlag()
		if f := cmd.Flags().Lookup("help"); f != nil {
			cmdPath := cmd.CommandPath()
			if strings.HasPrefix(cmdPath, "gh ") && !strings.HasPrefix(cmdPath, "gh aw") {
				cmdPath = "gh aw " + cmdPath[3:]
			}
			f.Usage = "Show help for " + cmdPath
		}
		for _, sub := range cmd.Commands() {
			fixSubCmdHelpFlags(sub)
		}
	}
	for _, sub := range rootCmd.Commands() {
		fixSubCmdHelpFlags(sub)
	}
}

func main() {
	// Set version information in the CLI package
	cli.SetVersionInfo(version)

	// Set version information in the workflow package for generated file headers
	workflow.SetVersion(version)

	// Set release flag in the workflow package
	workflow.SetIsRelease(isRelease == "true")

	// Set up a context that is cancelled when Ctrl-C (SIGINT) or SIGTERM is received.
	// This ensures all commands and subprocesses are properly interrupted on Ctrl-C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// ExitCodeError signals an intentional exit with a specific code (e.g.
		// after relaunching the upgraded binary). Honour it without printing an
		// error message.
		var exitCodeErr *cli.ExitCodeError
		if errors.As(err, &exitCodeErr) {
			os.Exit(exitCodeErr.Code)
		}

		errMsg := err.Error()
		// Check if error is already formatted to avoid double formatting:
		// - Contains suggestions (FormatErrorWithSuggestions)
		// - Starts with ✗ (FormatErrorMessage)
		// - Contains file:line:column: pattern (console.FormatError)
		isAlreadyFormatted := strings.Contains(errMsg, "Suggestions:") ||
			strings.HasPrefix(errMsg, "✗") ||
			strings.Contains(errMsg, ":") && (strings.Contains(errMsg, "error:") || strings.Contains(errMsg, "warning:"))

		if isAlreadyFormatted {
			fmt.Fprintln(os.Stderr, errMsg)
		} else {
			fmt.Fprintln(os.Stderr, console.FormatErrorChain(err))
		}
		os.Exit(1)
	}
}
