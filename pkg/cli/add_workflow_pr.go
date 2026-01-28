package cli

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var addPRLog = logger.New("cli:add_workflow_pr")

// addWorkflowsWithPR handles workflow addition with PR creation and returns the PR number and URL.
func addWorkflowsWithPR(workflows []*WorkflowSpec, number int, verbose bool, quiet bool, engineOverride string, name string, force bool, appendText string, push bool, noGitattributes bool, fromWildcard bool, workflowDir string, noStopAfter bool, stopAfter string) (int, string, error) {
	// Get current branch for restoration later
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create temporary branch with random 4-digit number
	randomNum := rand.Intn(9000) + 1000 // Generate number between 1000-9999
	branchName := fmt.Sprintf("add-workflow-%s-%04d", strings.ReplaceAll(workflows[0].WorkflowPath, "/", "-"), randomNum)

	if err := createAndSwitchBranch(branchName, verbose); err != nil {
		return 0, "", fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Create file tracker for rollback capability
	tracker, err := NewFileTracker()
	if err != nil {
		return 0, "", fmt.Errorf("failed to create file tracker: %w", err)
	}

	// Ensure we switch back to original branch on exit
	defer func() {
		if switchErr := switchBranch(currentBranch, verbose); switchErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to switch back to branch %s: %v", currentBranch, switchErr)))
		}
	}()

	// Add workflows using the normal function logic
	if err := addWorkflowsNormal(workflows, number, verbose, quiet, engineOverride, name, force, appendText, push, noGitattributes, fromWildcard, workflowDir, noStopAfter, stopAfter); err != nil {
		// Rollback on error
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return 0, "", fmt.Errorf("failed to add workflows: %w", err)
	}

	// Stage all files before creating PR
	if err := tracker.StageAllFiles(verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return 0, "", fmt.Errorf("failed to stage workflow files: %w", err)
	}

	// Update .gitattributes and stage it if modified
	if err := stageGitAttributesIfChanged(); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to stage .gitattributes: %v", err)))
	}

	// Commit changes
	var commitMessage, prTitle, prBody, joinedNames string
	if len(workflows) == 1 {
		joinedNames = workflows[0].WorkflowName
		commitMessage = fmt.Sprintf("Add agentic workflow %s", joinedNames)
		prTitle = fmt.Sprintf("Add agentic workflow %s", joinedNames)
		prBody = fmt.Sprintf("Add agentic workflow %s", joinedNames)
	} else {
		// Get workflow.Workflo
		workflowNames := make([]string, len(workflows))
		for i, wf := range workflows {
			workflowNames[i] = wf.WorkflowName
		}
		joinedNames = strings.Join(workflowNames, ", ")
		commitMessage = fmt.Sprintf("Add agentic workflows: %s", joinedNames)
		prTitle = fmt.Sprintf("Add agentic workflows: %s", joinedNames)
		prBody = fmt.Sprintf("Add agentic workflows: %s", joinedNames)
	}

	if err := commitChanges(commitMessage, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return 0, "", fmt.Errorf("failed to commit files: %w", err)
	}

	// Push branch
	if err := pushBranch(branchName, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return 0, "", fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	// Create PR
	prNumber, prURL, err := createPR(branchName, prTitle, prBody, verbose)
	if err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return 0, "", fmt.Errorf("failed to create PR: %w", err)
	}

	// Success - no rollback needed

	// Switch back to original branch
	if err := switchBranch(currentBranch, verbose); err != nil {
		return prNumber, prURL, fmt.Errorf("failed to switch back to branch %s: %w", currentBranch, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created pull request %s", prURL)))
	return prNumber, prURL, nil
}
