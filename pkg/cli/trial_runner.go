package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var trialLog = logger.New("cli:trial_command")

// RunWorkflowTrials executes the main logic for trialing one or more workflows
func RunWorkflowTrials(ctx context.Context, workflowSpecs []string, opts TrialOptions) error {
	trialLog.Printf("Starting trial execution: specs=%v, logicalRepo=%s, cloneRepo=%s, hostRepo=%s, repeat=%d", workflowSpecs, opts.Repos.LogicalRepo, opts.Repos.CloneRepo, opts.Repos.HostRepo, opts.RepeatCount)

	// Show welcome banner for interactive mode
	console.ShowWelcomeBanner("This tool will run a trial of your workflow in a test repository.")

	parsedSpecs, err := runWorkflowTrialsParseSpecs(workflowSpecs)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("[DRY RUN] Showing what would be done without making changes"))
	}
	runWorkflowTrialsPrintStart(parsedSpecs)

	mode, err := runWorkflowTrialsDetermineMode(opts)
	if err != nil {
		return err
	}
	hostRepoSlug, err := runWorkflowTrialsHostRepo(ctx, opts)
	if err != nil {
		return err
	}

	if err := runWorkflowTrialsConfirm(parsedSpecs, mode, hostRepoSlug, opts); err != nil {
		return err
	}

	// Step 2: Create or reuse host repository
	trialLog.Printf("Ensuring trial repository exists: %s", hostRepoSlug)
	if err := ensureTrialRepository(hostRepoSlug, mode.cloneRepoSlug, opts.ForceDelete, opts.DryRun, opts.Verbose); err != nil {
		return fmt.Errorf("failed to ensure host repository: %w", err)
	}

	// In dry-run mode, stop here after showing what would be done
	if opts.DryRun {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("[DRY RUN] Stopping here. No actual changes were made."))
		return nil
	}

	cleanup, err := runWorkflowTrialsPrepareExecution(ctx, mode, hostRepoSlug, opts)
	if err != nil {
		return err
	}
	defer cleanup()

	// Step 2.8: Disable all workflows except the ones being trialled (only in clone-repo mode, done once before all trials)
	if err := runWorkflowTrialsDisableClonedWorkflows(parsedSpecs, mode.cloneRepoSlug, hostRepoSlug, opts); err != nil {
		return err
	}

	return runWorkflowTrialsExecute(ctx, parsedSpecs, mode, hostRepoSlug, opts)

}

func runWorkflowTrialsPrepareExecution(ctx context.Context, mode runWorkflowTrialsMode, hostRepoSlug string, opts TrialOptions) (func(), error) {
	// Step 2.5: Ensure engine secrets are configured when an explicit engine override is provided
	// When no override is specified, the workflow will use its frontmatter engine and handle secrets during compilation
	if err := runWorkflowTrialsEnsureEngineSecrets(ctx, hostRepoSlug, opts); err != nil {
		return func() {}, err
	}

	cleanup := runWorkflowTrialsSetupCleanup(hostRepoSlug, opts)
	if cleanup == nil {
		cleanup = func() {}
	}

	// Step 2.7: Clone source repository contents if in clone-repo mode
	if mode.cloneRepoSlug != "" {
		if err := cloneRepoContentsIntoHost(mode.cloneRepoSlug, mode.cloneRepoVersion, hostRepoSlug, opts.Verbose); err != nil {
			cleanup()
			return func() {}, fmt.Errorf("failed to clone repository contents: %w", err)
		}
	}
	return cleanup, nil
}

type runWorkflowTrialsMode struct {
	logicalRepoSlug  string
	cloneRepoSlug    string
	cloneRepoVersion string
	directTrialMode  bool
}

func runWorkflowTrialsParseSpecs(workflowSpecs []string) ([]*WorkflowSpec, error) {
	var parsedSpecs []*WorkflowSpec
	for _, spec := range workflowSpecs {
		parsedSpec, err := parseWorkflowSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid workflow specification '%s': %w", spec, err)
		}
		parsedSpecs = append(parsedSpecs, parsedSpec)
	}
	return parsedSpecs, nil
}

func runWorkflowTrialsPrintStart(parsedSpecs []*WorkflowSpec) {
	if len(parsedSpecs) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of workflow '%s' from '%s'", parsedSpecs[0].WorkflowName, parsedSpecs[0].RepoSlug)))
		return
	}

	workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	joinedNames := strings.Join(workflowNames, ", ")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of %d workflows (%s)", len(parsedSpecs), joinedNames)))
}

func runWorkflowTrialsDetermineMode(opts TrialOptions) (runWorkflowTrialsMode, error) {
	if opts.Repos.CloneRepo != "" {
		cloneRepo, err := parseRepoSpec(opts.Repos.CloneRepo)
		if err != nil {
			return runWorkflowTrialsMode{}, fmt.Errorf("invalid --clone-repo specification '%s': %w", opts.Repos.CloneRepo, err)
		}
		trialLog.Printf("Using clone-repo mode: %s (version=%s)", cloneRepo.RepoSlug, cloneRepo.Version)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Clone mode: Will clone contents from %s into host repository", cloneRepo.RepoSlug)))
		return runWorkflowTrialsMode{cloneRepoSlug: cloneRepo.RepoSlug, cloneRepoVersion: cloneRepo.Version}, nil
	}
	if opts.Repos.LogicalRepo != "" {
		logicalRepo, err := parseRepoSpec(opts.Repos.LogicalRepo)
		if err != nil {
			return runWorkflowTrialsMode{}, fmt.Errorf("invalid --logical-repo specification '%s': %w", opts.Repos.LogicalRepo, err)
		}
		trialLog.Printf("Using logical-repo mode: %s", logicalRepo.RepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Target repository (specified): "+logicalRepo.RepoSlug))
		return runWorkflowTrialsMode{logicalRepoSlug: logicalRepo.RepoSlug}, nil
	}
	return runWorkflowTrialsDefaultMode(opts)
}

func runWorkflowTrialsDefaultMode(opts TrialOptions) (runWorkflowTrialsMode, error) {
	if opts.Repos.HostRepo != "" {
		trialLog.Print("Using direct trial mode (no simulation)")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Direct trial mode: Workflows will be installed and run directly in the specified repository"))
		return runWorkflowTrialsMode{directTrialMode: true}, nil
	}

	logicalRepoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		return runWorkflowTrialsMode{}, fmt.Errorf("failed to determine simulated host repository: %w", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Target repository (current): "+logicalRepoSlug))
	return runWorkflowTrialsMode{logicalRepoSlug: logicalRepoSlug}, nil
}

func runWorkflowTrialsHostRepo(ctx context.Context, opts TrialOptions) (string, error) {
	if opts.Repos.HostRepo != "" {
		hostRepo, err := parseRepoSpec(opts.Repos.HostRepo)
		if err != nil {
			return "", fmt.Errorf("invalid --host-repo specification '%s': %w", opts.Repos.HostRepo, err)
		}
		trialLog.Printf("Using specified host repository: %s", hostRepo.RepoSlug)
		return hostRepo.RepoSlug, nil
	}

	username, err := getCurrentGitHubUsername(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username for default trial repo: %w", err)
	}
	hostRepoSlug := username + "/gh-aw-trial"
	trialLog.Printf("Using default host repository: %s", hostRepoSlug)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Host repository (default): "+hostRepoSlug))
	return hostRepoSlug, nil
}

func runWorkflowTrialsConfirm(parsedSpecs []*WorkflowSpec, mode runWorkflowTrialsMode, hostRepoSlug string, opts TrialOptions) error {
	if opts.Quiet {
		return nil
	}
	return showTrialConfirmation(trialConfirmationOptions{
		parsedSpecs:         parsedSpecs,
		logicalRepoSlug:     mode.logicalRepoSlug,
		cloneRepoSlug:       mode.cloneRepoSlug,
		hostRepoSlug:        hostRepoSlug,
		deleteHostRepo:      opts.DeleteHostRepo,
		forceDeleteHostRepo: opts.ForceDelete,
		autoMergePRs:        opts.AutoMergePRs,
		repeatCount:         opts.RepeatCount,
		directTrialMode:     mode.directTrialMode,
		engineOverride:      opts.EngineOverride,
	})
}

func runWorkflowTrialsEnsureEngineSecrets(ctx context.Context, hostRepoSlug string, opts TrialOptions) error {
	if opts.EngineOverride == "" {
		return nil
	}

	existingSecrets, err := getExistingSecretsInRepo(hostRepoSlug)
	if err != nil {
		trialLog.Printf("Warning: could not check existing secrets: %v", err)
		existingSecrets = make(map[string]struct{})
	}

	secretConfig := EngineSecretConfig{
		Ctx:                  ctx,
		RepoSlug:             hostRepoSlug,
		Engine:               opts.EngineOverride,
		Verbose:              opts.Verbose,
		ExistingSecrets:      existingSecrets,
		IncludeSystemSecrets: false,
		IncludeOptional:      false,
	}
	if err := checkAndEnsureEngineSecretsForEngine(secretConfig); err != nil {
		return fmt.Errorf("failed to configure engine secret: %w", err)
	}
	return nil
}

func runWorkflowTrialsSetupCleanup(hostRepoSlug string, opts TrialOptions) func() {
	if !opts.DeleteHostRepo {
		return nil
	}
	return func() {
		if err := cleanupTrialRepository(hostRepoSlug, opts.Verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup host repository: %v", err)))
		}
	}
}

func runWorkflowTrialsDisableClonedWorkflows(parsedSpecs []*WorkflowSpec, cloneRepoSlug, hostRepoSlug string, opts TrialOptions) error {
	if cloneRepoSlug == "" {
		return nil
	}

	workflowsToKeep := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Disabling workflows in cloned repository (keeping: %s)", strings.Join(workflowsToKeep, ", "))))
	}

	tempDirForDisable, err := cloneTrialHostRepository(hostRepoSlug, opts.Verbose)
	if err != nil {
		return fmt.Errorf("failed to clone host repository for workflow disabling: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDirForDisable); err != nil {
			trialLog.Printf("Failed to cleanup temp directory for workflow disabling: %v", err)
		}
	}()
	return runWorkflowTrialsDisableInDir(tempDirForDisable, workflowsToKeep, opts.Verbose)
}

func runWorkflowTrialsDisableInDir(tempDirForDisable string, workflowsToKeep []string, verbose bool) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := os.Chdir(tempDirForDisable); err != nil {
		return fmt.Errorf("failed to change to temp directory: %w", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			trialLog.Printf("Failed to change back to original directory: %v", err)
		}
	}()

	disableErr := DisableAllWorkflowsExcept("", workflowsToKeep, verbose)
	if disableErr != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to disable workflows: %v", disableErr)))
	}
	return nil
}

func runWorkflowTrialsExecute(ctx context.Context, parsedSpecs []*WorkflowSpec, mode runWorkflowTrialsMode, hostRepoSlug string, opts TrialOptions) error {
	return ExecuteWithRepeat(RepeatOptions{
		RepeatCount:   opts.RepeatCount,
		RepeatMessage: "Repeating trial run",
		ExecuteFunc: func() error {
			return executeTrialRun(ctx, parsedSpecs, hostRepoSlug, mode.logicalRepoSlug, mode.cloneRepoSlug, mode.directTrialMode, opts)
		},
		CleanupFunc: func() {
			if opts.DeleteHostRepo {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Host repository will be cleaned up"))
			} else {
				githubHost := getGitHubHost()
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Host repository preserved: %s/%s", githubHost, hostRepoSlug)))
			}
		},
		UseStderr: true,
	})
}

// getCurrentGitHubUsername gets the current GitHub username from gh CLI
func getCurrentGitHubUsername(ctx context.Context) (string, error) {
	output, err := workflow.RunGHContext(ctx, "Fetching GitHub username...", "api", "user", "--jq", ".login")
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username: %w", err)
	}

	username := strings.TrimSpace(string(output))
	if username == "" {
		return "", errors.New("GitHub username is empty")
	}

	return username, nil
}
