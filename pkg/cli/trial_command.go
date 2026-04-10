package cli

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewTrialCommand creates the trial command
func NewTrialCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trial <workflow-spec>...",
		Short: "Run one or more agentic workflows in trial mode against a simulated repository",
		Long: `Run one or more agentic workflows in trial mode as if they were running in a repository.

This command creates a temporary private repository in your GitHub space, installs the specified
workflow(s) from their source repositories, and runs them in "trial mode" to capture safe outputs without
making actual changes to the "simulated" host repository.

Single workflow:
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/weekly-research
  Outputs: stdout + local trials/weekly-research.DATETIME-ID.json + trial repo trials/

Multiple workflows (for comparison):
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/daily-plan githubnext/agentics/weekly-research
  Outputs: stdout + local trials/ + trial repo trials/ (individual + combined results)

Workflows from different repositories:
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/daily-plan myorg/myrepo/custom-workflow

Repository mode examples:
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --host-repo myorg/myrepo         # Run directly in myorg/myrepo (no simulation)
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --logical-repo myorg/myrepo  # Simulate running against myorg/myrepo
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --clone-repo myorg/myrepo   # Clone myorg/myrepo contents into host

Repeat and cleanup examples:
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --repeat 3                # Run 4 times total (1 initial + 3 repeats)
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --delete-host-repo-after  # Delete repo after completion
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --host-repo my-trial       # Custom host repo
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --dry-run                 # Show what would be done without changes

Auto-merge examples:
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --auto-merge-prs          # Auto-merge any PRs created during trial

Advanced examples:
  ` + string(constants.CLIExtensionPrefix) + ` trial githubnext/agentics/my-workflow --host-repo . # Use current repo as host
  ` + string(constants.CLIExtensionPrefix) + ` trial ./local-workflow.md --clone-repo upstream/repo --repeat 2

Repository modes:
- Default mode (no flags): Creates a temporary trial repository and simulates execution as if running against the current repository (github.repository context points to current repo)
- --logical-repo REPO: Simulates execution against a specified repository (github.repository context points to REPO while actually running in a temporary trial repository)
- --host-repo REPO: Uses the specified repository as the host for trial execution instead of creating a temporary one
- --clone-repo REPO: Clones the specified repository's contents into the trial repository before execution (useful for testing against actual repository state)

All workflows must support workflow_dispatch trigger to be used in trial mode.
The host repository will be created as private and kept by default unless --delete-host-repo-after is specified.
Trial results are saved both locally (in trials/ directory) and in the host repository for future reference.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing workflow specification\n\nUsage:\n  %s <workflow-spec>...\n\nExamples:\n  %[1]s githubnext/agentics/daily-plan             Trial a workflow from a repository\n  %[1]s ./local-workflow.md                         Trial a local workflow\n\nRun '%[1]s --help' for more information", cmd.CommandPath())
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowSpecs := args
			logicalRepoSpec, _ := cmd.Flags().GetString("logical-repo")
			cloneRepoSpec, _ := cmd.Flags().GetString("clone-repo")
			hostRepoSpec, _ := cmd.Flags().GetString("host-repo")
			repoSpec, _ := cmd.Flags().GetString("repo")
			deleteHostRepo, _ := cmd.Flags().GetBool("delete-host-repo-after")
			forceDeleteHostRepo, _ := cmd.Flags().GetBool("force-delete-host-repo-before")
			yes, _ := cmd.Flags().GetBool("yes")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			timeout, _ := cmd.Flags().GetInt("timeout")
			triggerContext, _ := cmd.Flags().GetString("trigger-context")
			repeatCount, _ := cmd.Flags().GetInt("repeat")
			autoMergePRs, _ := cmd.Flags().GetBool("auto-merge-prs")
			engineOverride, _ := cmd.Flags().GetString("engine")
			appendText, _ := cmd.Flags().GetString("append")
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
			disableSecurityScanner, _ := cmd.Flags().GetBool("disable-security-scanner")

			if err := validateEngine(engineOverride); err != nil {
				return err
			}
			// If --repo was used instead of --host-repo, use its value
			if repoSpec != "" {
				hostRepoSpec = repoSpec
			}

			opts := TrialOptions{
				Repos: RepoConfig{
					LogicalRepo: logicalRepoSpec,
					CloneRepo:   cloneRepoSpec,
					HostRepo:    hostRepoSpec,
				},
				DeleteHostRepo:         deleteHostRepo,
				ForceDelete:            forceDeleteHostRepo,
				Quiet:                  yes,
				DryRun:                 dryRun,
				TimeoutMinutes:         timeout,
				TriggerContext:         triggerContext,
				RepeatCount:            repeatCount,
				AutoMergePRs:           autoMergePRs,
				EngineOverride:         engineOverride,
				AppendText:             appendText,
				Verbose:                verbose,
				DisableSecurityScanner: disableSecurityScanner,
			}

			if err := RunWorkflowTrials(cmd.Context(), workflowSpecs, opts); err != nil {
				return err
			}
			return nil
		},
	}

	// Add flags
	cmd.Flags().StringP("logical-repo", "l", "", "Repository to simulate workflow execution against, as if the workflow was installed there (defaults to current repository)")
	cmd.Flags().String("clone-repo", "", "Alternative to --logical-repo: clone the contents of the specified repo into the host repo instead of using logical repository simulation")

	cmd.Flags().String("host-repo", "", "Custom host repository slug (defaults to '<username>/gh-aw-trial'). Use '.' for current repository")
	cmd.Flags().String("repo", "", "Alias for --host-repo: the repository where workflows are installed and run (note: different semantics from --repo in other commands)")
	_ = cmd.Flags().MarkHidden("repo") // Hide alias to avoid semantic confusion with --repo in other commands
	cmd.Flags().Bool("delete-host-repo-after", false, "Delete the host repository after completion (kept by default)")
	cmd.Flags().Bool("force-delete-host-repo-before", false, "Force delete the host repository before creation if it already exists")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without making any changes")
	cmd.Flags().Int("timeout", 30, "Execution timeout in minutes (e.g., 30 for 30 minutes)")
	cmd.Flags().String("trigger-context", "", "Trigger context URL (e.g., GitHub issue URL) for issue-triggered workflows")
	cmd.Flags().Int("repeat", 0, "Number of additional times to run after the initial execution (e.g., --repeat 3 runs 4 times total)")
	cmd.Flags().Bool("auto-merge-prs", false, "Auto-merge any pull requests created during trial execution")
	addEngineFlag(cmd)
	cmd.Flags().String("append", "", "Append extra content to the end of agentic workflow on installation")
	cmd.Flags().Bool("disable-security-scanner", false, "Disable security scanning of workflow markdown content")
	cmd.MarkFlagsMutuallyExclusive("host-repo", "repo")
	cmd.MarkFlagsMutuallyExclusive("logical-repo", "clone-repo")

	return cmd
}
