package cli

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// NewUpdateCommand creates the update command
func NewUpdateCommand(verbosePtr *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [workflow]...",
		Short: "Update workflows that have source information",
		Long: `Update one or more workflows from their source repositories.

Only workflows with a 'source' field in their frontmatter can be updated.
The command will fetch the latest version from the source and merge changes
while preserving local modifications.

Examples:
  ` + constants.CLIExtensionPrefix + ` update                 # Update all workflows with source
  ` + constants.CLIExtensionPrefix + ` update weekly-research # Update specific workflow
  ` + constants.CLIExtensionPrefix + ` update --pr            # Create PR with updates`,
		Run: func(cmd *cobra.Command, args []string) {
			verbose := false
			if verbosePtr != nil {
				verbose = *verbosePtr
			}
			prFlag, _ := cmd.Flags().GetBool("pr")
			if err := UpdateWorkflows(args, verbose, prFlag); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().Bool("pr", false, "Create a pull request with the workflow changes")

	return cmd
}

// UpdateWorkflows updates workflows from their source repositories
func UpdateWorkflows(workflows []string, verbose bool, createPR bool) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("update requires being in a git repository: %w", err)
	}

	githubWorkflowsDir := filepath.Join(gitRoot, ".github/workflows")
	if _, err := os.Stat(githubWorkflowsDir); os.IsNotExist(err) {
		return fmt.Errorf(".github/workflows directory not found")
	}

	// Find all workflows with source field
	workflowsToUpdate, err := findWorkflowsWithSource(githubWorkflowsDir, workflows, verbose)
	if err != nil {
		return err
	}

	if len(workflowsToUpdate) == 0 {
		if len(workflows) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflows with source field found."))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflows need a 'source' field in their frontmatter to be updated."))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No matching workflows with source field found."))
		}
		return nil
	}

	// Handle PR creation workflow
	if createPR {
		return updateWorkflowsWithPR(workflowsToUpdate, verbose)
	}

	// Handle normal update without PR
	return updateWorkflowsNormal(workflowsToUpdate, verbose)
}

// updateWorkflowsNormal handles normal workflow update without PR creation
func updateWorkflowsNormal(workflowsToUpdate []WorkflowWithSource, verbose bool) error {
	// Create file tracker for all operations
	tracker, err := NewFileTracker()
	if err != nil {
		// If we can't create a tracker (e.g., not in git repo), fall back to non-tracking behavior
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not create file tracker: %v", err)))
		}
		tracker = nil
	}

	// Update each workflow
	updated := 0
	failed := 0
	for _, wf := range workflowsToUpdate {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Updating workflow: %s", wf.Name)))
		}

		if err := updateSingleWorkflow(wf, verbose, tracker); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to update %s: %v", wf.Name, err)))
			failed++
		} else {
			updated++
		}
	}

	// Report results
	if updated > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully updated %d workflow(s)", updated)))
	}
	if failed > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update %d workflow(s)", failed)))
	}

	if failed > 0 {
		return fmt.Errorf("some workflows failed to update")
	}

	return nil
}

// updateWorkflowsWithPR handles workflow update with PR creation
func updateWorkflowsWithPR(workflowsToUpdate []WorkflowWithSource, verbose bool) error {
	// Get current branch for restoration later
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create temporary branch
	branchName := fmt.Sprintf("update-workflows-%04d", rand.Intn(9000)+1000)

	if err := createAndSwitchBranch(branchName, verbose); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Create file tracker for rollback capability
	tracker, err := NewFileTracker()
	if err != nil {
		return fmt.Errorf("failed to create file tracker: %w", err)
	}

	// Ensure we switch back to original branch on exit
	defer func() {
		if switchErr := switchBranch(currentBranch, verbose); switchErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to switch back to branch %s: %v", currentBranch, switchErr)))
		}
	}()

	// Update workflows using the normal function logic
	if err := updateWorkflowsNormal(workflowsToUpdate, verbose); err != nil {
		// Rollback on error
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to update workflows: %w", err)
	}

	// Stage all files before creating PR
	if err := tracker.StageAllFiles(verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to stage workflow files: %w", err)
	}

	// Update .gitattributes and stage it if modified
	if err := stageGitAttributesIfChanged(); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to stage .gitattributes: %v", err)))
	}

	// Commit changes
	var commitMessage, prTitle, prBody string
	if len(workflowsToUpdate) == 1 {
		commitMessage = fmt.Sprintf("Update workflow: %s", workflowsToUpdate[0].Name)
		prTitle = fmt.Sprintf("Update workflow: %s", workflowsToUpdate[0].Name)
		prBody = fmt.Sprintf("Automatically created PR to update workflow: %s\n\nSource: %s", workflowsToUpdate[0].Name, workflowsToUpdate[0].SourceSpec)
	} else {
		var workflowNames []string
		for _, wf := range workflowsToUpdate {
			workflowNames = append(workflowNames, wf.Name)
		}
		commitMessage = fmt.Sprintf("Update workflows: %s", strings.Join(workflowNames, ", "))
		prTitle = fmt.Sprintf("Update %d workflows from source", len(workflowsToUpdate))
		prBody = "Automatically created PR to update workflows from their source repositories.\n\nUpdated workflows:\n"
		for _, wf := range workflowsToUpdate {
			prBody += fmt.Sprintf("- %s (source: %s)\n", wf.Name, wf.SourceSpec)
		}
	}

	if err := commitChanges(commitMessage, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to commit files: %w", err)
	}

	// Push branch
	if err := pushBranch(branchName, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	// Create PR
	if err := createPR(branchName, prTitle, prBody, verbose); err != nil {
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Success
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully created pull request with workflow updates"))

	return nil
}

// WorkflowWithSource represents a workflow file with source information
type WorkflowWithSource struct {
	Name       string // Base filename without .md extension
	FilePath   string // Full path to the workflow file
	SourceSpec string // The source field value (e.g., "org/repo sha path.md")
}

// findWorkflowsWithSource finds all workflows with source field
func findWorkflowsWithSource(workflowsDir string, filter []string, verbose bool) ([]WorkflowWithSource, error) {
	var result []WorkflowWithSource

	// Read all .md files in the workflows directory
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Skip .lock.yml files
		if strings.HasSuffix(entry.Name(), ".lock.yml") {
			continue
		}

		filePath := filepath.Join(workflowsDir, entry.Name())
		baseName := strings.TrimSuffix(entry.Name(), ".md")

		// If filter is specified, check if this workflow matches
		if len(filter) > 0 {
			matches := false
			for _, f := range filter {
				if baseName == f || baseName == strings.TrimSuffix(f, ".md") {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}

		// Read and parse the workflow to check for source field
		content, err := os.ReadFile(filePath)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", entry.Name(), err)))
			}
			continue
		}

		frontmatter, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse %s: %v", entry.Name(), err)))
			}
			continue
		}

		// Check if source field exists
		if source, ok := frontmatter.Frontmatter["source"].(string); ok && source != "" {
			result = append(result, WorkflowWithSource{
				Name:       baseName,
				FilePath:   filePath,
				SourceSpec: source,
			})
		}
	}

	return result, nil
}

// updateSingleWorkflow updates a single workflow from its source
func updateSingleWorkflow(wf WorkflowWithSource, verbose bool, tracker *FileTracker) error {
	// Parse the source spec: "org/repo ref path.md"
	parts := strings.Fields(wf.SourceSpec)
	if len(parts) != 3 {
		return fmt.Errorf("invalid source format: expected 'org/repo ref path.md', got '%s'", wf.SourceSpec)
	}

	repo := parts[0]
	ref := parts[1]
	workflowPath := parts[2]

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Source: %s @ %s (%s)", repo, ref, workflowPath)))
	}

	// Install or update the package to get the latest version
	// Note: This will update the package in the packages directory
	if err := InstallPackage(repo+"@"+ref, false, verbose); err != nil {
		return fmt.Errorf("failed to install/update package: %w", err)
	}

	// Now read the workflow from the package
	packagesDir, err := getPackagesDir(false)
	if err != nil {
		return err
	}

	packagePath := filepath.Join(packagesDir, repo)
	sourceFilePath := filepath.Join(packagePath, workflowPath)

	sourceContent, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return fmt.Errorf("failed to read source workflow: %w", err)
	}

	// Read the current local workflow
	localContent, err := os.ReadFile(wf.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read local workflow: %w", err)
	}

	// Parse both frontmatter and markdown
	sourceFrontmatter, err := parser.ExtractFrontmatterFromContent(string(sourceContent))
	if err != nil {
		return fmt.Errorf("failed to parse source frontmatter: %w", err)
	}

	localFrontmatter, err := parser.ExtractFrontmatterFromContent(string(localContent))
	if err != nil {
		return fmt.Errorf("failed to parse local frontmatter: %w", err)
	}

	// Merge strategy:
	// 1. Keep local markdown content (user may have customized it)
	// 2. Update frontmatter from source, but preserve the source field and any local customizations
	// 3. If frontmatter has conflicts, prefer source values but warn user

	// Start with source frontmatter
	mergedFrontmatter := make(map[string]any)
	for k, v := range sourceFrontmatter.Frontmatter {
		mergedFrontmatter[k] = v
	}

	// Preserve the source field from local (it may have been updated)
	if source, ok := localFrontmatter.Frontmatter["source"].(string); ok {
		mergedFrontmatter["source"] = source
	}

	// Use local markdown content
	mergedMarkdown := localFrontmatter.Markdown

	// Reconstruct the workflow
	frontmatterYAML, err := yaml.Marshal(mergedFrontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal merged frontmatter: %w", err)
	}

	var lines []string
	lines = append(lines, "---")
	frontmatterStr := strings.TrimSuffix(string(frontmatterYAML), "\n")
	if frontmatterStr != "" {
		lines = append(lines, strings.Split(frontmatterStr, "\n")...)
	}
	lines = append(lines, "---")
	if mergedMarkdown != "" {
		lines = append(lines, mergedMarkdown)
	}

	updatedContent := strings.Join(lines, "\n")

	// Check if content actually changed
	if string(localContent) == updatedContent {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No changes needed for %s", wf.Name)))
		}
		return nil
	}

	// Write the updated content
	if err := os.WriteFile(wf.FilePath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}

	// Track the modification if tracker is available
	if tracker != nil {
		tracker.TrackModified(wf.FilePath)
	}

	// Compile the updated workflow
	if tracker != nil {
		if err := compileWorkflowWithTracking(wf.FilePath, verbose, "", tracker); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to compile updated workflow: %v", err)))
		}
	} else {
		if err := compileWorkflow(wf.FilePath, verbose, ""); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to compile updated workflow: %v", err)))
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s", wf.Name)))

	return nil
}
