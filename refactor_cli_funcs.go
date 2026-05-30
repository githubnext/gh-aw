package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "sort"
)

type replacement struct {
    name string
    body string
}

func main() {
    replacements := map[string][]replacement{
        "pkg/cli/trial_runner.go": {{name: "RunWorkflowTrials", body: `// RunWorkflowTrials executes the main logic for trialing one or more workflows
func RunWorkflowTrials(ctx context.Context, workflowSpecs []string, opts TrialOptions) error {
	trialLog.Printf("Starting trial execution: specs=%v, logicalRepo=%s, cloneRepo=%s, hostRepo=%s, repeat=%d", workflowSpecs, opts.Repos.LogicalRepo, opts.Repos.CloneRepo, opts.Repos.HostRepo, opts.RepeatCount)
	console.ShowWelcomeBanner("This tool will run a trial of your workflow in a test repository.")

	parsedSpecs, err := parseWorkflowTrialSpecs(workflowSpecs)
	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("[DRY RUN] Showing what would be done without making changes"))
	}
	displayTrialWorkflowStart(parsedSpecs)

	modeConfig, err := resolveTrialModeConfig(ctx, opts)
	if err != nil {
		return err
	}
	hostRepoSlug, err := determineTrialHostRepoSlug(ctx, opts)
	if err != nil {
		return err
	}

	if !opts.Quiet {
		if err := showTrialConfirmation(trialConfirmationOptions{
			parsedSpecs:         parsedSpecs,
			logicalRepoSlug:     modeConfig.logicalRepoSlug,
			cloneRepoSlug:       modeConfig.cloneRepoSlug,
			hostRepoSlug:        hostRepoSlug,
			deleteHostRepo:      opts.DeleteHostRepo,
			forceDeleteHostRepo: opts.ForceDelete,
			autoMergePRs:        opts.AutoMergePRs,
			repeatCount:         opts.RepeatCount,
			directTrialMode:     modeConfig.directTrialMode,
			engineOverride:      opts.EngineOverride,
		}); err != nil {
			return err
		}
	}

	if err := ensureTrialRepository(hostRepoSlug, modeConfig.cloneRepoSlug, opts.ForceDelete, opts.DryRun, opts.Verbose); err != nil {
		return fmt.Errorf("failed to ensure host repository: %w", err)
	}
	if opts.DryRun {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("[DRY RUN] Stopping here. No actual changes were made."))
		return nil
	}
	if err := ensureTrialEngineSecret(ctx, hostRepoSlug, opts); err != nil {
		return err
	}
	if opts.DeleteHostRepo {
		defer cleanupTrialHostRepository(hostRepoSlug, opts.Verbose)
	}
	if err := prepareClonedTrialHost(modeConfig.cloneRepoSlug, modeConfig.cloneRepoVersion, hostRepoSlug, parsedSpecs, opts.Verbose); err != nil {
		return err
	}

	return ExecuteWithRepeat(RepeatOptions{
		RepeatCount:   opts.RepeatCount,
		RepeatMessage: "Repeating trial run",
		ExecuteFunc: func() error {
			return executeTrialRun(ctx, parsedSpecs, hostRepoSlug, modeConfig.logicalRepoSlug, modeConfig.cloneRepoSlug, modeConfig.directTrialMode, opts)
		},
		CleanupFunc: func() {
			if opts.DeleteHostRepo {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Host repository will be cleaned up"))
				return
			}
			githubHost := getGitHubHost()
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Host repository preserved: %s/%s", githubHost, hostRepoSlug)))
		},
		UseStderr: true,
	})
}

type trialModeConfig struct {
	logicalRepoSlug  string
	cloneRepoSlug    string
	cloneRepoVersion string
	directTrialMode  bool
}

func parseWorkflowTrialSpecs(workflowSpecs []string) ([]*WorkflowSpec, error) {
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

func displayTrialWorkflowStart(parsedSpecs []*WorkflowSpec) {
	if len(parsedSpecs) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of workflow '%s' from '%s'", parsedSpecs[0].WorkflowName, parsedSpecs[0].RepoSlug)))
		return
	}
	workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	joinedNames := strings.Join(workflowNames, ", ")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of %d workflows (%s)", len(parsedSpecs), joinedNames)))
}

func resolveTrialModeConfig(ctx context.Context, opts TrialOptions) (*trialModeConfig, error) {
	_ = ctx
	if opts.Repos.CloneRepo != "" {
		cloneRepo, err := parseRepoSpec(opts.Repos.CloneRepo)
		if err != nil {
			return nil, fmt.Errorf("invalid --clone-repo specification '%s': %w", opts.Repos.CloneRepo, err)
		}
		trialLog.Printf("Using clone-repo mode: %s (version=%s)", cloneRepo.RepoSlug, cloneRepo.Version)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Clone mode: Will clone contents from %s into host repository", cloneRepo.RepoSlug)))
		return &trialModeConfig{cloneRepoSlug: cloneRepo.RepoSlug, cloneRepoVersion: cloneRepo.Version}, nil
	}
	if opts.Repos.LogicalRepo != "" {
		logicalRepo, err := parseRepoSpec(opts.Repos.LogicalRepo)
		if err != nil {
			return nil, fmt.Errorf("invalid --logical-repo specification '%s': %w", opts.Repos.LogicalRepo, err)
		}
		trialLog.Printf("Using logical-repo mode: %s", logicalRepo.RepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Target repository (specified): "+logicalRepo.RepoSlug))
		return &trialModeConfig{logicalRepoSlug: logicalRepo.RepoSlug}, nil
	}
	if opts.Repos.HostRepo != "" {
		trialLog.Print("Using direct trial mode (no simulation)")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Direct trial mode: Workflows will be installed and run directly in the specified repository"))
		return &trialModeConfig{directTrialMode: true}, nil
	}
	logicalRepoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		return nil, fmt.Errorf("failed to determine simulated host repository: %w", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Target repository (current): "+logicalRepoSlug))
	return &trialModeConfig{logicalRepoSlug: logicalRepoSlug}, nil
}

func determineTrialHostRepoSlug(ctx context.Context, opts TrialOptions) (string, error) {
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

func ensureTrialEngineSecret(ctx context.Context, hostRepoSlug string, opts TrialOptions) error {
	if opts.EngineOverride == "" {
		return nil
	}
	existingSecrets, err := getExistingSecretsInRepo(hostRepoSlug)
	if err != nil {
		trialLog.Printf("Warning: could not check existing secrets: %v", err)
		existingSecrets = make(map[string]bool)
	}
	secretConfig := EngineSecretConfig{Ctx: ctx, RepoSlug: hostRepoSlug, Engine: opts.EngineOverride, Verbose: opts.Verbose, ExistingSecrets: existingSecrets, IncludeSystemSecrets: false, IncludeOptional: false}
	if err := checkAndEnsureEngineSecretsForEngine(secretConfig); err != nil {
		return fmt.Errorf("failed to configure engine secret: %w", err)
	}
	return nil
}

func cleanupTrialHostRepository(hostRepoSlug string, verbose bool) {
	if err := cleanupTrialRepository(hostRepoSlug, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup host repository: %v", err)))
	}
}

func prepareClonedTrialHost(cloneRepoSlug, cloneRepoVersion, hostRepoSlug string, parsedSpecs []*WorkflowSpec, verbose bool) error {
	if cloneRepoSlug == "" {
		return nil
	}
	if err := cloneRepoContentsIntoHost(cloneRepoSlug, cloneRepoVersion, hostRepoSlug, verbose); err != nil {
		return fmt.Errorf("failed to clone repository contents: %w", err)
	}
	return disableClonedRepoWorkflows(parsedSpecs, hostRepoSlug, verbose)
}

func disableClonedRepoWorkflows(parsedSpecs []*WorkflowSpec, hostRepoSlug string, verbose bool) error {
	workflowsToKeep := workflowNamesToKeep(parsedSpecs)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Disabling workflows in cloned repository (keeping: %s)", strings.Join(workflowsToKeep, ", "))))
	}
	return withClonedTrialHostRepository(hostRepoSlug, verbose, func() error {
		disableErr := DisableAllWorkflowsExcept("", workflowsToKeep, verbose)
		if disableErr != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to disable workflows: %v", disableErr)))
		}
		return nil
	})
}

func workflowNamesToKeep(parsedSpecs []*WorkflowSpec) []string {
	workflowsToKeep := make([]string, 0, len(parsedSpecs))
	for _, spec := range parsedSpecs {
		workflowsToKeep = append(workflowsToKeep, spec.WorkflowName)
	}
	return workflowsToKeep
}

func withClonedTrialHostRepository(hostRepoSlug string, verbose bool, fn func() error) error {
	tempDirForDisable, err := cloneTrialHostRepository(hostRepoSlug, verbose)
	if err != nil {
		return fmt.Errorf("failed to clone host repository for workflow disabling: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDirForDisable); err != nil {
			trialLog.Printf("Failed to cleanup temp directory for workflow disabling: %v", err)
		}
	}()
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
	return fn()
}`}},
        "pkg/cli/trial_support.go": {{name: "downloadAllArtifacts", body: `// downloadAllArtifacts downloads and parses all available artifacts from a workflow run
func downloadAllArtifacts(hostRepoSlug, runID string, verbose bool) (*TrialArtifacts, error) {
	trialSupportLog.Printf("Downloading artifacts: repo=%s, runID=%s", hostRepoSlug, runID)
	tempDir, err := createTrialArtifactTempDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	hasArtifacts, err := downloadTrialArtifacts(hostRepoSlug, runID, tempDir, verbose)
	if err != nil {
		return nil, err
	}
	artifacts := &TrialArtifacts{AdditionalArtifacts: make(map[string]any)}
	if !hasArtifacts {
		return artifacts, nil
	}
	if err := walkTrialArtifactFiles(tempDir, artifacts, verbose); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Error walking artifact directory: %v", err)))
	}
	return artifacts, nil
}

func createTrialArtifactTempDir() (string, error) {
	tempDir, err := os.MkdirTemp("", "trial-artifacts-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	return tempDir, nil
}

func downloadTrialArtifacts(repoSlug, runID, tempDir string, verbose bool) (bool, error) {
	output, err := workflow.RunGHCombined("Downloading artifacts...", "run", "download", runID, "--repo", repoSlug, "--dir", tempDir)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No artifacts found for run %s: %s", runID, string(output))))
		}
		return false, nil
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Downloaded all artifacts for run %s to %s", runID, tempDir)))
	}
	return true, nil
}

func walkTrialArtifactFiles(tempDir string, artifacts *TrialArtifacts, verbose bool) error {
	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		return processTrialArtifactFile(tempDir, path, artifacts, verbose)
	})
}

func processTrialArtifactFile(tempDir, path string, artifacts *TrialArtifacts, verbose bool) error {
	relPath, err := filepath.Rel(tempDir, path)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to get relative path for %s: %v", path, err)))
		}
		return nil
	}
	switch {
	case strings.HasSuffix(path, constants.AgentOutputFilename):
		trialSupportLog.Printf("Processing safe outputs artifact: %s", relPath)
		if safeOutputs := parseJSONArtifact(path, verbose); safeOutputs != nil {
			artifacts.SafeOutputs = safeOutputs
		}
	case strings.HasSuffix(path, "aw_info.json"):
		trialSupportLog.Printf("Processing agentic run info artifact: %s", relPath)
		if runInfo := parseJSONArtifact(path, verbose); runInfo != nil {
			artifacts.AgenticRunInfo = runInfo
		}
	case isStructuredTrialArtifact(path):
		storeStructuredTrialArtifact(relPath, path, artifacts, verbose)
	}
	return nil
}

func isStructuredTrialArtifact(path string) bool {
	return strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".log") || strings.HasSuffix(path, ".txt")
}

func storeStructuredTrialArtifact(relPath, path string, artifacts *TrialArtifacts, verbose bool) {
	if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") {
		if content := parseJSONArtifact(path, verbose); content != nil {
			artifacts.AdditionalArtifacts[relPath] = content
		}
		return
	}
	if content := readTextArtifact(path, verbose); content != "" {
		artifacts.AdditionalArtifacts[relPath] = content
	}
}`}},
        "pkg/cli/run_interactive.go": {
            {name: "RunWorkflowInteractively", body: `// RunWorkflowInteractively runs a workflow in interactive mode
func RunWorkflowInteractively(ctx context.Context, verbose bool, repoOverride string, refOverride string, autoMergePRs bool, push bool, engineOverride string, dryRun bool) error {
	runInteractiveLog.Print("Starting interactive workflow run")
	if err := ensureInteractiveWorkflowMode(verbose); err != nil {
		return err
	}
	workflows, err := findRunnableWorkflows(verbose)
	if err != nil {
		return fmt.Errorf("failed to find runnable workflows: %w", err)
	}
	if len(workflows) == 0 {
		return errors.New("no runnable workflows found. Workflows must have 'workflow_dispatch' trigger")
	}
	selectedWorkflow, err := selectWorkflow(ctx, workflows)
	if err != nil {
		return fmt.Errorf("workflow selection cancelled or failed: %w", err)
	}
	runInteractiveLog.Printf("Selected workflow: %s", selectedWorkflow.Name)
	showWorkflowInfo(selectedWorkflow)
	inputValues, err := collectWorkflowInputs(ctx, selectedWorkflow)
	if err != nil {
		return fmt.Errorf("failed to collect workflow inputs: %w", err)
	}
	if !confirmExecution(ctx, selectedWorkflow, inputValues) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow execution cancelled"))
		return nil
	}
	cmdStr := buildCommandString(selectedWorkflow.Name, inputValues, repoOverride, refOverride, autoMergePRs, push, engineOverride)
	showInteractiveWorkflowCommand(cmdStr)
	if err := runInteractiveWorkflow(ctx, selectedWorkflow.Name, RunOptions{Enable: false, EngineOverride: engineOverride, RepoOverride: repoOverride, RefOverride: refOverride, AutoMergePRs: autoMergePRs, Push: push, Inputs: inputValues, Verbose: verbose, DryRun: dryRun}); err != nil {
		return err
	}
	showInteractiveWorkflowSuccess(cmdStr)
	return nil
}

func ensureInteractiveWorkflowMode(verbose bool) error {
	if IsRunningInCI() {
		return errors.New("interactive mode cannot be used in CI environments")
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting interactive workflow run..."))
	}
	return nil
}

func showInteractiveWorkflowCommand(cmdStr string) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nRunning workflow..."))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("Equivalent command: "+cmdStr))
	fmt.Fprintln(os.Stderr, "")
}

func runInteractiveWorkflow(ctx context.Context, workflowName string, opts RunOptions) error {
	if err := RunWorkflowOnGitHub(ctx, workflowName, opts); err != nil {
		return fmt.Errorf("failed to run workflow: %w", err)
	}
	return nil
}

func showInteractiveWorkflowSuccess(cmdStr string) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Workflow dispatched successfully!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To run this workflow again, use:"))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(cmdStr))
}`},
            {name: "collectInputsWithMap", body: `// collectInputsWithMap collects inputs using a map to properly capture values
func collectInputsWithMap(ctx context.Context, inputs map[string]*workflow.InputDefinition) ([]string, error) {
	inputPtrs, formGroups := buildWorkflowInputForm(inputs)
	form := huh.NewForm(formGroups...).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())
	if err := form.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("input collection cancelled: %w", err)
	}
	result := collectNonEmptyWorkflowInputs(inputPtrs)
	runInteractiveLog.Printf("Collected %d input values", len(result))
	return result, nil
}

func buildWorkflowInputForm(inputs map[string]*workflow.InputDefinition) (map[string]*string, []*huh.Group) {
	inputPtrs := make(map[string]*string)
	formGroups := make([]*huh.Group, 0, len(inputs))
	for name, input := range inputs {
		defaultStr := ""
		if input.Default != nil {
			defaultStr = fmt.Sprintf("%v", input.Default)
		}
		valueStr := defaultStr
		inputPtrs[name] = &valueStr
		formGroups = append(formGroups, huh.NewGroup(newWorkflowInputField(name, input, inputPtrs[name])))
	}
	return inputPtrs, formGroups
}

func newWorkflowInputField(inputName string, inputDef *workflow.InputDefinition, valuePtr *string) *huh.Input {
	field := huh.NewInput().Title(fmt.Sprintf("Enter value for '%s'", inputName)).Value(valuePtr)
	if inputDef.Description != "" {
		field = field.Description(inputDef.Description)
	}
	if inputDef.Required {
		field = field.Validate(func(s string) error {
			if s == "" {
				return errors.New("this input is required")
			}
			return nil
		})
	}
	return field
}

func collectNonEmptyWorkflowInputs(inputPtrs map[string]*string) []string {
	var result []string
	for name, valuePtr := range inputPtrs {
		if value := *valuePtr; value != "" {
			result = append(result, fmt.Sprintf("%s=%s", name, value))
		}
	}
	return result
}`},
            {name: "RunSpecificWorkflowInteractively", body: `// RunSpecificWorkflowInteractively runs a specific workflow in interactive mode
// This is similar to RunWorkflowInteractively but skips the workflow selection step
// since the workflow name is already known. It will still collect inputs if the workflow has them.
func RunSpecificWorkflowInteractively(ctx context.Context, opts RunWorkflowOptions) error {
	runInteractiveLog.Printf("Running specific workflow interactively: %s", opts.WorkflowName)
	wf, err := getSpecificInteractiveWorkflowOption(opts)
	if err != nil {
		return err
	}
	if len(wf.Inputs) > 0 {
		showWorkflowInfo(wf)
	}
	inputValues, err := collectWorkflowInputs(ctx, wf)
	if err != nil {
		return fmt.Errorf("failed to collect workflow inputs: %w", err)
	}
	if len(inputValues) > 0 && !confirmExecution(ctx, wf, inputValues) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow execution cancelled"))
		return nil
	}
	cmdStr := buildCommandString(opts.WorkflowName, inputValues, opts.RepoOverride, opts.RefOverride, opts.AutoMergePRs, opts.Push, opts.EngineOverride)
	showInteractiveWorkflowCommand(cmdStr)
	return runInteractiveWorkflow(ctx, opts.WorkflowName, RunOptions{Enable: false, EngineOverride: opts.EngineOverride, RepoOverride: opts.RepoOverride, RefOverride: opts.RefOverride, AutoMergePRs: opts.AutoMergePRs, Push: opts.Push, WaitForCompletion: true, Inputs: inputValues, Verbose: opts.Verbose, DryRun: opts.DryRun})
}

func getSpecificInteractiveWorkflowOption(opts RunWorkflowOptions) (*WorkflowOption, error) {
	workflowsDir := constants.GetWorkflowDir()
	mdFile := filepath.Join(workflowsDir, opts.WorkflowName+".md")
	if _, err := os.Stat(mdFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflow file not found: %s", mdFile)
	}
	inputs, err := getWorkflowInputs(mdFile)
	if err != nil {
		runInteractiveLog.Printf("Failed to get inputs for workflow %s: %v", opts.WorkflowName, err)
		inputs = nil
	}
	return &WorkflowOption{Name: opts.WorkflowName, Description: buildWorkflowDescription(inputs), FilePath: mdFile, Inputs: inputs}, nil
}`},
        },
        "pkg/cli/run_workflow_tracking.go": {{name: "getLatestWorkflowRunWithRetry", body: `// getLatestWorkflowRunWithRetry gets information about the most recent run of the specified workflow
// with retry logic to handle timing issues when a workflow has just been triggered
func getLatestWorkflowRunWithRetry(lockFileName string, repo string, verbose bool) (*WorkflowRunInfo, error) {
	runWorkflowTrackingLog.Printf("Getting latest workflow run: workflow=%s, repo=%s, max_retries=6", lockFileName, repo)
	const maxRetries = 6
	const initialDelay = 2 * time.Second
	const maxDelay = 10 * time.Second

	logWorkflowRunLookup(lockFileName, repo, verbose)
	startTime := time.Now().UTC()
	spinner := newWorkflowRunLookupSpinner(verbose)
	var lastErr error

	for attempt := range maxRetries {
		waitForWorkflowRunRetry(attempt, maxRetries, startTime, initialDelay, maxDelay, verbose, spinner)
		runInfo, shouldReturn, err := lookupWorkflowRunAttempt(lockFileName, repo, verbose, startTime, attempt, maxRetries, spinner)
		if err != nil {
			lastErr = err
			continue
		}
		if shouldReturn {
			return runInfo, nil
		}
		lastErr = errors.New("workflow run appears to be from a previous execution")
	}
	if spinner != nil {
		spinner.Stop()
	}
	if lastErr != nil {
		return nil, fmt.Errorf("failed to get workflow run after %d attempts: %w", maxRetries, lastErr)
	}
	return nil, fmt.Errorf("no workflow run found after %d attempts", maxRetries)
}

type workflowRunListEntry struct {
	URL        string `json:"url"`
	DatabaseID int64  `json:"databaseId"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"createdAt"`
}

func logWorkflowRunLookup(lockFileName, repo string, verbose bool) {
	if repo != "" {
		console.LogVerbose(verbose, fmt.Sprintf("Getting latest run for workflow: %s in repo: %s (with retry logic)", lockFileName, repo))
		return
	}
	console.LogVerbose(verbose, fmt.Sprintf("Getting latest run for workflow: %s (with retry logic)", lockFileName))
}

func newWorkflowRunLookupSpinner(verbose bool) *console.SpinnerWrapper {
	if verbose {
		return nil
	}
	return console.NewSpinner("Waiting for workflow run to appear...")
}

func waitForWorkflowRunRetry(attempt, maxRetries int, startTime time.Time, initialDelay, maxDelay time.Duration, verbose bool, spinner *console.SpinnerWrapper) {
	if attempt == 0 {
		return
	}
	delay := min(time.Duration(attempt)*initialDelay, maxDelay)
	elapsed := time.Since(startTime).Round(time.Second)
	console.LogVerbose(verbose, fmt.Sprintf("Waiting %v before retry attempt %d/%d...", delay, attempt+1, maxRetries))
	if !verbose && spinner != nil {
		if attempt == 1 {
			spinner.Start()
		}
		spinner.UpdateMessage(fmt.Sprintf("Waiting for workflow run... (attempt %d/%d, %v elapsed)", attempt+1, maxRetries, elapsed))
	}
	time.Sleep(delay)
}

func lookupWorkflowRunAttempt(lockFileName, repo string, verbose bool, startTime time.Time, attempt, maxRetries int, spinner *console.SpinnerWrapper) (*WorkflowRunInfo, bool, error) {
	output, err := fetchLatestWorkflowRunOutput(lockFileName, repo)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Attempt %d/%d failed: %v", attempt+1, maxRetries, err)))
		}
		return nil, false, fmt.Errorf("failed to get workflow runs: %w", err)
	}
	entries, err := parseWorkflowRunEntries(output)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Attempt %d/%d failed to parse JSON: %v", attempt+1, maxRetries, err)))
		}
		return nil, false, fmt.Errorf("failed to parse workflow run data: %w", err)
	}
	if len(entries) == 0 {
		console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: no runs found yet", attempt+1, maxRetries))
		return nil, false, errors.New("no runs found for workflow")
	}
	return evaluateWorkflowRunAttempt(entries[0], verbose, startTime, attempt, maxRetries, spinner)
}

func fetchLatestWorkflowRunOutput(lockFileName, repo string) ([]byte, error) {
	if repo != "" {
		return workflow.ExecGH("run", "list", "--repo", repo, "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt").Output()
	}
	return workflow.ExecGH("run", "list", "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt").Output()
}

func parseWorkflowRunEntries(output []byte) ([]workflowRunListEntry, error) {
	if len(output) == 0 || string(output) == "[]" {
		return nil, nil
	}
	var runs []workflowRunListEntry
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func evaluateWorkflowRunAttempt(run workflowRunListEntry, verbose bool, startTime time.Time, attempt, maxRetries int, spinner *console.SpinnerWrapper) (*WorkflowRunInfo, bool, error) {
	createdAt := parseWorkflowRunTime(run.CreatedAt, verbose)
	runInfo := &WorkflowRunInfo{URL: run.URL, DatabaseID: run.DatabaseID, Status: run.Status, Conclusion: run.Conclusion, CreatedAt: createdAt}
	if !createdAt.IsZero() && createdAt.After(startTime.Add(-30*time.Second)) {
		console.LogVerbose(verbose, fmt.Sprintf("Found recent run (ID: %d) created at %v (started polling at %v)", run.DatabaseID, createdAt.Format(time.RFC3339), startTime.Format(time.RFC3339)))
		stopWorkflowRunSpinner(spinner)
		return runInfo, true, nil
	}
	logWorkflowRunAttemptAge(run, createdAt, verbose, attempt, maxRetries)
	if attempt < 3 {
		return nil, false, nil
	}
	console.LogVerbose(verbose, fmt.Sprintf("Returning workflow run (ID: %d) after %d attempts (timing uncertain)", run.DatabaseID, attempt+1))
	stopWorkflowRunSpinner(spinner)
	return runInfo, true, nil
}

func parseWorkflowRunTime(createdAt string, verbose bool) time.Time {
	if createdAt == "" {
		return time.Time{}
	}
	parsedTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not parse creation time '%s': %v", createdAt, err)))
		}
		return time.Time{}
	}
	return parsedTime
}

func logWorkflowRunAttemptAge(run workflowRunListEntry, createdAt time.Time, verbose bool, attempt, maxRetries int) {
	if createdAt.IsZero() {
		console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: Found run (ID: %d) but no creation timestamp available", attempt+1, maxRetries, run.DatabaseID))
		return
	}
	console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: Found run (ID: %d) but it was created at %v (too old)", attempt+1, maxRetries, run.DatabaseID, createdAt.Format(time.RFC3339)))
}

func stopWorkflowRunSpinner(spinner *console.SpinnerWrapper) {
	if spinner != nil {
		spinner.StopWithMessage("✓ Found workflow run")
	}
}`}},
        "pkg/cli/run_workflow_validation.go": {
            {name: "getWorkflowInputs", body: `// getWorkflowInputs extracts workflow_dispatch inputs from the compiled lock file
// This function checks the .lock.yml file because that's what GitHub Actions uses.
func getWorkflowInputs(markdownPath string) (map[string]*workflow.InputDefinition, error) {
	workflowYAML, err := loadCompiledWorkflowYAML(markdownPath)
	if err != nil {
		return nil, err
	}
	inputsMap := getWorkflowDispatchInputsMap(workflowYAML)
	if len(inputsMap) == 0 {
		return nil, nil
	}
	parsed := workflow.ParseInputDefinitions(inputsMap)
	delete(parsed, workflow.AwContextInputName)
	return parsed, nil
}

func loadCompiledWorkflowYAML(markdownPath string) (map[string]any, error) {
	lockPath := getLockFilePath(markdownPath)
	cleanLockPath := filepath.Clean(lockPath)
	validationLog.Printf("Extracting workflow inputs from lock file: %s", lockPath)
	if _, err := os.Stat(cleanLockPath); os.IsNotExist(err) {
		validationLog.Printf("Lock file does not exist: %s", cleanLockPath)
		return nil, errors.New("workflow has not been compiled yet - run 'gh aw compile' first")
	}
	contentBytes, err := os.ReadFile(cleanLockPath) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}
	var workflowYAML map[string]any
	if err := yaml.Unmarshal(contentBytes, &workflowYAML); err != nil {
		return nil, fmt.Errorf("failed to parse lock file YAML: %w", err)
	}
	return workflowYAML, nil
}

func getWorkflowDispatchInputsMap(workflowYAML map[string]any) map[string]any {
	onMap, ok := workflowYAML["on"].(map[string]any)
	if !ok {
		return nil
	}
	workflowDispatchMap, ok := onMap["workflow_dispatch"].(map[string]any)
	if !ok {
		return nil
	}
	inputsMap, ok := workflowDispatchMap["inputs"].(map[string]any)
	if !ok {
		return nil
	}
	return inputsMap
}`},
            {name: "validateWorkflowInputs", body: `// validateWorkflowInputs validates that required inputs are provided and checks for typos.
//
// This validation function is co-located with the run command implementation because:
//   - It's specific to the workflow run operation
//   - It's only called during workflow dispatch
//   - It provides immediate feedback before triggering the workflow
//
// The function validates:
//   - All required inputs are provided
//   - Provided input names match defined inputs (typo detection)
//   - Suggestions for misspelled input names
//
// This follows the principle that domain-specific validation belongs in domain files.
func validateWorkflowInputs(markdownPath string, providedInputs []string) error {
	workflowInputs, err := getWorkflowInputs(markdownPath)
	if err != nil {
		validationLog.Printf("Failed to extract workflow inputs: %v", err)
		return nil
	}
	if len(workflowInputs) == 0 {
		return nil
	}
	providedInputsMap := parseProvidedWorkflowInputs(providedInputs)
	missingInputs := findMissingRequiredWorkflowInputs(workflowInputs, providedInputsMap)
	typos, suggestions := findWorkflowInputTypos(workflowInputs, providedInputsMap)
	if len(missingInputs) == 0 && len(typos) == 0 {
		return nil
	}
	workflowName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
	errorParts := buildWorkflowInputErrorParts(workflowInputs, missingInputs, typos, suggestions, workflowName)
	return workflow.NewValidationError("on.workflow_dispatch.inputs", strings.Join(providedInputs, ", "), strings.Join(errorParts, "\n\n"), fmt.Sprintf("Define and provide valid workflow_dispatch inputs.\n\nExample workflow frontmatter:\n\non:\n  workflow_dispatch:\n    inputs:\n      issue_url:\n        description: \"Issue URL\"\n        required: true\n        type: string\n\nExample command:\n  gh aw run %s -F issue_url=https://github.com/org/repo/issues/123", workflowName))
}

func parseProvidedWorkflowInputs(providedInputs []string) map[string]string {
	providedInputsMap := make(map[string]string)
	for _, input := range providedInputs {
		parts := strings.SplitN(input, "=", 2)
		if len(parts) == 2 {
			providedInputsMap[parts[0]] = parts[1]
		}
	}
	return providedInputsMap
}

func findMissingRequiredWorkflowInputs(workflowInputs map[string]*workflow.InputDefinition, providedInputsMap map[string]string) []string {
	var missingInputs []string
	for inputName, inputDef := range workflowInputs {
		if inputDef.Required && inputDef.Default == nil {
			if _, exists := providedInputsMap[inputName]; !exists {
				missingInputs = append(missingInputs, inputName)
			}
		}
	}
	return missingInputs
}

func findWorkflowInputTypos(workflowInputs map[string]*workflow.InputDefinition, providedInputsMap map[string]string) ([]string, []string) {
	validInputNames := slices.Collect(maps.Keys(workflowInputs))
	var typos []string
	var suggestions []string
	for providedName := range providedInputsMap {
		if _, exists := workflowInputs[providedName]; exists {
			continue
		}
		matches := parser.FindClosestMatches(providedName, validInputNames, 3)
		typos = append(typos, providedName)
		if len(matches) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("'%s' -> did you mean '%s'?", providedName, strings.Join(matches, "', '")))
			continue
		}
		suggestions = append(suggestions, fmt.Sprintf("'%s' is not a valid input name", providedName))
	}
	return typos, suggestions
}

func buildWorkflowInputErrorParts(workflowInputs map[string]*workflow.InputDefinition, missingInputs []string, typos []string, suggestions []string, workflowName string) []string {
	var errorParts []string
	if len(missingInputs) > 0 {
		errorParts = append(errorParts, "Missing required input(s): "+strings.Join(missingInputs, ", "))
	}
	if len(typos) > 0 {
		errorParts = append(errorParts, "Invalid input name(s):\n  "+strings.Join(suggestions, "\n  "))
	}
	if validInputsMsg := buildValidWorkflowInputsMessage(workflowInputs, workflowName); validInputsMsg != "" {
		errorParts = append(errorParts, validInputsMsg)
	}
	return errorParts
}

func buildValidWorkflowInputsMessage(workflowInputs map[string]*workflow.InputDefinition, workflowName string) string {
	if len(workflowInputs) == 0 {
		return ""
	}
	sortedNames := slices.Sorted(maps.Keys(workflowInputs))
	inputDescriptions := make([]string, 0, len(sortedNames))
	var syntaxExamples []string
	for _, name := range sortedNames {
		def := workflowInputs[name]
		required := ""
		if def.Required && def.Default == nil {
			required = " (required)"
			syntaxExamples = append(syntaxExamples, fmt.Sprintf("  gh aw run %s -F %s=<value>", workflowName, name))
		}
		desc := ""
		if def.Description != "" {
			desc = ": " + def.Description
		}
		defaultStr := ""
		if def.Default != nil {
			defaultStr = fmt.Sprintf(" [default: %s]", def.GetDefaultAsString())
		}
		inputDescriptions = append(inputDescriptions, fmt.Sprintf("  %s%s%s%s", name, required, desc, defaultStr))
	}
	validInputsMsg := "\nValid inputs:\n" + strings.Join(inputDescriptions, "\n")
	if len(syntaxExamples) > 0 {
		validInputsMsg += "\n\nTo set required inputs, use:\n" + strings.Join(syntaxExamples, "\n")
	}
	return validInputsMsg
}`},
        },
        "pkg/cli/runner_guard.go": {
            {name: "runRunnerGuardOnDirectory", body: `// runRunnerGuardOnDirectory runs the runner-guard taint analysis scanner on a directory
// containing workflows using the Docker image.
func runRunnerGuardOnDirectory(workflowDir string, verbose bool, strict bool) error {
	runnerGuardLog.Printf("Running runner-guard taint analysis on directory: %s", workflowDir)
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}
	if !filepath.IsAbs(gitRoot) {
		return fmt.Errorf("git root is not an absolute path: %s", gitRoot)
	}
	scanPath := runnerGuardScanPath(gitRoot, workflowDir)
	cmd := newRunnerGuardCommand(gitRoot, scanPath)
	showRunnerGuardCommandInfo(gitRoot, scanPath, verbose)
	stdout, stderr, err := executeRunnerGuardCommand(cmd)
	totalFindings, parseErr := parseAndDisplayRunnerGuardOutput(stdout, verbose, gitRoot)
	if parseErr != nil {
		runnerGuardLog.Printf("Failed to parse runner-guard output: %v", parseErr)
		printRawRunnerGuardOutput(stdout, stderr)
	}
	return handleRunnerGuardExit(err, totalFindings, parseErr, strict)
}

func runnerGuardScanPath(gitRoot, workflowDir string) string {
	scanPath := "."
	if workflowDir == "" {
		return scanPath
	}
	relDir, relErr := filepath.Rel(gitRoot, workflowDir)
	if relErr == nil && relDir != ".." && !strings.HasPrefix(relDir, ".."+string(filepath.Separator)) {
		scanPath = relDir
	}
	return scanPath
}

func newRunnerGuardCommand(gitRoot, scanPath string) *exec.Cmd {
	return exec.Command("docker", "run", "--rm", "-v", gitRoot+":/workdir", "-w", "/workdir", RunnerGuardImage, "scan", scanPath, "--format", "json")
}

func showRunnerGuardCommandInfo(gitRoot, scanPath string, verbose bool) {
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running runner-guard taint analysis scanner"))
	if !verbose {
		return
	}
	dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir %s scan %s --format json", gitRoot, RunnerGuardImage, scanPath)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run runner-guard directly: "+dockerCmd))
}

func executeRunnerGuardCommand(cmd *exec.Cmd) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func printRawRunnerGuardOutput(stdout, stderr string) {
	if stdout != "" {
		fmt.Fprint(os.Stderr, stdout)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
}

func handleRunnerGuardExit(err error, totalFindings int, parseErr error, strict bool) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("runner-guard failed: %w", err)
	}
	exitCode := exitErr.ExitCode()
	if exitCode != 1 {
		return fmt.Errorf("runner-guard failed with exit code %d", exitCode)
	}
	if !strict {
		return nil
	}
	if parseErr != nil {
		return fmt.Errorf("strict mode: runner-guard exited with code 1 (findings present) and output could not be parsed: %w", parseErr)
	}
	if totalFindings > 0 {
		return fmt.Errorf("strict mode: runner-guard found %d security findings - workflows must have no runner-guard findings in strict mode", totalFindings)
	}
	return errors.New("strict mode: runner-guard exited with code 1 indicating findings are present")
}`},
            {name: "parseAndDisplayRunnerGuardOutput", body: `// parseAndDisplayRunnerGuardOutput parses runner-guard JSON output and displays findings.
// Returns the total number of findings found.
func parseAndDisplayRunnerGuardOutput(stdout string, verbose bool, gitRoot string) (int, error) {
	_ = verbose
	output, err := parseRunnerGuardOutput(stdout)
	if err != nil || output == nil {
		return 0, err
	}
	totalFindings := len(output.Findings)
	if totalFindings == 0 {
		return 0, nil
	}
	displayRunnerGuardScore(output)
	findingsByFile := groupRunnerGuardFindingsByFile(output.Findings)
	for filePath, findings := range findingsByFile {
		if err := displayRunnerGuardFindingsForFile(filePath, findings, gitRoot); err != nil {
			runnerGuardLog.Printf("Skipping file %s: %v", filePath, err)
		}
	}
	return totalFindings, nil
}

func parseRunnerGuardOutput(stdout string) (*runnerGuardOutput, error) {
	if stdout == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(stdout)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		if trimmed == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected runner-guard output format: %s", trimmed)
	}
	var output runnerGuardOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		return nil, fmt.Errorf("failed to parse runner-guard JSON output: %w", err)
	}
	return &output, nil
}

func displayRunnerGuardScore(output *runnerGuardOutput) {
	if output.Score == 0 && output.Grade == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Runner-Guard Score: %d/100 (Grade: %s)", output.Score, output.Grade)))
}

func groupRunnerGuardFindingsByFile(findings []runnerGuardFinding) map[string][]runnerGuardFinding {
	findingsByFile := make(map[string][]runnerGuardFinding)
	for _, finding := range findings {
		findingsByFile[finding.File] = append(findingsByFile[finding.File], finding)
	}
	return findingsByFile
}

func displayRunnerGuardFindingsForFile(filePath string, findings []runnerGuardFinding, gitRoot string) error {
	absPath, err := validatedRunnerGuardFilePath(filePath, gitRoot)
	if err != nil {
		return err
	}
	fileLines := readRunnerGuardFileLines(absPath)
	for _, finding := range findings {
		lineNum := finding.Line
		if lineNum == 0 {
			lineNum = 1
		}
		compilerErr := console.CompilerError{Position: console.ErrorPosition{File: finding.File, Line: lineNum, Column: 1}, Type: runnerGuardSeverityType(finding.Severity), Message: runnerGuardFindingMessage(finding), Context: runnerGuardFindingContext(fileLines, lineNum)}
		fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
	}
	return nil
}

func validatedRunnerGuardFilePath(filePath, gitRoot string) (string, error) {
	cleanPath := filepath.Clean(filePath)
	absPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(gitRoot, cleanPath)
	}
	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		return "", err
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(absGitRoot, absPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file outside git root: %s", filePath)
	}
	return absPath, nil
}

func readRunnerGuardFileLines(absPath string) []string {
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	return strings.Split(string(fileContent), "\n")
}

func runnerGuardSeverityType(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "high", "error":
		return "error"
	case "note", "info":
		return "info"
	default:
		return "warning"
	}
}

func runnerGuardFindingMessage(finding runnerGuardFinding) string {
	message := fmt.Sprintf("[%s] %s: %s", finding.Severity, finding.RuleID, finding.Name)
	if finding.Description != "" {
		message = fmt.Sprintf("%s - %s", message, finding.Description)
	}
	return message
}

func runnerGuardFindingContext(fileLines []string, lineNum int) []string {
	if len(fileLines) == 0 || lineNum <= 0 || lineNum > len(fileLines) {
		return nil
	}
	startLine := max(1, lineNum-2)
	endLine := min(len(fileLines), lineNum+2)
	context := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		context = append(context, fileLines[i-1])
	}
	return context
}`}},
    }

    for path, repls := range replacements {
        if err := apply(path, repls); err != nil {
            fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
            os.Exit(1)
        }
    }
}

func apply(path string, repls []replacement) error {
    data, err := os.ReadFile(path)
    if err != nil { return err }
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
    if err != nil { return err }
    type edit struct{ start, end int; text string }
    var edits []edit
    for _, decl := range file.Decls {
        fn, ok := decl.(*ast.FuncDecl)
        if !ok { continue }
        for _, r := range repls {
            if fn.Name.Name == r.name {
                edits = append(edits, edit{start: fset.Position(fn.Pos()).Offset, end: fset.Position(fn.End()).Offset, text: r.body})
            }
        }
    }
    sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
    out := string(data)
    for _, e := range edits { out = out[:e.start] + e.text + out[e.end:] }
    return os.WriteFile(path, []byte(out), 0o644)
}
