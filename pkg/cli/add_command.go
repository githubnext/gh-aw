package cli

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var addLog = logger.New("cli:add_command")

// NewAddCommand creates the add command
func NewAddCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <workflow>...",
		Short: "Add one or more workflows from the components to .github/workflows",
		Long: `Add one or more workflows from repositories to .github/workflows.

Examples:
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/ci-doctor
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/ci-doctor@v1.0.0
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/workflows/ci-doctor.md@main
  ` + constants.CLIExtensionPrefix + ` add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md
  ` + constants.CLIExtensionPrefix + ` add githubnext/agentics/ci-doctor --pr --force

Workflow specifications:
  - Three parts: "owner/repo/workflow-name[@version]" (implicitly looks in workflows/ directory)
  - Four+ parts: "owner/repo/workflows/workflow-name.md[@version]" (requires explicit .md extension)
  - GitHub URL: "https://github.com/owner/repo/blob/branch/path/to/workflow.md"
  - Version can be tag, branch, or SHA

The -n flag allows you to specify a custom name for the workflow file (only applies to the first workflow when adding multiple).
The --pr flag automatically creates a pull request with the workflow changes.
The --force flag overwrites existing workflow files.`,
		Run: func(cmd *cobra.Command, args []string) {
			workflows := args
			numberFlag, _ := cmd.Flags().GetInt("number")
			engineOverride, _ := cmd.Flags().GetString("engine")
			nameFlag, _ := cmd.Flags().GetString("name")
			prFlag, _ := cmd.Flags().GetBool("pr")
			forceFlag, _ := cmd.Flags().GetBool("force")
			appendText, _ := cmd.Flags().GetString("append")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// If no arguments provided and not in CI, automatically use interactive mode
			if len(args) == 0 && !IsRunningInCI() {
				// Auto-enable interactive mode
				var workflowName = "my-workflow" // Default name
				if err := CreateWorkflowInteractively(workflowName, verbose, false); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
					os.Exit(1)
				}
				// Exit successfully after interactive creation
				os.Exit(0)
			}

			if err := validateEngine(engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}

			// Handle normal mode
			if prFlag {
				if err := AddWorkflows(workflows, numberFlag, verbose, engineOverride, nameFlag, forceFlag, appendText, true); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
					os.Exit(1)
				}
			} else {
				if err := AddWorkflows(workflows, numberFlag, verbose, engineOverride, nameFlag, forceFlag, appendText, false); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
					os.Exit(1)
				}
			}
		},
	}

	// Add number flag to add command
	cmd.Flags().IntP("number", "c", 1, "Create multiple numbered copies")

	// Add name flag to add command
	cmd.Flags().StringP("name", "n", "", "Specify name for the added workflow (without .md extension)")

	// Add AI flag to add command
	cmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, copilot, custom)")

	// Add repository flag to add command
	cmd.Flags().StringP("repo", "r", "", "Install and use workflows from specified repository (org/repo)")

	// Add PR flag to add command
	cmd.Flags().Bool("pr", false, "Create a pull request with the workflow changes")

	// Add force flag to add command
	cmd.Flags().Bool("force", false, "Overwrite existing workflow files")

	// Add append flag to add command
	cmd.Flags().String("append", "", "Append extra content to the end of agentic workflow on installation")

	return cmd
}

// AddWorkflows adds one or more workflows from components to .github/workflows
// with optional repository installation and PR creation
func AddWorkflows(workflows []string, number int, verbose bool, engineOverride string, name string, force bool, appendText string, createPR bool) error {
	addLog.Printf("Adding workflows: count=%d, engineOverride=%s, createPR=%v", len(workflows), engineOverride, createPR)

	if len(workflows) == 0 {
		return fmt.Errorf("at least one workflow name is required")
	}

	for i, workflow := range workflows {
		if workflow == "" {
			return fmt.Errorf("workflow name cannot be empty (workflow %d)", i+1)
		}
	}

	// If creating a PR, check prerequisites
	if createPR {
		// Check if GitHub CLI is available
		if !isGHCLIAvailable() {
			return fmt.Errorf("GitHub CLI (gh) is required for PR creation but not available")
		}

		// Check if we're in a git repository
		if !isGitRepo() {
			return fmt.Errorf("not in a git repository - PR creation requires a git repository")
		}

		// Check no other changes are present
		if err := checkCleanWorkingDirectory(verbose); err != nil {
			return fmt.Errorf("working directory is not clean: %w", err)
		}
	}

	// Parse workflow specifications and group by repository
	repoVersions := make(map[string]string) // repo -> version
	processedWorkflows := []*WorkflowSpec{} // List of processed workflow specs

	for _, workflow := range workflows {
		spec, err := parseWorkflowSpec(workflow)
		if err != nil {
			return fmt.Errorf("invalid workflow specification '%s': %w", workflow, err)
		}

		// Handle repository installation and workflow name extraction
		if existing, exists := repoVersions[spec.RepoSlug]; exists && existing != spec.Version {
			return fmt.Errorf("conflicting versions for repository %s: %s vs %s", spec.RepoSlug, existing, spec.Version)
		}
		repoVersions[spec.RepoSlug] = spec.Version

		// Create qualified name for processing
		processedWorkflows = append(processedWorkflows, spec)
	}

	// Install required repositories
	for repo, version := range repoVersions {
		repoWithVersion := repo
		if version != "" {
			repoWithVersion = fmt.Sprintf("%s@%s", repo, version)
		}

		addLog.Printf("Installing repository: %s", repoWithVersion)

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing repository %s before adding workflows...", repoWithVersion)))
		}

		// Install as global package (not local) to match the behavior expected
		if err := InstallPackage(repoWithVersion, verbose); err != nil {
			addLog.Printf("Failed to install repository %s: %v", repoWithVersion, err)
			return fmt.Errorf("failed to install repository %s: %w", repoWithVersion, err)
		}
	}

	// Handle PR creation workflow
	if createPR {
		addLog.Print("Creating workflow with PR")
		return addWorkflowsWithPR(processedWorkflows, number, verbose, engineOverride, name, force, appendText)
	}

	// Handle normal workflow addition
	addLog.Print("Adding workflows normally without PR")
	return addWorkflowsNormal(processedWorkflows, number, verbose, engineOverride, name, force, appendText)
}

// addWorkflowsNormal handles normal workflow addition without PR creation
func addWorkflowsNormal(workflows []*WorkflowSpec, number int, verbose bool, engineOverride string, name string, force bool, appendText string) error {
	// Create file tracker for all operations
	tracker, err := NewFileTracker()
	if err != nil {
		// If we can't create a tracker (e.g., not in git repo), fall back to non-tracking behavior
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not create file tracker: %v", err)))
		}
		tracker = nil
	}

	if len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding %d workflow(s)...", len(workflows))))
	}

	// Add each workflow
	for i, workflow := range workflows {
		if len(workflows) > 1 {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Adding workflow %d/%d: %s", i+1, len(workflows), workflow.WorkflowName)))
		}

		// For multiple workflows, only use the name flag for the first one
		currentName := ""
		if i == 0 && name != "" {
			currentName = name
		}

		if err := addWorkflowWithTracking(workflow, number, verbose, engineOverride, currentName, force, appendText, tracker); err != nil {
			return fmt.Errorf("failed to add workflow '%s': %w", workflow.String(), err)
		}
	}

	if len(workflows) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully added all %d workflows", len(workflows))))
	}

	return nil
}

// addWorkflowsWithPR handles workflow addition with PR creation
func addWorkflowsWithPR(workflows []*WorkflowSpec, number int, verbose bool, engineOverride string, name string, force bool, appendText string) error {
	// Get current branch for restoration later
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create temporary branch with random 4-digit number
	randomNum := rand.Intn(9000) + 1000 // Generate number between 1000-9999
	branchName := fmt.Sprintf("add-workflow-%s-%04d", strings.ReplaceAll(workflows[0].WorkflowPath, "/", "-"), randomNum)

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

	// Add workflows using the normal function logic
	if err := addWorkflowsNormal(workflows, number, verbose, engineOverride, name, force, appendText); err != nil {
		// Rollback on error
		if rollbackErr := tracker.RollbackAllFiles(verbose); rollbackErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to rollback files: %v", rollbackErr)))
		}
		return fmt.Errorf("failed to add workflows: %w", err)
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

	// Success - no rollback needed

	// Switch back to original branch
	if err := switchBranch(currentBranch, verbose); err != nil {
		return fmt.Errorf("failed to switch back to branch %s: %w", currentBranch, err)
	}

	if len(workflows) == 1 {
		fmt.Printf("Successfully created PR for workflow: %s\n", workflows[0].WorkflowName)
	} else {
		fmt.Printf("Successfully created PR for workflows: %s\n", joinedNames)
	}
	return nil
}

// addWorkflowWithTracking adds a workflow from components to .github/workflows with file tracking
func addWorkflowWithTracking(workflow *WorkflowSpec, number int, verbose bool, engineOverride string, name string, force bool, appendText string, tracker *FileTracker) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding workflow: %s", workflow.String())))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Number of copies: %d", number)))
		if force {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Force flag enabled: will overwrite existing files"))
		}
	}

	// Validate number of copies
	if number < 1 {
		return fmt.Errorf("number of copies must be a positive integer")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, "Locating workflow components...")
	}

	workflowPath := workflow.WorkflowPath

	if verbose {
		fmt.Printf("Looking for workflow file: %s\n", workflowPath)
	}

	// Try to read the workflow content from multiple sources
	sourceContent, sourceInfo, err := findWorkflowInPackageForRepo(workflow, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Workflow '%s' not found.", workflowPath)))

		// Provide information about workflow repositories
		fmt.Println("\nTo add workflows to your project:")
		fmt.Println("=================================")
		fmt.Println("Use the 'add' command with repository/workflow specifications:")
		fmt.Println("  " + constants.CLIExtensionPrefix + " add owner/repo/workflow-name")
		fmt.Println("  " + constants.CLIExtensionPrefix + " add owner/repo/workflow-name@version")
		fmt.Println("\nExample:")
		fmt.Println("  " + constants.CLIExtensionPrefix + " add githubnext/agentics/ci-doctor")
		fmt.Println("  " + constants.CLIExtensionPrefix + " add githubnext/agentics/daily-plan@main")

		return fmt.Errorf("workflow not found: %s", workflowPath)
	}

	if verbose {
		fmt.Printf("Successfully read workflow content (%d bytes)\n", len(sourceContent))
	}

	// Find git root to ensure consistent placement
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("add workflow requires being in a git repository: %w", err)
	}

	// Ensure .github/workflows directory exists relative to git root
	githubWorkflowsDir := filepath.Join(gitRoot, ".github/workflows")
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Determine the workflowName to use
	var workflowName string
	if name != "" {
		// Use the explicitly provided name
		workflowName = name
	} else {
		// Extract filename from workflow path and remove .md extension for processing
		workflowName = workflow.WorkflowName
	}

	// Check if a workflow with this name already exists
	existingFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	if _, err := os.Stat(existingFile); err == nil && !force {
		return fmt.Errorf("workflow '%s' already exists in .github/workflows/. Use a different name with -n flag, remove the existing workflow first, or use --force to overwrite", workflowName)
	}

	// Collect all @include dependencies from the workflow file
	includeDeps, err := collectPackageIncludeDependencies(string(sourceContent), sourceInfo.PackagePath, verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to collect include dependencies: %v", err)))
	}

	// Copy all @include dependencies to .github/workflows maintaining relative paths
	if err := copyIncludeDependenciesFromPackageWithForce(includeDeps, githubWorkflowsDir, verbose, force, tracker); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy include dependencies: %v", err)))
	}

	// Process each copy
	for i := 1; i <= number; i++ {
		// Construct the destination file path with numbering in .github/workflows
		var destFile string
		if number == 1 {
			destFile = filepath.Join(githubWorkflowsDir, workflowName+".md")
		} else {
			destFile = filepath.Join(githubWorkflowsDir, fmt.Sprintf("%s-%d.md", workflowName, i))
		}

		// Check if destination file already exists
		fileExists := false
		if _, err := os.Stat(destFile); err == nil {
			fileExists = true
			if !force {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Destination file '%s' already exists, skipping.", destFile)))
				continue
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Overwriting existing file: %s", destFile)))
		}

		// Process content for numbered workflows
		content := string(sourceContent)
		if number > 1 {
			// Update H1 title to include number
			content = updateWorkflowTitle(content, i)
		}

		// Add source field to frontmatter
		sourceString := buildSourceStringWithCommitSHA(workflow, sourceInfo.CommitSHA)
		if sourceString != "" {
			updatedContent, err := addSourceToWorkflow(content, sourceString)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to add source field: %v", err)))
				}
			} else {
				content = updatedContent
			}

			// Process imports field and replace with workflowspec
			processedImportsContent, err := processImportsWithWorkflowSpec(content, workflow, sourceInfo.CommitSHA, verbose)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process imports: %v", err)))
				}
			} else {
				content = processedImportsContent
			}

			// Process @include directives and replace with workflowspec
			processedContent, err := processIncludesWithWorkflowSpec(content, workflow, sourceInfo.CommitSHA, sourceInfo.PackagePath, verbose)
			if err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process includes: %v", err)))
				}
			} else {
				content = processedContent
			}
		}

		// Append text if provided
		if appendText != "" {
			// Ensure we have a newline before appending
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "\n" + appendText
		}

		// Track the file based on whether it existed before (if tracker is available)
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(destFile)
			} else {
				tracker.TrackCreated(destFile)
			}
		}

		// Write the file
		if err := os.WriteFile(destFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write destination file '%s': %w", destFile, err)
		}

		fmt.Printf("Added workflow: %s\n", destFile)

		// Try to compile the workflow and track generated files
		if tracker != nil {
			if err := compileWorkflowWithTracking(destFile, verbose, engineOverride, tracker); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			// Fall back to basic compilation without tracking
			if err := compileWorkflow(destFile, verbose, engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}

	// Stage tracked files to git if in a git repository
	if isGitRepo() && tracker != nil {
		if err := tracker.StageAllFiles(verbose); err != nil {
			return fmt.Errorf("failed to stage workflow files: %w", err)
		}
	}

	return nil
}

func updateWorkflowTitle(content string, number int) string {
	// Find and update the first H1 header
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			// Extract the title part and add number
			title := strings.TrimSpace(line[2:])
			lines[i] = fmt.Sprintf("# %s %d", title, number)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func compileWorkflow(filePath string, verbose bool, engineOverride string) error {
	// Create compiler and compile the workflow
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	if err := CompileWorkflowWithValidation(compiler, filePath, verbose, false, false, false); err != nil {
		return err
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to update .gitattributes: %v\n", err)
		}
	}

	// Note: Instructions are only written when explicitly requested via the compile command flag
	// This helper function is used in contexts where instructions should not be automatically written

	return nil
}

// compileWorkflowWithTracking compiles a workflow and tracks generated files
func compileWorkflowWithTracking(filePath string, verbose bool, engineOverride string, tracker *FileTracker) error {
	// Generate the expected lock file path
	lockFile := strings.TrimSuffix(filePath, ".md") + ".lock.yml"

	// Check if lock file exists before compilation
	lockFileExists := false
	if _, err := os.Stat(lockFile); err == nil {
		lockFileExists = true
	}

	// Check if .gitattributes exists before ensuring it
	gitRoot, err := findGitRoot()
	if err != nil {
		return err
	}
	gitAttributesPath := filepath.Join(gitRoot, ".gitattributes")
	gitAttributesExists := false
	if _, err := os.Stat(gitAttributesPath); err == nil {
		gitAttributesExists = true
	}

	// Track the lock file before compilation
	if lockFileExists {
		tracker.TrackModified(lockFile)
	} else {
		tracker.TrackCreated(lockFile)
	}

	// Track .gitattributes file before modification
	if gitAttributesExists {
		tracker.TrackModified(gitAttributesPath)
	} else {
		tracker.TrackCreated(gitAttributesPath)
	}

	// Create compiler and set the file tracker
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	compiler.SetFileTracker(tracker)
	if err := CompileWorkflowWithValidation(compiler, filePath, verbose, false, false, false); err != nil {
		return err
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to update .gitattributes: %v\n", err)
		}
	}

	return nil
}

// ensureCopilotInstructions ensures that .github/instructions/github-agentic-workflows.md contains the copilot instructions
func ensureCopilotInstructions(verbose bool, skipInstructions bool) error {
	if skipInstructions {
		return nil // Skip writing instructions if flag is set
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	copilotDir := filepath.Join(gitRoot, ".github", "instructions")
	copilotInstructionsPath := filepath.Join(copilotDir, "github-agentic-workflows.instructions.md")

	// Ensure the .github/instructions directory exists
	if err := os.MkdirAll(copilotDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/instructions directory: %w", err)
	}

	// Check if the instructions file already exists and matches the template
	existingContent := ""
	if content, err := os.ReadFile(copilotInstructionsPath); err == nil {
		existingContent = string(content)
	}

	// Check if content matches our expected template
	expectedContent := strings.TrimSpace(copilotInstructionsTemplate)
	if strings.TrimSpace(existingContent) == expectedContent {
		if verbose {
			fmt.Printf("Copilot instructions are up-to-date: %s\n", copilotInstructionsPath)
		}
		return nil
	}

	// Write the copilot instructions file
	if err := os.WriteFile(copilotInstructionsPath, []byte(copilotInstructionsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}

	if verbose {
		if existingContent == "" {
			fmt.Printf("Created copilot instructions: %s\n", copilotInstructionsPath)
		} else {
			fmt.Printf("Updated copilot instructions: %s\n", copilotInstructionsPath)
		}
	}

	return nil
}

// ensureAgenticWorkflowPrompt removes the old agentic workflow prompt file if it exists
func ensureAgenticWorkflowPrompt(verbose bool, skipInstructions bool) error {
	// This function now removes the old prompt file since we've migrated to agent format
	if skipInstructions {
		return nil
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	promptsDir := filepath.Join(gitRoot, ".github", "prompts")
	oldPromptPath := filepath.Join(promptsDir, "create-agentic-workflow.prompt.md")

	// Check if the old prompt file exists and remove it
	if _, err := os.Stat(oldPromptPath); err == nil {
		if err := os.Remove(oldPromptPath); err != nil {
			return fmt.Errorf("failed to remove old prompt file: %w", err)
		}
		if verbose {
			fmt.Printf("Removed old prompt file: %s\n", oldPromptPath)
		}
	}

	return nil
}

// ensureAgenticWorkflowAgent ensures that .github/agents/create-agentic-workflow.md contains the agentic workflow creation agent
func ensureAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("create-agentic-workflow.md", agenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// ensureSharedAgenticWorkflowAgent ensures that .github/agents/create-shared-agentic-workflow.md contains the shared workflow creation agent
func ensureSharedAgenticWorkflowAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("create-shared-agentic-workflow.md", sharedAgenticWorkflowAgentTemplate, verbose, skipInstructions)
}

// ensureSetupAgenticWorkflowsAgent ensures that .github/agents/setup-agentic-workflows.md contains the setup guide agent
func ensureSetupAgenticWorkflowsAgent(verbose bool, skipInstructions bool) error {
	return ensureAgentFromTemplate("setup-agentic-workflows.md", setupAgenticWorkflowsAgentTemplate, verbose, skipInstructions)
}

// checkCleanWorkingDirectory checks if there are uncommitted changes
func checkCleanWorkingDirectory(verbose bool) error {
	if verbose {
		fmt.Printf("Checking for uncommitted changes...\n")
	}

	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		return fmt.Errorf("working directory has uncommitted changes, please commit or stash them first")
	}

	if verbose {
		fmt.Printf("Working directory is clean\n")
	}
	return nil
}

// createPR creates a pull request using GitHub CLI
func createPR(branchName, title, body string, verbose bool) error {
	if verbose {
		fmt.Printf("Creating PR: %s\n", title)
	}

	// Get the current repository info to ensure PR is created in the correct repo
	cmd := exec.Command("gh", "repo", "view", "--json", "owner,name")
	repoOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current repository info: %w", err)
	}

	var repoInfo struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(repoOutput, &repoInfo); err != nil {
		return fmt.Errorf("failed to parse repository info: %w", err)
	}

	repoSpec := fmt.Sprintf("%s/%s", repoInfo.Owner.Login, repoInfo.Name)

	// Explicitly specify the repository to ensure PR is created in the current repo (not upstream)
	cmd = exec.Command("gh", "pr", "create", "--repo", repoSpec, "--title", title, "--body", body, "--head", branchName)
	output, err := cmd.Output()
	if err != nil {
		// Try to get stderr for better error reporting
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("failed to create PR: %w\nOutput: %s\nError: %s", err, string(output), string(exitError.Stderr))
		}
		return fmt.Errorf("failed to create PR: %w", err)
	}

	prURL := strings.TrimSpace(string(output))
	fmt.Printf("📢 Pull Request created: %s\n", prURL)

	return nil
}

// addSourceToWorkflow adds the source field to the workflow's frontmatter
func addSourceToWorkflow(content, source string) (string, error) {
	// Use shared frontmatter logic that preserves formatting
	return addFieldToFrontmatter(content, "source", source)
}
