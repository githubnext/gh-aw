package cli

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var addWorkflowPRLog = logger.New("cli:add_workflow_pr")

// invalidBranchCharsPattern matches characters not allowed in git branch names
var invalidBranchCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// consecutiveHyphensPattern matches two or more consecutive hyphens
var consecutiveHyphensPattern = regexp.MustCompile(`-{2,}`)

// sanitizeBranchName sanitizes a string for use in a git branch name.
// Git branch names cannot contain:
// - spaces, ~, ^, :, \, ?, *, [, @{
// - consecutive dots (..)
// - leading/trailing dots or slashes
// - control characters
func sanitizeBranchName(name string) string {
	// Use base name only (no directory path)
	name = normalizeWorkflowID(name)

	// Replace problematic characters with hyphens
	// This regex matches any character that's not alphanumeric, hyphen, or underscore
	name = invalidBranchCharsPattern.ReplaceAllString(name, "-")

	// Remove consecutive hyphens
	name = consecutiveHyphensPattern.ReplaceAllString(name, "-")

	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Ensure non-empty (fallback to "workflow")
	if name == "" {
		name = "workflow"
	}

	return name
}

// addWorkflowsWithPR handles workflow addition with PR creation using pre-resolved workflows.
func addWorkflowsWithPR(ctx context.Context, workflows []*ResolvedWorkflow, opts AddOptions) (int, string, error) {
	addWorkflowPRLog.Printf("Adding %d workflow(s) with PR creation (resolved)", len(workflows))
	currentBranch, branchName, tracker, err := prepareWorkflowPRBranch(workflows, opts)
	if err != nil {
		return 0, "", err
	}
	defer restoreWorkflowPRBranch(currentBranch, opts.Verbose)
	if err := stageWorkflowsForPR(ctx, workflows, tracker, opts); err != nil {
		return 0, "", err
	}
	commitMessage, prTitle, prBody := workflowPRMessages(workflows)
	if err := commitWorkflowPRChanges(commitMessage, branchName, prTitle, opts.Verbose); err != nil {
		return 0, "", err
	}
	if err := pushWorkflowPRBranch(branchName, prTitle, opts.Verbose); err != nil {
		return 0, "", err
	}
	return createWorkflowPullRequest(ctx, tracker, currentBranch, branchName, prTitle, prBody, opts.Verbose)
}

func prepareWorkflowPRBranch(workflows []*ResolvedWorkflow, opts AddOptions) (string, string, *FileTracker, error) {
	currentBranch, err := getCurrentBranch()
	if err != nil {
		addWorkflowPRLog.Printf("Failed to get current branch: %v", err)
		return "", "", nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	branchName := fmt.Sprintf("add-workflow-%s-%04d", sanitizeBranchName(workflows[0].Spec.WorkflowPath), rand.Intn(9000)+1000)
	addWorkflowPRLog.Printf("Creating temporary branch: %s", branchName)
	if err := createAndSwitchBranch(branchName, opts.Verbose); err != nil {
		return "", "", nil, fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}
	return currentBranch, branchName, NewFileTracker(), nil
}

func restoreWorkflowPRBranch(currentBranch string, verbose bool) {
	if switchErr := switchBranch(currentBranch, verbose); switchErr != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to switch back to branch %s: %v", currentBranch, switchErr)))
	}
}

func stageWorkflowsForPR(ctx context.Context, workflows []*ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
	addWorkflowPRLog.Print("Adding workflows to repository")
	if err := addWorkflowsWithTracking(ctx, workflows, tracker, opts); err != nil {
		addWorkflowPRLog.Printf("Failed to add workflows: %v", err)
		return fmt.Errorf("failed to add workflows: %w", err)
	}
	addWorkflowPRLog.Print("Staging workflow files")
	if err := tracker.StageAllFiles(opts.Verbose); err != nil {
		rollbackWorkflowPRFiles(tracker, opts.Verbose)
		return fmt.Errorf("failed to stage workflow files: %w", err)
	}
	if err := stageGitAttributesIfChanged(); err != nil && opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to stage .gitattributes: %v", err)))
	}
	return nil
}

func rollbackWorkflowPRFiles(tracker *FileTracker, verbose bool) {
	if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
	}
}

func workflowPRMessages(workflows []*ResolvedWorkflow) (string, string, string) {
	if len(workflows) == 1 {
		message := "Add agentic workflow " + workflows[0].Spec.WorkflowName
		return message, message, message
	}
	workflowNames := sliceutil.Map(workflows, func(wf *ResolvedWorkflow) string { return wf.Spec.WorkflowName })
	message := "Add agentic workflows: " + strings.Join(workflowNames, ", ")
	return message, message, message
}

func commitWorkflowPRChanges(commitMessage, branchName, prTitle string, verbose bool) error {
	if err := commitChanges(commitMessage, verbose); err != nil {
		return fmt.Errorf("failed to commit workflow files: %w\n\nThe workflow files have been written to disk and staged in git.\nPlease commit the files manually, then either push them to the\nrepository or create a pull request:\n\n  git commit -m %q\n  git push\n\nOr to create a pull request:\n\n  git checkout -b %s\n  git commit -m %q\n  git push -u origin %s\n  gh pr create --title %q", err, commitMessage, branchName, commitMessage, branchName, prTitle)
	}
	return nil
}

func pushWorkflowPRBranch(branchName, prTitle string, verbose bool) error {
	addWorkflowPRLog.Printf("Pushing branch %s to remote", branchName)
	if err := pushBranch(branchName, verbose); err != nil {
		addWorkflowPRLog.Printf("Failed to push branch: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to push branch %s: %v", branchName, err)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			"The workflow files have been committed to local branch "+branchName+".\n"+
				"  To push the branch and create a pull request, run:\n\n"+
				"    git push -u origin "+branchName+"\n"+
				"    gh pr create --title "+fmt.Sprintf("%q", prTitle),
		))
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}
	return nil
}

func createWorkflowPullRequest(ctx context.Context, tracker *FileTracker, currentBranch, branchName, prTitle, prBody string, verbose bool) (int, string, error) {
	addWorkflowPRLog.Printf("Creating pull request: %s", prTitle)
	prNumber, prURL, err := createPR(ctx, branchName, prTitle, prBody, verbose)
	if err != nil {
		addWorkflowPRLog.Printf("Failed to create PR: %v", err)
		rollbackWorkflowPRFiles(tracker, verbose)
		return 0, "", fmt.Errorf("failed to create PR: %w", err)
	}
	addWorkflowPRLog.Printf("Successfully created PR #%d: %s", prNumber, prURL)
	if err := switchBranch(currentBranch, verbose); err != nil {
		return prNumber, prURL, fmt.Errorf("failed to switch back to branch %s: %w", currentBranch, err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created pull request "+prURL))
	return prNumber, prURL, nil
}
