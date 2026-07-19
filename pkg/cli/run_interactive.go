package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/tty"
	"github.com/github/gh-aw/pkg/workflow"
)

var runInteractiveLog = logger.New("cli:run_interactive")

// WorkflowOption represents a workflow that can be run
type WorkflowOption struct {
	Name        string
	Description string
	FilePath    string
	Inputs      map[string]*workflow.InputDefinition
}

// RunWorkflowInteractively runs a workflow in interactive mode
func RunWorkflowInteractively(ctx context.Context, verbose bool, repoOverride string, refOverride string, autoMergePRs bool, push bool, engineOverride string, dryRun bool) error {
	runInteractiveLog.Print("Starting interactive workflow run")
	if IsRunningInCI() {
		return errors.New("interactive mode cannot be used in CI environments")
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting interactive workflow run..."))
	}

	selectedWorkflow, inputValues, err := runWorkflowInteractivelySelect(ctx, verbose)
	if err != nil {
		return err
	}
	if !confirmExecution(ctx, selectedWorkflow, inputValues) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow execution cancelled"))
		return nil
	}

	cmdStr := buildCommandString(selectedWorkflow.Name, inputValues, repoOverride, refOverride, autoMergePRs, push, engineOverride)
	runWorkflowInteractivelyPrintCommand(cmdStr)
	if err := runWorkflowInteractivelyExecute(runWorkflowInteractivelyExecuteParams{
		Ctx:            ctx,
		WorkflowName:   selectedWorkflow.Name,
		InputValues:    inputValues,
		Verbose:        verbose,
		RepoOverride:   repoOverride,
		RefOverride:    refOverride,
		AutoMergePRs:   autoMergePRs,
		Push:           push,
		EngineOverride: engineOverride,
		DryRun:         dryRun,
	}); err != nil {
		return err
	}
	runWorkflowInteractivelyPrintSuccess(cmdStr)
	return nil
}

func runWorkflowInteractivelySelect(ctx context.Context, verbose bool) (*WorkflowOption, []string, error) {
	workflows, err := findRunnableWorkflows(verbose)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find runnable workflows: %w", err)
	}
	if len(workflows) == 0 {
		return nil, nil, errors.New("no runnable workflows found. Workflows must have 'workflow_dispatch' trigger")
	}
	selectedWorkflow, err := selectWorkflow(ctx, workflows)
	if err != nil {
		return nil, nil, fmt.Errorf("workflow selection cancelled or failed: %w", err)
	}
	runInteractiveLog.Printf("Selected workflow: %s", selectedWorkflow.Name)
	showWorkflowInfo(selectedWorkflow)
	inputValues, err := collectWorkflowInputs(ctx, selectedWorkflow)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect workflow inputs: %w", err)
	}
	return selectedWorkflow, inputValues, nil
}

func runWorkflowInteractivelyPrintCommand(cmdStr string) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nRunning workflow..."))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage("Equivalent command: "+cmdStr))
	fmt.Fprintln(os.Stderr, "")
}

type runWorkflowInteractivelyExecuteParams struct {
	Ctx            context.Context
	WorkflowName   string
	InputValues    []string
	Verbose        bool
	RepoOverride   string
	RefOverride    string
	AutoMergePRs   bool
	Push           bool
	EngineOverride string
	DryRun         bool
}

func runWorkflowInteractivelyExecute(p runWorkflowInteractivelyExecuteParams) error {
	err := RunWorkflowOnGitHub(p.Ctx, p.WorkflowName, RunOptions{
		Enable:         false,
		EngineOverride: p.EngineOverride,
		RepoOverride:   p.RepoOverride,
		RefOverride:    p.RefOverride,
		AutoMergePRs:   p.AutoMergePRs,
		Push:           p.Push,
		Inputs:         p.InputValues,
		Verbose:        p.Verbose,
		DryRun:         p.DryRun,
	})
	if err != nil {
		return fmt.Errorf("failed to run workflow: %w", err)
	}
	return nil
}

func runWorkflowInteractivelyPrintSuccess(cmdStr string) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Workflow dispatched successfully!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To run this workflow again, use:"))
	fmt.Fprintln(os.Stderr, console.FormatCommandMessage(cmdStr))
}

// findRunnableWorkflows finds all workflows that support workflow_dispatch
func findRunnableWorkflows(verbose bool) ([]WorkflowOption, error) {
	runInteractiveLog.Print("Finding runnable workflows")

	// Get all markdown workflow files
	workflowsDir := constants.GetWorkflowDir()
	mdFiles, err := getMarkdownWorkflowFiles(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow files: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d workflow files, checking for workflow_dispatch trigger...\n", len(mdFiles))
	}

	var runnableWorkflows []WorkflowOption

	for _, mdFile := range mdFiles {
		// Check if workflow is runnable
		runnable, err := IsRunnable(mdFile)
		if err != nil {
			runInteractiveLog.Printf("Failed to check if workflow %s is runnable: %v", mdFile, err)
			continue
		}

		if !runnable {
			continue
		}

		// Extract workflow name
		name := normalizeWorkflowID(mdFile)

		// Get workflow inputs
		inputs, err := getWorkflowInputs(mdFile)
		if err != nil {
			runInteractiveLog.Printf("Failed to get inputs for workflow %s: %v", mdFile, err)
			// Continue without inputs
			inputs = nil
		}

		// Build description
		description := buildWorkflowDescription(inputs)

		runnableWorkflows = append(runnableWorkflows, WorkflowOption{
			Name:        name,
			Description: description,
			FilePath:    mdFile,
			Inputs:      inputs,
		})
	}

	runInteractiveLog.Printf("Found %d runnable workflows", len(runnableWorkflows))
	return runnableWorkflows, nil
}

// buildWorkflowDescription creates a description string for a workflow
func buildWorkflowDescription(inputs map[string]*workflow.InputDefinition) string {
	// Always return empty string to avoid showing input counts
	return ""
}

// selectWorkflow displays an interactive list for workflow selection with fuzzy search
func selectWorkflow(ctx context.Context, workflows []WorkflowOption) (*WorkflowOption, error) {
	runInteractiveLog.Printf("Displaying workflow selection: %d workflows", len(workflows))

	// Check if we're in a TTY environment
	if !tty.IsStderrTerminal() {
		return selectWorkflowNonInteractive(workflows)
	}

	// Build select options
	options := sliceutil.Map(workflows, func(wf WorkflowOption) huh.Option[string] { return huh.NewOption(wf.Name, wf.Name) })

	var selected string
	form := console.NewSelectForm(
		huh.NewSelect[string]().
			Title("Select a workflow to run").
			Description("↑/↓ to navigate, / to search, Enter to select").
			Options(options...).
			Filtering(true).
			Height(15).
			Value(&selected),
	)

	if err := form.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("workflow selection cancelled or failed: %w", err)
	}

	// Find the selected workflow
	for i := range workflows {
		if workflows[i].Name == selected {
			return &workflows[i], nil
		}
	}

	return nil, fmt.Errorf("selected workflow not found: %s", selected)
}

// selectWorkflowNonInteractive provides a fallback for non-TTY environments
func selectWorkflowNonInteractive(workflows []WorkflowOption) (*WorkflowOption, error) {
	runInteractiveLog.Printf("Non-TTY detected, showing text list: %d workflows", len(workflows))

	fmt.Fprintf(os.Stderr, "\nSelect a workflow to run:\n\n")
	for i, wf := range workflows {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, wf.Name)
	}
	fmt.Fprintf(os.Stderr, "\nSelect (1-%d): ", len(workflows))

	var choice int
	_, err := fmt.Scanf("%d", &choice)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if choice < 1 || choice > len(workflows) {
		return nil, fmt.Errorf("selection out of range (must be 1-%d)", len(workflows))
	}

	selectedWorkflow := &workflows[choice-1]
	runInteractiveLog.Printf("Selected workflow from text list: %s", selectedWorkflow.Name)
	return selectedWorkflow, nil
}

// showWorkflowInfo displays information about the selected workflow
func showWorkflowInfo(wf *WorkflowOption) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflow: "+wf.Name))

	if len(wf.Inputs) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nWorkflow Inputs:"))
		for name, input := range wf.Inputs {
			required := ""
			if input.Required {
				required = " (required)"
			}
			desc := ""
			if input.Description != "" {
				desc = " - " + input.Description
			}
			defaultVal := ""
			if input.Default != "" {
				defaultVal = fmt.Sprintf(" [default: %s]", input.Default)
			}
			fmt.Fprintf(os.Stderr, "  • %s%s%s%s\n", name, required, desc, defaultVal)
		}
	}
	fmt.Fprintln(os.Stderr, "")
}

// collectWorkflowInputs collects input values from the user
func collectWorkflowInputs(ctx context.Context, wf *WorkflowOption) ([]string, error) {
	if len(wf.Inputs) == 0 {
		return nil, nil
	}

	runInteractiveLog.Printf("Collecting %d workflow inputs", len(wf.Inputs))
	return collectInputsWithMap(ctx, wf.Inputs)
}

// collectInputsWithMap collects inputs using a map to properly capture values
func collectInputsWithMap(ctx context.Context, inputs map[string]*workflow.InputDefinition) ([]string, error) {
	inputPtrs := make(map[string]*string)
	var formGroups []*huh.Group
	for name, input := range inputs {
		group := collectInputsWithMapGroup(name, input, inputPtrs)
		formGroups = append(formGroups, group)
	}

	form := console.NewForm(formGroups...)
	if err := form.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("input collection cancelled: %w", err)
	}

	var result []string
	for name, valuePtr := range inputPtrs {
		value := *valuePtr
		if value != "" {
			result = append(result, fmt.Sprintf("%s=%s", name, value))
		}
	}
	runInteractiveLog.Printf("Collected %d input values", len(result))
	return result, nil
}

func collectInputsWithMapGroup(name string, input *workflow.InputDefinition, inputPtrs map[string]*string) *huh.Group {
	inputName := name
	inputDef := input
	defaultStr := ""
	if inputDef.Default != nil {
		defaultStr = fmt.Sprintf("%v", inputDef.Default)
	}
	valueStr := defaultStr
	inputPtrs[inputName] = &valueStr
	field := huh.NewInput().Title(fmt.Sprintf("Enter value for '%s'", inputName)).Value(inputPtrs[inputName])
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
	return huh.NewGroup(field)
}

// confirmExecution asks the user to confirm workflow execution
func confirmExecution(ctx context.Context, wf *WorkflowOption, inputs []string) bool {
	runInteractiveLog.Print("Requesting execution confirmation")

	var confirm bool
	message := fmt.Sprintf("Run workflow '%s'?", wf.Name)

	if len(inputs) > 0 {
		message = fmt.Sprintf("Run workflow '%s' with %d input(s)?", wf.Name, len(inputs))
	}

	form := console.NewConfirmForm(
		huh.NewConfirm().
			Title(message).
			Affirmative("Yes, run it").
			Negative("No, cancel").
			Value(&confirm),
	)

	if err := form.RunWithContext(ctx); err != nil {
		if console.IsCancelled(err) {
			runInteractiveLog.Print("User aborted confirmation")
		} else {
			runInteractiveLog.Printf("Confirmation failed: %v", err)
		}
		return false
	}

	runInteractiveLog.Printf("User confirmed: %v", confirm)
	return confirm
}

// RunWorkflowOptions holds parameters for RunSpecificWorkflowInteractively.
type RunWorkflowOptions struct {
	WorkflowName   string
	Verbose        bool
	EngineOverride string
	RepoOverride   string
	RefOverride    string
	AutoMergePRs   bool
	Push           bool
	DryRun         bool
}

// RunSpecificWorkflowInteractively runs a specific workflow in interactive mode
// This is similar to RunWorkflowInteractively but skips the workflow selection step
// since the workflow name is already known. It will still collect inputs if the workflow has them.
func RunSpecificWorkflowInteractively(ctx context.Context, opts RunWorkflowOptions) error {
	runInteractiveLog.Printf("Running specific workflow interactively: %s", opts.WorkflowName)
	wf, err := runSpecificWorkflowInteractivelyOption(opts.WorkflowName)
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
	runWorkflowInteractivelyPrintCommand(cmdStr)
	if err := runSpecificWorkflowInteractivelyExecute(ctx, opts, inputValues); err != nil {
		return err
	}
	return nil
}

func runSpecificWorkflowInteractivelyOption(workflowName string) (*WorkflowOption, error) {
	workflowsDir := constants.GetWorkflowDir()
	mdFile := filepath.Join(workflowsDir, workflowName+".md")
	if _, err := os.Stat(mdFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflow file not found: %s", mdFile)
	}
	inputs, err := getWorkflowInputs(mdFile)
	if err != nil {
		runInteractiveLog.Printf("Failed to get inputs for workflow %s: %v", workflowName, err)
		inputs = nil
	}
	return &WorkflowOption{Name: workflowName, Description: buildWorkflowDescription(inputs), FilePath: mdFile, Inputs: inputs}, nil
}

func runSpecificWorkflowInteractivelyExecute(ctx context.Context, opts RunWorkflowOptions, inputValues []string) error {
	err := RunWorkflowOnGitHub(ctx, opts.WorkflowName, RunOptions{
		Enable:            false,
		EngineOverride:    opts.EngineOverride,
		RepoOverride:      opts.RepoOverride,
		RefOverride:       opts.RefOverride,
		AutoMergePRs:      opts.AutoMergePRs,
		Push:              opts.Push,
		WaitForCompletion: true,
		Inputs:            inputValues,
		Verbose:           opts.Verbose,
		DryRun:            opts.DryRun,
	})
	if err != nil {
		return fmt.Errorf("failed to run workflow: %w", err)
	}
	return nil
}

// buildCommandString builds the equivalent command string for display
func buildCommandString(workflowName string, inputs []string, repoOverride, refOverride string, autoMergePRs, push bool, engineOverride string) string {
	parts := []string{string(constants.CLIExtensionPrefix), "run", workflowName}

	// Add inputs
	for _, input := range inputs {
		parts = append(parts, "-F", input)
	}

	// Add flags
	if repoOverride != "" {
		parts = append(parts, "--repo", repoOverride)
	}
	if refOverride != "" {
		parts = append(parts, "--ref", refOverride)
	}
	if autoMergePRs {
		parts = append(parts, "--auto-merge-prs")
	}
	if push {
		parts = append(parts, "--push")
	}
	if engineOverride != "" {
		parts = append(parts, "--engine", engineOverride)
	}

	return strings.Join(parts, " ")
}
