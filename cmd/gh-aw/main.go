package main

import (
	"context"
	"errors"
	"fmt"
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
	Short:   "GitHub Agentic Workflows CLI from GitHub Next",
	Version: version,
	Long: `GitHub Agentic Workflows from GitHub Next

Common Tasks:
  gh aw init                  # Set up a new repository
  gh aw add-wizard            # Add workflows with interactive guided setup
  gh aw new my-workflow       # Create your first workflow
  gh aw compile               # Compile all workflows
  gh aw run my-workflow       # Execute a workflow
  gh aw logs my-workflow      # View execution logs
  gh aw audit <run-id-or-url> # Debug a failed run

For detailed help on any command, use:
  gh aw [command] --help`,
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
- Tools configuration (github, claude, MCPs)
- All frontmatter options with explanations

` + cli.WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` new                      # Interactive mode
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
by any workflow. Use --keep-orphans to skip this cleanup.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` remove my-workflow              # Remove specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` remove test-                    # Remove all workflows containing 'test-' in name
  ` + string(constants.CLIExtensionPrefix) + ` remove old- --keep-orphans      # Remove workflows but keep orphaned includes
  ` + string(constants.CLIExtensionPrefix) + ` remove my-workflow --dir .github/workflows/shared  # Remove from custom directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pattern string
		if len(args) > 0 {
			pattern = args[0]
		}
		keepOrphans, _ := cmd.Flags().GetBool("keep-orphans")
		workflowDir, _ := cmd.Flags().GetString("dir")
		return cli.RemoveWorkflows(pattern, keepOrphans, workflowDir)
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable [workflow]...",
	Short: "Enable agentic workflows",
	Long: `Enable one or more workflows by ID, or all workflows if no IDs are provided.

` + cli.WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` enable                   # Enable all workflows
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

Any in-progress runs will be cancelled before disabling.

` + cli.WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` disable                   # Disable all workflows
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
  - Creates .github/dependabot.yml with all detected ecosystems
  - Use --force to overwrite existing dependabot.yml
  - Cannot be used with specific workflow files or custom --dir
  - Only processes workflows in the default .github/workflows directory

Action mode controls how gh-aw action scripts are referenced in compiled workflows.
Three flags govern this. --gh-aw-ref is mutually exclusive with the other two;
--action-tag and --action-mode may be combined (e.g. --action-mode action --action-tag v1.2.3):

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
    The value is used as-is; branch names are not resolved. Use --gh-aw-ref to
    pin to a branch by resolving it to its current commit SHA first.

  --gh-aw-ref <branch-tag-or-sha>
    Resolve a branch name, tag, or SHA from github/gh-aw to its full commit SHA
    at compile time and pin the compiled workflow to that immutable SHA.
    Equivalent to --action-mode release --action-tag <resolved-sha>.
    Branch and tag names are resolved via the GitHub API.
    Cannot be combined with --action-tag or --action-mode.
    Use this when E2E-testing compiled workflows against a specific gh-aw revision.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` compile                    # Compile all Markdown files
  ` + string(constants.CLIExtensionPrefix) + ` compile ci-doctor          # Compile a specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` compile ci-doctor daily-plan  # Compile multiple workflows
  ` + string(constants.CLIExtensionPrefix) + ` compile workflow.md        # Compile by file path
  ` + string(constants.CLIExtensionPrefix) + ` compile .github/workflows  # Compile all workflows in a directory
  ` + string(constants.CLIExtensionPrefix) + ` compile --dir custom/workflows  # Compile from custom directory
  ` + string(constants.CLIExtensionPrefix) + ` compile --watch ci-doctor     # Watch and auto-compile
  ` + string(constants.CLIExtensionPrefix) + ` compile --trial --logical-repo owner/repo  # Compile for trial mode
  ` + string(constants.CLIExtensionPrefix) + ` compile --dependabot        # Generate Dependabot manifests
  ` + string(constants.CLIExtensionPrefix) + ` compile --dependabot --force  # Force overwrite existing dependabot.yml
  ` + string(constants.CLIExtensionPrefix) + ` compile --gh-aw-ref main       # Pin workflows to current HEAD of github/gh-aw main
  ` + string(constants.CLIExtensionPrefix) + ` compile --action-tag v1.2.3    # Pin workflows to a specific release tag`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engineOverride, _ := cmd.Flags().GetString("engine")
		actionMode, _ := cmd.Flags().GetString("action-mode")
		actionTag, _ := cmd.Flags().GetString("action-tag")
		actionsRepo, _ := cmd.Flags().GetString("actions-repo")
		ghAwRef, _ := cmd.Flags().GetString("gh-aw-ref")
		if ghAwRef != "" {
			// --gh-aw-ref is a convenience alias: emit refs like
			// `github/gh-aw/actions/setup@<sha>` so external e2e harnesses can
			// test the compiled workflows against a specific gh-aw revision.
			// Resolve branch/tag names to their commit SHA so the baked-in ref
			// is immutable and not vulnerable to branch/tag mutation.
			resolvedRef, resolveErr := workflow.ResolveGhAwRef(cmd.Context(), ghAwRef)
			if resolveErr != nil {
				return fmt.Errorf("--gh-aw-ref: %w", resolveErr)
			}
			actionMode = string(workflow.ActionModeRelease)
			actionTag = resolvedRef
		}
		validate, _ := cmd.Flags().GetBool("validate")
		watch, _ := cmd.Flags().GetBool("watch")
		dir, _ := cmd.Flags().GetString("dir")
		workflowsDir, _ := cmd.Flags().GetString("workflows-dir")
		noEmit, _ := cmd.Flags().GetBool("no-emit")
		purge, _ := cmd.Flags().GetBool("purge")
		strict, _ := cmd.Flags().GetBool("strict")
		trial, _ := cmd.Flags().GetBool("trial")
		logicalRepo, _ := cmd.Flags().GetString("logical-repo")
		dependabot, _ := cmd.Flags().GetBool("dependabot")
		forceOverwrite, _ := cmd.Flags().GetBool("force")
		refreshStopTime, _ := cmd.Flags().GetBool("refresh-stop-time")
		forceRefreshActionPins, _ := cmd.Flags().GetBool("force-refresh-action-pins")
		allowActionRefs, _ := cmd.Flags().GetBool("allow-action-refs")
		zizmor, _ := cmd.Flags().GetBool("zizmor")
		poutine, _ := cmd.Flags().GetBool("poutine")
		actionlint, _ := cmd.Flags().GetBool("actionlint")
		runnerGuard, _ := cmd.Flags().GetBool("runner-guard")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		showAllErrors, _ := cmd.Flags().GetBool("show-all")
		fix, _ := cmd.Flags().GetBool("fix")
		stats, _ := cmd.Flags().GetBool("stats")
		failFast, _ := cmd.Flags().GetBool("fail-fast")
		noCheckUpdate, _ := cmd.Flags().GetBool("no-check-update")
		scheduleSeed, _ := cmd.Flags().GetString("schedule-seed")
		staged, _ := cmd.Flags().GetBool("staged")
		approve, _ := cmd.Flags().GetBool("approve")
		validateImages, _ := cmd.Flags().GetBool("validate-images")
		priorManifestFile, _ := cmd.Flags().GetString("prior-manifest-file")
		ghes, _ := cmd.Flags().GetBool("ghes")
		verbose, _ := cmd.Flags().GetBool("verbose")
		useSamples, _ := cmd.Flags().GetBool("use-samples")
		if err := validateEngine(engineOverride); err != nil {
			return err
		}

		finishCompileUpdateCheck := cli.StartCompileUpdateCheck(cmd.Context(), noCheckUpdate, verbose)
		defer finishCompileUpdateCheck()

		// If --fix is specified, run fix --write first
		if fix {
			fixConfig := cli.FixConfig{
				WorkflowIDs: args,
				Write:       true,
				Verbose:     verbose,
				WorkflowDir: dir,
			}
			if err := cli.RunFix(fixConfig); err != nil {
				return err
			}
		}

		// Handle --workflows-dir deprecation (mutual exclusion is enforced by Cobra)
		workflowDir := dir
		if workflowsDir != "" {
			workflowDir = workflowsDir
		}
		config := cli.CompileConfig{
			MarkdownFiles:          args,
			Verbose:                verbose,
			EngineOverride:         engineOverride,
			ActionMode:             actionMode,
			ActionTag:              actionTag,
			ActionsRepo:            actionsRepo,
			Validate:               validate,
			Watch:                  watch,
			WorkflowDir:            workflowDir,
			SkipInstructions:       false, // Deprecated field, kept for backward compatibility
			NoEmit:                 noEmit,
			Purge:                  purge,
			TrialMode:              trial,
			TrialLogicalRepoSlug:   logicalRepo,
			Strict:                 strict,
			Dependabot:             dependabot,
			ForceOverwrite:         forceOverwrite,
			RefreshStopTime:        refreshStopTime,
			ForceRefreshActionPins: forceRefreshActionPins,
			AllowActionRefs:        allowActionRefs,
			Zizmor:                 zizmor,
			Poutine:                poutine,
			Actionlint:             actionlint,
			RunnerGuard:            runnerGuard,
			JSONOutput:             jsonOutput,
			ShowAllErrors:          showAllErrors,
			Stats:                  stats,
			FailFast:               failFast,
			ScheduleSeed:           scheduleSeed,
			Staged:                 staged,
			Approve:                approve,
			ValidateImages:         validateImages,
			PriorManifestFile:      priorManifestFile,
			GHESCompat:             ghes,
			UseSamples:             useSamples,
		}
		if _, err := cli.CompileWorkflows(cmd.Context(), config); err != nil {
			// Return error as-is without additional formatting
			// Errors from CompileWorkflows are already formatted with console.FormatError
			// which provides IDE-parseable location information (file:line:column)
			return err
		}
		return nil
	},
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

` + cli.WorkflowIDExplanation + `

Examples:
  gh aw run                          # Interactive mode
  gh aw run daily-perf-improver
  gh aw run daily-perf-improver.md   # Alternative format
  gh aw run daily-perf-improver --ref main  # Run on specific branch
  gh aw run daily-perf-improver --repeat 3  # Run 4 times total (1 initial + 3 repeats)
  gh aw run daily-perf-improver --enable-if-needed  # Enable if disabled, run, then restore state
  gh aw run daily-perf-improver --auto-merge-prs  # Auto-merge any PRs created during execution
  gh aw run daily-perf-improver -F name=value -F env=prod  # Pass workflow inputs
  gh aw run daily-perf-improver --push  # Commit and push workflow files before running
  gh aw run daily-perf-improver --dry-run  # Validate without actually running
  gh aw run daily-perf-improver --json  # Output results in JSON format`,
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

			return cli.RunWorkflowInteractively(cmd.Context(), verboseFlag, repoOverride, refOverride, autoMergePRs, push, engineOverride, dryRun)
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
	Use:   "version",
	Short: "Print the current version",
	Long: `Show the installed version of the gh aw extension.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` version   # Print the current version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(os.Stderr, "%s version %s\n", string(constants.CLIExtensionPrefix), version)
		return nil
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
	rootCmd.AddGroup(&cobra.Group{
		ID:    "utilities",
		Title: "Utilities:",
	})

	// Add global verbose flag to root command
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output showing detailed information")

	// Add global banner flag to root command
	rootCmd.PersistentFlags().BoolVar(&bannerFlag, "banner", false, "Display ASCII logo banner with purple GitHub color theme")

	// Set output to stderr for consistency with CLI logging guidelines
	rootCmd.SetOut(os.Stderr)

	// Silence usage output on errors - prevents cluttering terminal output with
	// full usage text when application errors occur (e.g., compilation errors,
	// network timeouts). Users can still run --help for usage information.
	rootCmd.SilenceUsage = true

	// Silence errors - since we're using RunE and returning errors, Cobra will
	// print errors automatically. We handle error formatting ourselves in main().
	rootCmd.SilenceErrors = true

	// Set version template to match the version subcommand format
	rootCmd.SetVersionTemplate(string(constants.CLIExtensionPrefix) + " version {{.Version}}\n")

	// Cobra generates flag descriptions using c.Name() which returns the first
	// word of Use ("gh" from "gh aw"), producing "help for gh" and "version for
	// gh". Explicitly initialize and override these flags so they display "gh aw".
	rootCmd.InitDefaultHelpFlag()
	if f := rootCmd.Flags().Lookup("help"); f != nil {
		f.Usage = "Show help for " + string(constants.CLIExtensionPrefix)
	}
	rootCmd.InitDefaultVersionFlag()
	if f := rootCmd.Flags().Lookup("version"); f != nil {
		f.Usage = "Print the current version"
	}

	// Fix usage lines so subcommands show "gh aw <cmd>" instead of "gh <cmd>".
	// Cobra derives the root name from the first word of Use ("gh" from "gh aw"),
	// so CommandPath() for subcommands omits "aw". We use SetUsageFunc to
	// post-process the default output, replacing "gh " with "gh aw " in the
	// two lines that reference the command path.
	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fixPath := func(s string) string {
			if s == "gh" {
				return "gh aw"
			}
			if strings.HasPrefix(s, "gh ") && !strings.HasPrefix(s, "gh aw") {
				return "gh aw " + s[3:]
			}
			return s
		}
		out := cmd.OutOrStderr()
		fmt.Fprint(out, "Usage:")
		if cmd.Runnable() {
			fmt.Fprintf(out, "\n  %s", fixPath(cmd.UseLine()))
		}
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintf(out, "\n  %s [command]", fixPath(cmd.CommandPath()))
		}
		if len(cmd.Aliases) > 0 {
			fmt.Fprintf(out, "\n\nAliases:\n  %s", cmd.NameAndAliases())
		}
		if cmd.HasExample() {
			fmt.Fprintf(out, "\n\nExamples:\n%s", cmd.Example)
		}
		if cmd.HasAvailableSubCommands() {
			cmds := cmd.Commands()
			// Compute column width dynamically so long command names (e.g. hash-frontmatter)
			// are aligned properly instead of overflowing a hard-coded width.
			colWidth := 0
			for _, sub := range cmds {
				if (sub.IsAvailableCommand() || sub.Name() == "help") && len(sub.Name()) > colWidth {
					colWidth = len(sub.Name())
				}
			}
			colFmt := fmt.Sprintf("\n  %%-%ds %%s", colWidth)
			if len(cmd.Groups()) == 0 {
				fmt.Fprint(out, "\n\nAvailable Commands:")
				for _, sub := range cmds {
					if sub.IsAvailableCommand() || sub.Name() == "help" {
						fmt.Fprintf(out, colFmt, sub.Name(), sub.Short)
					}
				}
			} else {
				for _, group := range cmd.Groups() {
					fmt.Fprintf(out, "\n\n%s", group.Title)
					for _, sub := range cmds {
						if sub.GroupID == group.ID && (sub.IsAvailableCommand() || sub.Name() == "help") {
							fmt.Fprintf(out, colFmt, sub.Name(), sub.Short)
						}
					}
				}
				if !cmd.AllChildCommandsHaveGroup() {
					fmt.Fprint(out, "\n\nAdditional Commands:")
					for _, sub := range cmds {
						if sub.GroupID == "" && (sub.IsAvailableCommand() || sub.Name() == "help") {
							fmt.Fprintf(out, colFmt, sub.Name(), sub.Short)
						}
					}
				}
			}
		}
		if cmd.HasAvailableLocalFlags() {
			fmt.Fprintf(out, "\n\nFlags:\n%s", strings.TrimRight(cmd.LocalFlags().FlagUsages(), " \t\n"))
		}
		if cmd.HasAvailableInheritedFlags() {
			fmt.Fprintf(out, "\n\nGlobal Flags:\n%s", strings.TrimRight(cmd.InheritedFlags().FlagUsages(), " \t\n"))
		}
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintf(out, "\n\nUse \"%s [command] --help\" for more information about a command.\n", fixPath(cmd.CommandPath()))
		} else {
			fmt.Fprintln(out)
		}
		return nil
	})

	// Create custom help command that supports "all" subcommand
	customHelpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type ` + string(constants.CLIExtensionPrefix) + ` help [path to command] for full details.

Use "` + string(constants.CLIExtensionPrefix) + ` help all" to show help for all commands.`,
		RunE: func(c *cobra.Command, args []string) error {
			// Check if the argument is "all"
			if len(args) == 1 && args[0] == "all" {
				// Print header
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Agentic Workflows CLI - Complete Command Reference"))
				fmt.Fprintln(os.Stderr, "")

				// Iterate through all commands and print their help
				for _, subCmd := range rootCmd.Commands() {
					// Skip hidden commands and help itself
					if subCmd.Hidden || subCmd.Name() == "help" {
						continue
					}

					// Print command separator
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
					fmt.Fprintf(os.Stderr, "\n%s\n\n", console.FormatInfoMessage(fmt.Sprintf("Command: %s %s", string(constants.CLIExtensionPrefix), subCmd.Name())))

					// Print the command's help
					_ = subCmd.Help()
					fmt.Fprintln(os.Stderr, "")
				}

				// Print footer
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════════════"))
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("For more information, visit: https://github.github.com/gh-aw/"))
				return nil
			}

			// Otherwise, use the default help behavior
			cmd, _, e := rootCmd.Find(args)
			if cmd == nil || e != nil {
				return fmt.Errorf("unknown help topic [%#q]", args)
			} else {
				cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
				return cmd.Help()
			}
		},
	}

	// Replace the default help command
	rootCmd.SetHelpCommand(customHelpCmd)

	// Create and setup add command
	addCmd := cli.NewAddCommand(validateEngine)

	// Create and setup add-wizard command
	addWizardCmd := cli.NewAddWizardCommand(validateEngine)

	// Create and setup update command
	updateCmd := cli.NewUpdateCommand(validateEngine)

	// Create and setup deploy command
	deployCmd := cli.NewDeployCommand(validateEngine)

	// Create and setup trial command
	trialCmd := cli.NewTrialCommand(validateEngine)

	// Create and setup init command
	initCmd := cli.NewInitCommand()
	cli.RegisterEngineFlagCompletion(initCmd)

	// Add flags to new command
	newCmd.Flags().BoolP("force", "f", false, "Overwrite existing files without confirmation")
	newCmd.Flags().BoolP("interactive", "i", false, "Launch interactive workflow creation wizard")
	newCmd.Flags().StringP("engine", "e", "", "Override AI engine (copilot, claude, codex, gemini, crush)")
	cli.RegisterEngineFlagCompletion(newCmd)

	// Add AI flag to compile and add commands
	compileCmd.Flags().StringP("engine", "e", "", "Override AI engine (copilot, claude, codex, gemini, crush)")
	compileCmd.Flags().String("action-mode", "", "How gh-aw action scripts are referenced in compiled workflows: 'dev' uses local paths (for developing gh-aw itself), 'release' emits SHA-pinned remote refs from github/gh-aw, 'action' uses the github/gh-aw-actions repository. Auto-detected from the binary build type if not specified")
	compileCmd.Flags().String("action-tag", "", "Pin compiled workflows to a specific version of gh-aw actions. Accepts a full commit SHA or a version tag (e.g. v1, v1.2.3). Sets --action-mode to 'release' unless --action-mode action is also specified. Cannot be combined with --gh-aw-ref; use --gh-aw-ref when you want to resolve a branch or tag name to its current SHA")
	compileCmd.Flags().String("actions-repo", "", "Override the external actions repository used in action mode (default: github/gh-aw-actions)")
	compileCmd.Flags().String("gh-aw-ref", "", "Pin compiled workflows to a specific branch, tag, or commit SHA of github/gh-aw (e.g. main, my-feature, abc123). Branch and tag names are resolved to their full commit SHA at compile time so the baked-in ref is immutable. Equivalent to --action-mode release --action-tag <resolved-sha>. Cannot be combined with --action-tag or --action-mode. Use this to E2E-test workflows against a specific gh-aw revision")
	compileCmd.Flags().Bool("validate", false, "Enable GitHub Actions workflow schema validation, container image validation, and action SHA validation")
	compileCmd.Flags().BoolP("watch", "w", false, "Watch for changes to workflow files and recompile automatically")
	compileCmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	compileCmd.Flags().String("workflows-dir", "", "Deprecated: use --dir instead")
	_ = compileCmd.Flags().MarkDeprecated("workflows-dir", "use --dir instead")
	compileCmd.Flags().Bool("no-emit", false, "Validate workflow without generating lock files")
	compileCmd.Flags().Bool("purge", false, "Delete .lock.yml files that were not regenerated during compilation (only when no specific files are specified)")
	compileCmd.Flags().Bool("strict", false, "Override frontmatter to enforce strict mode validation for all workflows (enforces action pinning, network config, safe-outputs, refuses write permissions and deprecated fields). Note: Workflows default to strict mode unless frontmatter sets strict: false")
	compileCmd.Flags().Bool("trial", false, "Enable trial mode compilation (modifies workflows for trial execution)")
	compileCmd.Flags().String("logical-repo", "", "Repository to simulate workflow execution against (for trial mode)")
	compileCmd.Flags().Bool("use-samples", false, "Hidden: replace the agentic 'Execute coding agent' step with a deterministic driver that replays the workflow's safe-outputs `samples` frontmatter entries through the safe-outputs MCP server. Used to make end-to-end tests deterministic.")
	_ = compileCmd.Flags().MarkHidden("use-samples")
	compileCmd.Flags().Bool("dependabot", false, "Generate dependency manifests (package.json, requirements.txt, go.mod) and Dependabot config when dependencies are detected")
	compileCmd.Flags().Bool("force", false, "Force overwrite of existing dependency files (e.g., dependabot.yml)")
	compileCmd.Flags().Bool("refresh-stop-time", false, "Force regeneration of stop-after times instead of preserving existing values from lock files")
	compileCmd.Flags().Bool("force-refresh-action-pins", false, "Force refresh of action pins by clearing the cache and resolving all action SHAs from GitHub API")
	compileCmd.Flags().Bool("allow-action-refs", false, "Allow unresolved action refs and emit warnings instead of failing compilation")
	compileCmd.Flags().Bool("zizmor", false, "Run zizmor security scanner on generated .lock.yml files")
	compileCmd.Flags().Bool("poutine", false, "Run poutine security scanner on generated .lock.yml files")
	compileCmd.Flags().Bool("actionlint", false, "Run actionlint linter on generated .lock.yml files")
	compileCmd.Flags().Bool("runner-guard", false, "Run runner-guard taint analysis scanner on generated .lock.yml files (uses Docker image "+cli.RunnerGuardImage+")")
	compileCmd.Flags().Bool("fix", false, "Apply automatic codemod fixes to workflows before compiling")
	compileCmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	compileCmd.Flags().Bool("show-all", false, "Display all prioritized compilation errors instead of the default top five")
	compileCmd.Flags().Bool("stats", false, "Display statistics table sorted by workflow file size (shows jobs, steps, scripts, and shells)")
	compileCmd.Flags().Bool("fail-fast", false, "Stop at the first validation error instead of collecting all errors")
	compileCmd.Flags().Bool("no-check-update", false, "Skip checking for gh-aw updates")
	compileCmd.Flags().String("schedule-seed", "", "Override the repository slug (owner/repo) used as seed for fuzzy schedule scattering (e.g., \"github/gh-aw\"). Bypasses git remote detection entirely. Use this when your git remote is not named \"origin\" and you have multiple remotes configured")
	compileCmd.Flags().Bool("staged", false, "Force all safe-outputs into staged mode")
	compileCmd.Flags().Bool("approve", false, "Approve all safe update changes. When strict mode is active (the default), the compiler emits warnings for new restricted secrets or unapproved action additions/removals not present in the existing gh-aw-manifest. Use this flag to approve and skip safe update enforcement")
	compileCmd.Flags().Bool("validate-images", false, "Require Docker to be available for container image validation. Without this flag, container image validation is silently skipped when Docker is not installed or the daemon is not running")
	compileCmd.Flags().String("prior-manifest-file", "", "Path to a JSON file containing pre-cached gh-aw-manifests (map[lockFile]*GHAWManifest); used by the MCP server to supply a tamper-proof manifest baseline captured at startup")
	compileCmd.Flags().Bool("ghes", false, "Enable GitHub Enterprise Server (GHES) compatibility mode: emit upload-artifact@v3 and download-artifact@v3 instead of the latest v7/v8 which are not supported on GHES. Overrides the aw.json ghes field")
	if err := compileCmd.Flags().MarkHidden("prior-manifest-file"); err != nil {
		// Non-fatal: flag is registered even if MarkHidden fails
		_ = err
	}
	compileCmd.MarkFlagsMutuallyExclusive("dir", "workflows-dir")
	// --gh-aw-ref is a convenience alias for --action-mode release --action-tag <sha>;
	// combining it with either of those flags leads to one silently overwriting the other.
	compileCmd.MarkFlagsMutuallyExclusive("gh-aw-ref", "action-tag")
	compileCmd.MarkFlagsMutuallyExclusive("gh-aw-ref", "action-mode")

	// Register completions for compile command
	compileCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	cli.RegisterEngineFlagCompletion(compileCmd)
	cli.RegisterDirFlagCompletion(compileCmd, "dir")

	rootCmd.AddCommand(compileCmd)

	// Add flags to remove command
	removeCmd.Flags().Bool("keep-orphans", false, "Skip removal of orphaned include files that are no longer referenced by any workflow")
	removeCmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	// Register completions for remove command
	removeCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	cli.RegisterDirFlagCompletion(removeCmd, "dir")

	// Add flags to enable/disable commands
	enableCmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
	disableCmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
	// Register completions for enable/disable commands
	enableCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	disableCmd.ValidArgsFunction = cli.CompleteWorkflowNames

	// Add flags to run command
	runCmd.Flags().Int("repeat", 0, "Number of additional times to run after the initial execution (e.g., --repeat 3 runs 4 times total)")
	runCmd.Flags().Bool("enable-if-needed", false, "Enable the workflow before running if needed, and restore state afterward")
	runCmd.Flags().StringP("engine", "e", "", "Override AI engine (copilot, claude, codex, gemini, crush)")
	runCmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
	runCmd.Flags().String("ref", "", "Branch or tag name to run the workflow on (default: current branch)")
	runCmd.Flags().Bool("auto-merge-prs", false, "Auto-merge any pull requests created during the workflow execution")
	runCmd.Flags().StringArrayP("raw-field", "F", []string{}, "Add a string parameter in key=value format (can be used multiple times)")
	runCmd.Flags().Bool("push", false, "Commit and push workflow files (including transitive imports) before running")
	runCmd.Flags().Bool("dry-run", false, "Validate workflow without actually triggering execution on GitHub Actions")
	runCmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	runCmd.Flags().Bool("approve", false, "Approve all safe update changes. When strict mode is active (the default), the compiler emits warnings for new restricted secrets or unapproved action additions/removals not present in the existing gh-aw-manifest. Use this flag to approve and skip safe update enforcement")
	// Register completions for run command
	runCmd.ValidArgsFunction = cli.CompleteWorkflowNames
	cli.RegisterEngineFlagCompletion(runCmd)

	// Create and setup status command
	statusCmd := cli.NewStatusCommand()

	// Create and setup list command
	listCmd := cli.NewListCommand()

	// Create commands that need group assignment
	mcpCmd := cli.NewMCPCommand()
	logsCmd := cli.NewLogsCommand()
	auditCmd := cli.NewAuditCommand()
	viewCmd := cli.NewViewCommand()
	healthCmd := cli.NewHealthCommand()
	mcpServerCmd := cli.NewMCPServerCommand()
	prCmd := cli.NewPRCommand()
	secretsCmd := cli.NewSecretsCommand()
	fixCmd := cli.NewFixCommand()
	upgradeCmd := cli.NewUpgradeCommand()
	completionCmd := cli.NewCompletionCommand()
	hashCmd := cli.NewHashCommand()
	projectCmd := cli.NewProjectCommand()
	checksCmd := cli.NewChecksCommand()
	validateCmd := cli.NewValidateCommand(validateEngine)
	lintCmd := cli.NewLintCommand()
	domainsCmd := cli.NewDomainsCommand()
	experimentsCmd := cli.NewExperimentsCommand()
	forecastCmd := cli.NewForecastCommand()
	envCmd := cli.NewEnvCommand()

	// Assign commands to groups
	// Setup Commands
	initCmd.GroupID = "setup"
	newCmd.GroupID = "setup"
	addCmd.GroupID = "setup"
	addWizardCmd.GroupID = "setup"
	removeCmd.GroupID = "setup"
	updateCmd.GroupID = "setup"
	deployCmd.GroupID = "setup"
	upgradeCmd.GroupID = "setup"
	secretsCmd.GroupID = "setup"
	envCmd.GroupID = "setup"

	// Development Commands
	compileCmd.GroupID = "development"
	validateCmd.GroupID = "development"
	lintCmd.GroupID = "development"
	mcpCmd.GroupID = "development"
	fixCmd.GroupID = "development"
	domainsCmd.GroupID = "development"
	statusCmd.GroupID = "analysis"
	listCmd.GroupID = "analysis"

	// Execution Commands
	runCmd.GroupID = "execution"
	enableCmd.GroupID = "execution"
	disableCmd.GroupID = "execution"
	trialCmd.GroupID = "execution"

	// Analysis Commands
	logsCmd.GroupID = "analysis"
	auditCmd.GroupID = "analysis"
	viewCmd.GroupID = "analysis"
	healthCmd.GroupID = "analysis"
	checksCmd.GroupID = "analysis"
	experimentsCmd.GroupID = "analysis"
	forecastCmd.GroupID = "analysis"

	// Utilities
	mcpServerCmd.GroupID = "utilities"
	prCmd.GroupID = "utilities"
	completionCmd.GroupID = "utilities"
	hashCmd.GroupID = "utilities"
	projectCmd.GroupID = "utilities"

	// version command is intentionally left without a group (common practice)

	// Add all commands to root
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(addWizardCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(trialCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(initCmd)

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	outcomesCmd := cli.NewOutcomesCommand()
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(outcomesCmd)
	rootCmd.AddCommand(checksCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(mcpServerCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(fixCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(hashCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(domainsCmd)
	rootCmd.AddCommand(experimentsCmd)
	rootCmd.AddCommand(forecastCmd)
	rootCmd.AddCommand(envCmd)

	// Fix help flag descriptions for all subcommands to be consistent with the
	// root command ("Show help for gh aw" vs the Cobra default "help for [cmd]").
	var fixSubCmdHelpFlags func(cmd *cobra.Command)
	fixSubCmdHelpFlags = func(cmd *cobra.Command) {
		cmd.InitDefaultHelpFlag()
		if f := cmd.Flags().Lookup("help"); f != nil {
			cmdPath := cmd.CommandPath()
			// CommandPath() uses Name() which returns the first word of Use
			// ("gh" from "gh aw"), so subcommand paths look like "gh compile".
			// Replace the leading "gh " prefix with "gh aw " to match the root
			// command's display name.
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
