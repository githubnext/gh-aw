package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// NewTestCommand creates the test command for local workflow execution with act
func NewTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <workflow-id-or-name>",
		Short: "Test an agentic workflow locally using Docker and act",
		Long: `Test an agentic workflow locally using Docker and the nektos/act tool.

This command compiles the workflow and runs it locally in Docker containers instead of GitHub Actions.
It automatically detects and installs the 'act' tool if not available.

The workflow must have been added as an action and compiled.
This command works with workflows that have workflow_dispatch, push, pull_request, or other triggers.

Examples:
  gh aw test weekly-research
  gh aw test weekly-research --event workflow_dispatch`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			event, _ := cmd.Flags().GetString("event")
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			if err := TestWorkflowLocally(args[0], event, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
					Type:    "error",
					Message: fmt.Sprintf("testing workflow locally: %v", err),
				}))
				os.Exit(1)
			}
		},
	}

	// Add flags to test command
	cmd.Flags().StringP("event", "e", "workflow_dispatch", "Event type to simulate (workflow_dispatch, push, pull_request, etc.)")

	return cmd
}

// TestWorkflowLocally runs a single agentic workflow locally using Docker and act
func TestWorkflowLocally(workflowName, event string, verbose bool) error {
	if workflowName == "" {
		return fmt.Errorf("workflow name or ID is required")
	}

	// Check and install act if needed
	if err := checkActInstalled(verbose); err != nil {
		return fmt.Errorf("failed to ensure act is available: %w", err)
	}

	// Compile workflow first to ensure it's up to date (with safe-outputs staged if any)
	fmt.Println(console.FormatProgressMessage("Compiling workflow before local testing..."))
	if err := CompileWorkflowForTesting(workflowName, verbose); err != nil {
		return fmt.Errorf("failed to compile workflow: %w", err)
	}

	fmt.Println(console.FormatSuccessMessage("Workflow compiled successfully"))

	if err := testSingleWorkflowLocally(workflowName, event, verbose); err != nil {
		return fmt.Errorf("failed to test workflow '%s': %w", workflowName, err)
	}

	fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Successfully tested workflow: %s", workflowName)))
	return nil
}

// testSingleWorkflowLocally tests a single workflow locally using act
func testSingleWorkflowLocally(workflowName, event string, verbose bool) error {
	// Resolve workflow file
	workflowFile, err := resolveWorkflowFile(workflowName, verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve workflow file: %w", err)
	}

	// Check if it's a lock file or markdown file
	lockFile := workflowFile
	if !strings.HasSuffix(workflowFile, ".lock.yml") {
		// Find corresponding lock file
		baseName := strings.TrimSuffix(filepath.Base(workflowFile), ".md")
		lockFile = filepath.Join(getWorkflowsDir(), baseName+".lock.yml")

		// Check if lock file exists
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			return fmt.Errorf("compiled workflow file not found: %s. Run 'gh aw compile %s' first", lockFile, workflowName)
		}
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Testing workflow file: %s", lockFile)))
	}

	// Build act command
	args := []string{"act"}

	// Add event type
	if event != "" {
		args = append(args, event)
	}

	// Add workflow file
	args = append(args, "--workflows", lockFile)

	// Add verbose flag if requested
	if verbose {
		args = append(args, "--verbose")
	}

	if verbose {
		fmt.Println(console.FormatCommandMessage(fmt.Sprintf("gh %s", strings.Join(args, " "))))
	}

	// Execute act command via GitHub CLI
	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("act execution failed: %w", err)
	}

	return nil
}

// checkActInstalled checks if the act tool is available and installs it if needed
func checkActInstalled(verbose bool) error {
	// Check if act is already installed
	if _, err := exec.LookPath("act"); err == nil {
		if verbose {
			fmt.Println(console.FormatInfoMessage("act tool is already installed"))
		}
		return nil
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage("act tool not found, attempting to install via GitHub CLI extension"))
	}

	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) not found. Please install GitHub CLI first: https://cli.github.com/")
	}

	// Install act via GitHub CLI extension
	fmt.Println(console.FormatProgressMessage("Installing act via GitHub CLI extension..."))
	cmd := exec.Command("gh", "extension", "install", "https://github.com/nektos/act")

	// Capture output to provide better error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "GH_TOKEN") {
			return fmt.Errorf("GitHub CLI authentication required. Please run 'gh auth login' first, or install act manually: https://nektosact.com")
		}
		return fmt.Errorf("failed to install act extension: %w. Please install manually: https://nektosact.com", err)
	}

	fmt.Println(console.FormatSuccessMessage("Successfully installed act via GitHub CLI extension"))
	return nil
}

// CompileWorkflowForTesting compiles a single workflow with safe-outputs forced to staged mode
func CompileWorkflowForTesting(workflowName string, verbose bool) error {
	// Create compiler with forced staged mode for testing
	compiler := workflow.NewCompiler(verbose, "", GetVersion())
	compiler.SetSkipValidation(false) // Enable validation for testing
	compiler.SetForceStaged(true)     // Force safe-outputs to be staged for local testing

	// Resolve workflow file
	workflowFile, err := resolveWorkflowFile(workflowName, verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve workflow file: %w", err)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Compiling workflow: %s", workflowFile)))
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to compile workflow '%s': %w", workflowFile, err)
	}

	return nil
}