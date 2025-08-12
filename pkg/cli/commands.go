package cli

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// Package-level version information
var (
	version = "dev"
)

//go:embed templates/auto-compile-workflow.yml
var autoCompileWorkflowTemplate string

//go:embed templates/instructions.md
var copilotInstructionsTemplate string

// ensureAutoCompileWorkflow checks if the auto-compile workflow exists and is up-to-date
func ensureAutoCompileWorkflow(verbose bool) error {
	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("auto-compile workflow management requires being in a git repository: %w", err)
	}

	workflowsDir := filepath.Join(gitRoot, ".github/workflows")
	autoCompileFile := filepath.Join(workflowsDir, "auto-compile-workflows.yml")

	// Check if the workflow file exists
	needsUpdate := false
	existingContent := ""

	if content, err := os.ReadFile(autoCompileFile); err != nil {
		if os.IsNotExist(err) {
			needsUpdate = true
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Auto-compile workflow not found, will create: %s", autoCompileFile)))
			}
		} else {
			return fmt.Errorf("failed to read auto-compile workflow: %w", err)
		}
	} else {
		existingContent = string(content)

		// Always use the fast install template (no rebuild)
		expectedTemplate := autoCompileWorkflowTemplate

		// Check if content matches our expected template
		if strings.TrimSpace(existingContent) != strings.TrimSpace(expectedTemplate) {
			needsUpdate = true
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Auto-compile workflow is outdated, will update: %s", autoCompileFile)))
			}
		} else if verbose {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Auto-compile workflow is up-to-date: %s", autoCompileFile)))
		}
	}

	// Update the workflow if needed
	if needsUpdate {
		// Always use the fast install template (no rebuild)
		templateToWrite := autoCompileWorkflowTemplate

		// Ensure the workflows directory exists
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			return fmt.Errorf("failed to create workflows directory: %w", err)
		}

		// Write the auto-compile workflow
		if err := os.WriteFile(autoCompileFile, []byte(templateToWrite), 0644); err != nil {
			return fmt.Errorf("failed to write auto-compile workflow: %w", err)
		}

		if verbose {
			if existingContent == "" {
				fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Created auto-compile workflow: %s", autoCompileFile)))
			} else {
				fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Updated auto-compile workflow: %s", autoCompileFile)))
			}
		}
	}

	return nil
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(v string) {
	version = v
}

// GetVersion returns the current version
func GetVersion() string {
	return version
}

// GitHubWorkflow represents a GitHub Actions workflow from the API
type GitHubWorkflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

// GitHubWorkflowsResponse represents the GitHub API response for workflows
// Note: The API returns an array directly, not wrapped in a workflows field

// ListWorkflows lists available workflow components
func ListWorkflows(verbose bool) error {
	if verbose {
		fmt.Println(console.FormatProgressMessage("Searching for available workflow components..."))
	}

	// First list available agentic engines
	if err := listAgenticEngines(verbose); err != nil {
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to list agentic engines: %v", err)))
	}

	// Then list package workflows
	return listPackageWorkflows(verbose)
}

// listAgenticEngines lists all available agentic engines with their characteristics
func listAgenticEngines(verbose bool) error {
	// Create an engine registry directly to access the engines
	registry := workflow.NewEngineRegistry()

	// Get all supported engines from the registry
	engines := registry.GetSupportedEngines()

	if len(engines) == 0 {
		fmt.Println(console.FormatInfoMessage("No agentic engines available."))
		return nil
	}

	fmt.Println(console.FormatListHeader("Available Agentic Engines"))
	fmt.Println(console.FormatListHeader("========================"))

	if verbose {
		fmt.Printf("%-15s %-20s %-12s %-8s %-8s %s\n", "ID", "Display Name", "Status", "MCP", "Node.js", "Description")
		fmt.Printf("%-15s %-20s %-12s %-8s %-8s %s\n", "--", "------------", "------", "---", "-------", "-----------")
	} else {
		fmt.Printf("%-15s %-20s %-12s %-8s %-8s\n", "ID", "Display Name", "Status", "MCP", "Node.js")
		fmt.Printf("%-15s %-20s %-12s %-8s %-8s\n", "--", "------------", "------", "---", "-------")
	}

	for _, engineID := range engines {
		engine, err := registry.GetEngine(engineID)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to get engine '%s': %v\n", engineID, err)
			}
			continue
		}

		// Determine status
		status := "Stable"
		if engine.IsExperimental() {
			status = "Experimental"
		}

		// MCP support
		mcpSupport := "No"
		if engine.SupportsToolsWhitelist() {
			mcpSupport = "Yes"
		}

		if verbose {
			fmt.Printf("%-15s %-20s %-12s %-8s %s\n",
				engine.GetID(),
				engine.GetDisplayName(),
				status,
				mcpSupport,
				engine.GetDescription())

		} else {
			fmt.Printf("%-15s %-20s %-12s %-8s\n",
				engine.GetID(),
				engine.GetDisplayName(),
				status,
				mcpSupport)
		}
	}

	fmt.Println()
	return nil
}

// AddWorkflowWithRepo adds a workflow from components to .github/workflows
// with optional repository installation
func AddWorkflowWithRepo(workflow string, number int, verbose bool, engineOverride string, repoSpec string, name string, force bool) error {
	// If repo spec is specified, install it first
	if repoSpec != "" {
		repo, _, err := parseRepoSpec(repoSpec)
		if err != nil {
			return fmt.Errorf("invalid repository specification: %w", err)
		}

		if verbose {
			fmt.Printf("Installing repository %s before adding workflow...\n", repoSpec)
		}
		// Install as global package (not local) to match the behavior expected
		if err := InstallPackage(repoSpec, false, verbose); err != nil {
			return fmt.Errorf("failed to install repository %s: %w", repoSpec, err)
		}

		// Prepend the repo to the workflow name to form a qualified name
		// This ensures we use the workflow from the newly installed package
		workflow = fmt.Sprintf("%s/%s", repo, workflow)
	}

	// Call the original AddWorkflow function
	return AddWorkflow(workflow, number, verbose, engineOverride, name, force)
}

// AddWorkflowWithRepoAndPR adds a workflow from components to .github/workflows
// with optional repository installation and creates a PR
func AddWorkflowWithRepoAndPR(workflow string, number int, verbose bool, engineOverride string, repoSpec string, name string, force bool) error {
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

	// Get current branch for restoration later
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create temporary branch with random 4-digit number
	randomNum := rand.Intn(9000) + 1000 // Generate number between 1000-9999
	branchName := fmt.Sprintf("add-workflow-%s-%04d", strings.ReplaceAll(workflow, "/", "-"), randomNum)
	if err := createAndSwitchBranch(branchName, verbose); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Ensure we return to original branch on error
	defer func() {
		if err := switchBranch(currentBranch, verbose); err != nil && verbose {
			fmt.Printf("Warning: Failed to switch back to original branch %s: %v\n", currentBranch, err)
		}
	}()

	// If repo spec is specified, install it first
	if repoSpec != "" {
		repo, _, err := parseRepoSpec(repoSpec)
		if err != nil {
			return fmt.Errorf("invalid repository specification: %w", err)
		}

		if verbose {
			fmt.Printf("Installing repository %s before adding workflow...\n", repoSpec)
		}
		// Install as global package (not local) to match the behavior expected
		if err := InstallPackage(repoSpec, false, verbose); err != nil {
			return fmt.Errorf("failed to install repository %s: %w", repoSpec, err)
		}

		// Prepend the repo to the workflow name to form a qualified name
		// This ensures we use the workflow from the newly installed package
		workflow = fmt.Sprintf("%s/%s", repo, workflow)
	}

	// Add workflow files using existing logic
	if err := AddWorkflow(workflow, number, verbose, engineOverride, name, force); err != nil {
		return fmt.Errorf("failed to add workflow: %w", err)
	}

	// Commit changes
	commitMessage := fmt.Sprintf("Add workflow: %s", workflow)
	if err := commitChanges(commitMessage, verbose); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push branch
	if err := pushBranch(branchName, verbose); err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	// Create PR
	prTitle := fmt.Sprintf("Add workflow: %s", workflow)
	prBody := fmt.Sprintf("Automatically created PR to add workflow: %s", workflow)
	if err := createPR(branchName, prTitle, prBody, verbose); err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Switch back to original branch
	if err := switchBranch(currentBranch, verbose); err != nil {
		return fmt.Errorf("failed to switch back to branch %s: %w", currentBranch, err)
	}

	fmt.Printf("Successfully created PR for workflow: %s\n", workflow)
	return nil
}

// AddWorkflow adds a workflow from components to .github/workflows
func AddWorkflow(workflow string, number int, verbose bool, engineOverride string, name string, force bool) error {
	if workflow == "" {
		fmt.Println("Error: No components path specified. Usage: " + constants.CLIExtensionPrefix + " add <name>")
		// Show available workflows using the same logic as ListWorkflows
		return ListWorkflows(false)
	}

	if verbose {
		fmt.Printf("Adding workflow: %s\n", workflow)
		fmt.Printf("Number of copies: %d\n", number)
		if force {
			fmt.Printf("Force flag enabled: will overwrite existing files\n")
		}
	}

	// Validate number of copies
	if number < 1 {
		return fmt.Errorf("number of copies must be a positive integer")
	}

	if verbose {
		fmt.Println("Locating workflow components...")
	}

	workflowsDir := getWorkflowsDir()

	// Add .md extension if not present
	workflowPath := workflow
	if !strings.HasSuffix(workflowPath, ".md") {
		workflowPath += ".md"
	}

	if verbose {
		fmt.Printf("Looking for workflow file: %s\n", workflowPath)
	}

	// Try to read the workflow content from multiple sources
	sourceContent, sourceInfo, err := findAndReadWorkflow(workflowPath, workflowsDir, verbose)
	if err != nil {
		fmt.Printf("Error: Workflow '%s' not found.\n", workflow)

		// Show available workflows using the same logic as ListWorkflows
		fmt.Println("\nRun '" + constants.CLIExtensionPrefix + " list' to see available workflows.")
		fmt.Println("For packages, use '" + constants.CLIExtensionPrefix + " list --packages' to see installed packages.")
		return fmt.Errorf("workflow not found: %s", workflow)
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

	// Determine the filename to use
	var filename string
	if name != "" {
		// Use the explicitly provided name
		filename = name
	} else {
		// Extract filename from workflow path and remove .md extension for processing
		filename = filepath.Base(workflow)
		filename = strings.TrimSuffix(filename, ".md")
	}

	// Check if a workflow with this name already exists
	existingFile := filepath.Join(githubWorkflowsDir, filename+".md")
	if _, err := os.Stat(existingFile); err == nil && !force {
		return fmt.Errorf("workflow '%s' already exists in .github/workflows/. Use a different name with -n flag, remove the existing workflow first, or use --force to overwrite", filename)
	}

	// Collect all @include dependencies from the workflow file
	includeDeps, err := collectIncludeDependenciesFromSource(string(sourceContent), sourceInfo, verbose)
	if err != nil {
		fmt.Printf("Warning: Failed to collect include dependencies: %v\n", err)
	}

	// Copy all @include dependencies to .github/workflows maintaining relative paths
	if err := copyIncludeDependenciesFromSourceWithForce(includeDeps, githubWorkflowsDir, sourceInfo, verbose, force); err != nil {
		fmt.Printf("Warning: Failed to copy include dependencies: %v\n", err)
	}

	// Process each copy
	for i := 1; i <= number; i++ {
		// Construct the destination file path with numbering in .github/workflows
		var destFile string
		if number == 1 {
			destFile = filepath.Join(githubWorkflowsDir, filename+".md")
		} else {
			destFile = filepath.Join(githubWorkflowsDir, fmt.Sprintf("%s-%d.md", filename, i))
		}

		// Check if destination file already exists
		if _, err := os.Stat(destFile); err == nil && !force {
			fmt.Printf("Warning: Destination file '%s' already exists, skipping.\n", destFile)
			continue
		}

		// If force is enabled and file exists, show overwrite message
		if _, err := os.Stat(destFile); err == nil && force {
			fmt.Printf("Overwriting existing file: %s\n", destFile)
		}

		// Process content for numbered workflows
		content := string(sourceContent)
		if number > 1 {
			// Update H1 title to include number
			content = updateWorkflowTitle(content, i)
		}

		// Write the file
		if err := os.WriteFile(destFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write destination file '%s': %w", destFile, err)
		}

		fmt.Printf("Added workflow: %s\n", destFile)

		// Try to compile the workflow and then move lock file to git root
		if err := compileWorkflow(destFile, verbose, engineOverride); err != nil {
			fmt.Println(err)
		}
	}

	// Try to stage changes to git if in a git repository
	if isGitRepo() {
		stageWorkflowChanges()
	}

	return nil
}

// CompileWorkflows compiles markdown files into GitHub Actions workflow files
func CompileWorkflows(markdownFile string, verbose bool, engineOverride string, validate bool, autoCompile bool, watch bool) error {
	// Create compiler with verbose flag and AI engine override
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())

	// Set validation based on the validate flag (false by default for compatibility)
	compiler.SetSkipValidation(!validate)

	if watch {
		// Watch mode: watch for file changes and recompile automatically
		return watchAndCompileWorkflows(markdownFile, compiler, verbose, autoCompile)
	}

	if markdownFile != "" {
		if verbose {
			fmt.Printf("Compiling %s\n", markdownFile)
		}
		if err := compiler.CompileWorkflow(markdownFile); err != nil {
			return err
		}

		// Ensure auto-compile workflow is present and up-to-date if requested
		if autoCompile {
			if err := ensureAutoCompileWorkflow(verbose); err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to manage auto-compile workflow: %v\n", err)
				}
			}
		}

		// Ensure .gitattributes marks .lock.yml files as generated
		if err := ensureGitAttributes(); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to update .gitattributes: %v\n", err)
			}
		} else if verbose {
			fmt.Printf("Updated .gitattributes to mark .lock.yml files as generated\n")
		}

		// Ensure copilot instructions are present
		if err := ensureCopilotInstructions(verbose); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to update copilot instructions: %v\n", err)
			}
		}

		return nil
	}

	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}

	// Ensure auto-compile workflow is present and up-to-date if requested
	if autoCompile {
		if err := ensureAutoCompileWorkflow(verbose); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to manage auto-compile workflow: %v\n", err)
			}
		}
	}

	// Compile all markdown files in .github/workflows relative to git root
	workflowsDir := filepath.Join(gitRoot, ".github/workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("the .github/workflows directory does not exist in git root (%s)", gitRoot)
	}

	if verbose {
		fmt.Printf("Scanning for markdown files in %s\n", workflowsDir)
	}

	// Find all markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		return fmt.Errorf("no markdown files found in %s", workflowsDir)
	}

	if verbose {
		fmt.Printf("Found %d markdown files to compile\n", len(mdFiles))
	}

	// Compile each file
	for _, file := range mdFiles {
		if verbose {
			fmt.Printf("Compiling: %s\n", file)
		}
		if err := compiler.CompileWorkflow(file); err != nil {
			return err
		}
	}

	if verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled all %d workflow files", len(mdFiles))))
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to update .gitattributes: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("Updated .gitattributes to mark .lock.yml files as generated\n")
	}

	// Ensure copilot instructions are present
	if err := ensureCopilotInstructions(verbose); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to update copilot instructions: %v\n", err)
		}
	}

	return nil
}

// watchAndCompileWorkflows watches for changes to workflow files and recompiles them automatically
func watchAndCompileWorkflows(markdownFile string, compiler *workflow.Compiler, verbose bool, autoCompile bool) error {
	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("watch mode requires being in a git repository: %w", err)
	}

	workflowsDir := filepath.Join(gitRoot, ".github/workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("the .github/workflows directory does not exist in git root (%s)", gitRoot)
	}

	// If a specific file is provided, watch only that file and its directory
	if markdownFile != "" {
		if !filepath.IsAbs(markdownFile) {
			markdownFile = filepath.Join(workflowsDir, markdownFile)
		}
		if _, err := os.Stat(markdownFile); os.IsNotExist(err) {
			return fmt.Errorf("specified markdown file does not exist: %s", markdownFile)
		}
	}

	// Set up file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Add the workflows directory to the watcher
	if err := watcher.Add(workflowsDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", workflowsDir, err)
	}

	// Always emit the begin pattern for task integration
	if markdownFile != "" {
		fmt.Printf("Watching for file changes to %s...\n", markdownFile)
	} else {
		fmt.Printf("Watching for file changes in %s...\n", workflowsDir)
	}

	if verbose {
		fmt.Println("Press Ctrl+C to stop watching.")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Debouncing setup
	const debounceDelay = 300 * time.Millisecond
	var debounceTimer *time.Timer
	modifiedFiles := make(map[string]struct{})

	// Compile initially if no specific file provided
	if markdownFile == "" {
		fmt.Println("Watching for file changes")
		if verbose {
			fmt.Println("🔨 Initial compilation of all workflow files...")
		}
		if err := compileAllWorkflowFiles(compiler, workflowsDir, verbose, autoCompile); err != nil {
			// Always show initial compilation errors, not just in verbose mode
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Initial compilation failed: %v", err)))
		}
		fmt.Println("Recompiled")
	} else {
		fmt.Println("Watching for file changes")
		if verbose {
			fmt.Printf("🔨 Initial compilation of %s...\n", markdownFile)
		}
		if err := compiler.CompileWorkflow(markdownFile); err != nil {
			// Always show initial compilation errors, not just in verbose mode
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Initial compilation failed: %v", err)))
		}
		fmt.Println("Recompiled")
	}

	// Main watch loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}

			// Only process markdown files and ignore lock files
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			// If watching a specific file, only process that file
			if markdownFile != "" && event.Name != markdownFile {
				continue
			}

			if verbose {
				fmt.Printf("📝 Detected change: %s (%s)\n", event.Name, event.Op.String())
			}

			// Handle file operations
			switch {
			case event.Has(fsnotify.Remove):
				// Handle file deletion
				handleFileDeleted(event.Name, verbose)
			case event.Has(fsnotify.Write) || event.Has(fsnotify.Create):
				// Handle file modification or creation - add to debounced compilation
				modifiedFiles[event.Name] = struct{}{}

				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					filesToCompile := make([]string, 0, len(modifiedFiles))
					for file := range modifiedFiles {
						filesToCompile = append(filesToCompile, file)
					}
					// Clear the modifiedFiles map
					modifiedFiles = make(map[string]struct{})

					// Compile the modified files
					compileModifiedFiles(compiler, filesToCompile, verbose, autoCompile)
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			if verbose {
				fmt.Printf("⚠️  Watcher error: %v\n", err)
			}

		case <-sigChan:
			if verbose {
				fmt.Println("\n🛑 Stopping watch mode...")
			}
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return nil
		}
	}
}

// compileAllWorkflowFiles compiles all markdown files in the workflows directory
func compileAllWorkflowFiles(compiler *workflow.Compiler, workflowsDir string, verbose bool, autoCompile bool) error {
	// Find all markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		if verbose {
			fmt.Printf("No markdown files found in %s\n", workflowsDir)
		}
		return nil
	}

	// Compile each file
	for _, file := range mdFiles {
		if verbose {
			fmt.Printf("🔨 Compiling: %s\n", file)
		}
		if err := compiler.CompileWorkflow(file); err != nil {
			// Always show compilation errors, not just in verbose mode
			fmt.Println(err)
		} else if verbose {
			fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Compiled %s", file)))
		}
	}

	// Handle auto-compile workflow if requested
	if autoCompile {
		if err := ensureAutoCompileWorkflow(verbose); err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to manage auto-compile workflow: %v\n", err)
			}
		}
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
		}
	}

	return nil
}

// compileModifiedFiles compiles a list of modified markdown files
func compileModifiedFiles(compiler *workflow.Compiler, files []string, verbose bool, autoCompile bool) {
	if len(files) == 0 {
		return
	}

	fmt.Println("Watching for file changes")
	if verbose {
		fmt.Printf("🔨 Compiling %d modified file(s)...\n", len(files))
	}

	for _, file := range files {
		// Check if file still exists (might have been deleted between detection and compilation)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if verbose {
				fmt.Printf("📝 File %s was deleted, skipping compilation\n", file)
			}
			continue
		}

		if verbose {
			fmt.Printf("🔨 Compiling: %s\n", file)
		}

		if err := compiler.CompileWorkflow(file); err != nil {
			// Always show compilation errors, not just in verbose mode
			fmt.Println(err)
		} else if verbose {
			fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Compiled %s", file)))
		}
	}

	// Handle auto-compile workflow if requested
	if autoCompile {
		if err := ensureAutoCompileWorkflow(verbose); err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to manage auto-compile workflow: %v\n", err)
			}
		}
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
		}
	}

	fmt.Println("Recompiled")
}

// handleFileDeleted handles the deletion of a markdown file by removing its corresponding lock file
func handleFileDeleted(mdFile string, verbose bool) {
	// Generate the corresponding lock file path
	lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"

	// Check if the lock file exists and remove it
	if _, err := os.Stat(lockFile); err == nil {
		if err := os.Remove(lockFile); err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to remove lock file %s: %v\n", lockFile, err)
			}
		} else {
			if verbose {
				fmt.Printf("🗑️  Removed corresponding lock file: %s\n", lockFile)
			}
		}
	}
}

// RemoveWorkflows removes workflows matching a pattern
func RemoveWorkflows(pattern string) error {
	workflowsDir := getWorkflowsDir()

	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		fmt.Println("No .github/workflows directory found.")
		return nil
	}

	// Find all markdown files in .github/workflows
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to find workflow files: %w", err)
	}

	if len(mdFiles) == 0 {
		fmt.Println("No workflow files found to remove.")
		return nil
	}

	var filesToRemove []string

	// If no pattern specified, list all files for user to see
	if pattern == "" {
		fmt.Println("Available workflows to remove:")
		for _, file := range mdFiles {
			workflowName, _ := extractWorkflowNameFromFile(file)
			base := filepath.Base(file)
			name := strings.TrimSuffix(base, ".md")
			if workflowName != "" {
				fmt.Printf("  %-20s - %s\n", name, workflowName)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
		fmt.Println("\nUsage: " + constants.CLIExtensionPrefix + " remove <pattern>")
		return nil
	}

	// Find matching files by workflow name or filename
	for _, file := range mdFiles {
		base := filepath.Base(file)
		filename := strings.TrimSuffix(base, ".md")
		workflowName, _ := extractWorkflowNameFromFile(file)

		// Check if pattern matches filename or workflow name
		if strings.Contains(strings.ToLower(filename), strings.ToLower(pattern)) ||
			strings.Contains(strings.ToLower(workflowName), strings.ToLower(pattern)) {
			filesToRemove = append(filesToRemove, file)
		}
	}

	if len(filesToRemove) == 0 {
		fmt.Printf("No workflows found matching pattern: %s\n", pattern)
		return nil
	}

	// Show what will be removed
	fmt.Printf("The following workflows will be removed:\n")
	for _, file := range filesToRemove {
		workflowName, _ := extractWorkflowNameFromFile(file)
		if workflowName != "" {
			fmt.Printf("  %s - %s\n", filepath.Base(file), workflowName)
		} else {
			fmt.Printf("  %s\n", filepath.Base(file))
		}

		// Also check for corresponding .lock.yml file in .github/workflows
		lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
		if _, err := os.Stat(lockFile); err == nil {
			fmt.Printf("  %s (compiled workflow)\n", filepath.Base(lockFile))
		}
	}

	// Ask for confirmation
	fmt.Print("\nAre you sure you want to remove these workflows? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Operation cancelled.")
		return nil
	}

	// Remove the files
	var removedFiles []string
	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil {
			fmt.Printf("Warning: Failed to remove %s: %v\n", file, err)
		} else {
			fmt.Printf("Removed: %s\n", filepath.Base(file))
			removedFiles = append(removedFiles, file)
		}

		// Also remove corresponding .lock.yml file
		lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
		if _, err := os.Stat(lockFile); err == nil {
			if err := os.Remove(lockFile); err != nil {
				fmt.Printf("Warning: Failed to remove %s: %v\n", lockFile, err)
			} else {
				fmt.Printf("Removed: %s\n", filepath.Base(lockFile))
			}
		}
	}

	// Clean up orphaned include files
	if len(removedFiles) > 0 {
		if err := cleanupOrphanedIncludes(false); err != nil {
			fmt.Printf("Warning: Failed to clean up orphaned includes: %v\n", err)
		}
	}

	// Stage changes to git if in a git repository
	if len(removedFiles) > 0 && isGitRepo() {
		stageWorkflowChanges()
	}

	return nil
}

// StatusWorkflows shows status of workflows
// getMarkdownWorkflowFiles finds all markdown files in .github/workflows directory
func getMarkdownWorkflowFiles() ([]string, error) {
	workflowsDir := getWorkflowsDir()

	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no .github/workflows directory found")
	}

	// Find all markdown files in .github/workflows
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow files: %w", err)
	}

	return mdFiles, nil
}

func StatusWorkflows(pattern string, verbose bool) error {
	if verbose {
		fmt.Printf("Checking status of workflow files\n")
		if pattern != "" {
			fmt.Printf("Filtering by pattern: %s\n", pattern)
		}
	}

	mdFiles, err := getMarkdownWorkflowFiles()
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	if len(mdFiles) == 0 {
		fmt.Println("No workflow files found.")
		return nil
	}

	if verbose {
		fmt.Printf("Found %d markdown workflow files\n", len(mdFiles))
		fmt.Printf("Fetching GitHub workflow status...\n")
	}

	// Get GitHub workflows data
	githubWorkflows, err := fetchGitHubWorkflows(verbose)
	if err != nil {
		if verbose {
			fmt.Printf("Verbose: Failed to fetch GitHub workflows: %v\n", err)
		}
		fmt.Printf("Warning: Could not fetch GitHub workflow status: %v\n", err)
		githubWorkflows = make(map[string]*GitHubWorkflow)
	} else if verbose {
		fmt.Printf("Successfully fetched %d GitHub workflows\n", len(githubWorkflows))
	}

	fmt.Println("Workflow Status:")
	fmt.Println("================")
	fmt.Printf("%-30s %-12s %-12s %-10s\n", "Name", "Installed", "Up-to-date", "Status")
	fmt.Printf("%-30s %-12s %-12s %-10s\n", "----", "---------", "----------", "------")

	for _, file := range mdFiles {
		base := filepath.Base(file)
		name := strings.TrimSuffix(base, ".md")

		// Skip if pattern specified and doesn't match
		if pattern != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			continue
		}

		// Check if compiled (.lock.yml file is in .github/workflows)
		lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
		compiled := "No"
		upToDate := "N/A"

		if _, err := os.Stat(lockFile); err == nil {
			compiled = "Yes"

			// Check if up to date
			mdStat, _ := os.Stat(file)
			lockStat, _ := os.Stat(lockFile)
			if mdStat.ModTime().After(lockStat.ModTime()) {
				upToDate = "No"
			} else {
				upToDate = "Yes"
			}
		}

		// Get GitHub workflow status
		status := "Unknown"
		if workflow, exists := githubWorkflows[name]; exists {
			if workflow.State == "disabled_manually" {
				status = "disabled"
			} else {
				status = workflow.State
			}
		}

		fmt.Printf("%-30s %-12s %-12s %-10s\n", name, compiled, upToDate, status)
	}

	return nil
}

// EnableWorkflows enables workflows matching a pattern
func EnableWorkflows(pattern string) error {
	return toggleWorkflows(pattern, true)
}

// DisableWorkflows disables workflows matching a pattern
func DisableWorkflows(pattern string) error {
	return toggleWorkflows(pattern, false)
}

// Helper function to toggle workflows
func toggleWorkflows(pattern string, enable bool) error {
	action := "enable"
	if !enable {
		action = "disable"
	}

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is required but not available")
	}

	// Get the core set of workflows from markdown files in .github/workflows
	mdFiles, err := getMarkdownWorkflowFiles()
	if err != nil {
		// Handle missing .github/workflows directory gracefully
		fmt.Printf("No workflow files found to %s.\n", action)
		return nil
	}

	if len(mdFiles) == 0 {
		fmt.Printf("No markdown workflow files found to %s.\n", action)
		return nil
	}

	// Get GitHub workflows status for comparison
	githubWorkflows, err := fetchGitHubWorkflows(false)
	if err != nil {
		// Handle GitHub CLI authentication/connection issues gracefully
		fmt.Printf("Unable to fetch GitHub workflows (gh CLI may not be authenticated): %v\n", err)
		fmt.Printf("No workflows to %s.\n", action)
		return nil
	}

	var matchingWorkflows []GitHubWorkflow

	// Find matching workflows from the markdown files
	for _, file := range mdFiles {
		base := filepath.Base(file)
		name := strings.TrimSuffix(base, ".md")

		// Skip if pattern specified and doesn't match
		if pattern != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			continue
		}

		// Find the corresponding GitHub workflow to get status and ID
		githubWorkflow, exists := githubWorkflows[name]
		if !exists {
			fmt.Printf("Warning: No GitHub workflow found for %s\n", name)
			continue
		}

		// Check if action is needed
		if enable && githubWorkflow.State == "active" {
			continue // Already enabled
		}
		if !enable && githubWorkflow.State == "disabled_manually" {
			continue // Already disabled
		}

		matchingWorkflows = append(matchingWorkflows, *githubWorkflow)
	}

	if len(matchingWorkflows) == 0 {
		fmt.Printf("No workflows found matching pattern '%s' that need to be %sd.\n", pattern, action)
		return nil
	}

	// Show what will be changed
	fmt.Printf("The following workflows will be %sd:\n", action)
	for _, workflow := range matchingWorkflows {
		fmt.Printf("  %s (current state: %s)\n", workflow.Name, workflow.State)
	}

	// Perform the action
	for _, workflow := range matchingWorkflows {
		var cmd *exec.Cmd
		if enable {
			cmd = exec.Command("gh", "workflow", "enable", strconv.FormatInt(workflow.ID, 10))
		} else {
			// First cancel any running workflows
			if err := cancelWorkflowRuns(workflow.ID); err != nil {
				fmt.Printf("Warning: Failed to cancel runs for workflow %s: %v\n", workflow.Name, err)
			}
			cmd = exec.Command("gh", "workflow", "disable", strconv.FormatInt(workflow.ID, 10))
		}

		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to %s workflow %s: %v\n", action, workflow.Name, err)
		} else {
			fmt.Printf("%sd workflow: %s\n", strings.ToUpper(action[:1])+action[1:], workflow.Name)
		}
	}

	return nil
}

// Helper functions

func extractWorkflowNameFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Extract markdown content (excluding frontmatter)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return "", err
	}

	// Look for first H1 header
	scanner := bufio.NewScanner(strings.NewReader(result.Markdown))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:]), nil
		}
	}

	// No H1 header found, generate default name from filename
	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	baseName = strings.ReplaceAll(baseName, "-", " ")

	// Capitalize first letter of each word
	words := strings.Fields(baseName)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " "), nil
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
	if err := compiler.CompileWorkflow(filePath); err != nil {
		return err
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to update .gitattributes: %v\n", err)
		}
	}

	// Ensure copilot instructions are present
	if err := ensureCopilotInstructions(verbose); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to update copilot instructions: %v\n", err)
		}
	}

	return nil
}

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// findGitRoot finds the root directory of the git repository
func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository or git command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func stageWorkflowChanges() {
	// Find git root and add .github/workflows relative to it
	if gitRoot, err := findGitRoot(); err == nil {
		workflowsPath := filepath.Join(gitRoot, ".github/workflows/")
		_ = exec.Command("git", "-C", gitRoot, "add", workflowsPath).Run()

		// Also stage .gitattributes if it was modified
		_ = stageGitAttributesIfChanged()
	} else {
		// Fallback to relative path if git root can't be found
		_ = exec.Command("git", "add", ".github/workflows/").Run()
		_ = exec.Command("git", "add", ".gitattributes").Run()
	}
}

// ensureGitAttributes ensures that .gitattributes contains the entry to mark .lock.yml files as generated
func ensureGitAttributes() error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	gitAttributesPath := filepath.Join(gitRoot, ".gitattributes")
	lockYmlEntry := ".github/workflows/*.lock.yml linguist-generated=true merge=ours"

	// Read existing .gitattributes file if it exists
	var lines []string
	if content, err := os.ReadFile(gitAttributesPath); err == nil {
		lines = strings.Split(string(content), "\n")
	}

	// Check if the entry already exists or needs updating
	found := false
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == lockYmlEntry {
			return nil // Entry already exists with correct format
		}
		// Check for old format entry that needs updating
		if strings.HasPrefix(trimmedLine, ".github/workflows/*.lock.yml") {
			lines[i] = lockYmlEntry
			found = true
			break
		}
	}

	// Add the entry if not found
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "") // Add empty line before our entry if file doesn't end with newline
		}
		lines = append(lines, lockYmlEntry)
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(gitAttributesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .gitattributes: %w", err)
	}

	return nil
}

// ensureCopilotInstructions ensures that .github/instructions/github-agentic-workflows.md contains the copilot instructions
func ensureCopilotInstructions(verbose bool) error {
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

// stageGitAttributesIfChanged stages .gitattributes if it was modified
func stageGitAttributesIfChanged() error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return err
	}
	gitAttributesPath := filepath.Join(gitRoot, ".gitattributes")
	return exec.Command("git", "-C", gitRoot, "add", gitAttributesPath).Run()
}

func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}

func fetchGitHubWorkflows(verbose bool) (map[string]*GitHubWorkflow, error) {
	// Start spinner for network operation (only if not in verbose mode)
	spinner := console.NewSpinner("Fetching GitHub workflow status...")
	if !verbose {
		spinner.Start()
	}

	cmd := exec.Command("gh", "workflow", "list", "--all", "--json", "id,name,path,state")
	output, err := cmd.Output()

	// Stop spinner
	if !verbose {
		spinner.Stop()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute gh workflow list command: %w", err)
	}

	// Check if output is empty
	if len(output) == 0 {
		return nil, fmt.Errorf("gh workflow list returned empty output - check if repository has workflows and gh CLI is authenticated")
	}

	// Validate JSON before unmarshaling
	if !json.Valid(output) {
		return nil, fmt.Errorf("gh workflow list returned invalid JSON - this may be due to network issues or authentication problems")
	}

	var workflows []GitHubWorkflow
	if err := json.Unmarshal(output, &workflows); err != nil {
		return nil, fmt.Errorf("failed to parse workflow data: %w", err)
	}

	workflowMap := make(map[string]*GitHubWorkflow)
	for i, workflow := range workflows {
		name := extractWorkflowNameFromPath(workflow.Path)
		workflowMap[name] = &workflows[i]
	}

	return workflowMap, nil
}

func extractWorkflowNameFromPath(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return strings.TrimSuffix(name, ".lock")
}

func cancelWorkflowRuns(workflowID int64) error {
	// Start spinner for network operation
	spinner := console.NewSpinner("Cancelling workflow runs...")
	spinner.Start()

	// Get running workflow runs
	cmd := exec.Command("gh", "run", "list", "--workflow", strconv.FormatInt(workflowID, 10), "--status", "in_progress", "--json", "databaseId")
	output, err := cmd.Output()
	if err != nil {
		spinner.Stop()
		return err
	}

	var runs []struct {
		DatabaseID int64 `json:"databaseId"`
	}
	if err := json.Unmarshal(output, &runs); err != nil {
		spinner.Stop()
		return err
	}

	// Cancel each running workflow
	for _, run := range runs {
		cancelCmd := exec.Command("gh", "run", "cancel", strconv.FormatInt(run.DatabaseID, 10))
		_ = cancelCmd.Run() // Ignore errors for individual cancellations
	}

	spinner.Stop()
	return nil
}

// IncludeDependency represents a file dependency from @include directives
type IncludeDependency struct {
	SourcePath string // Path in the source (local)
	TargetPath string // Relative path where it should be copied in .github/workflows
}

// collectIncludeDependencies recursively collects all @include dependencies from a workflow file
func collectIncludeDependencies(content, workflowPath, workflowsDir string) ([]IncludeDependency, error) {
	var dependencies []IncludeDependency
	seen := make(map[string]bool) // Track already processed files to avoid cycles

	// Get the directory of the workflow file for resolving relative paths
	var workflowDir = filepath.Dir(workflowPath)

	err := collectIncludesRecursive(content, workflowDir, workflowsDir, &dependencies, seen)
	return dependencies, err
}

// collectIncludesRecursive recursively processes @include directives in content
func collectIncludesRecursive(content, baseDir, workflowsDir string, dependencies *[]IncludeDependency, seen map[string]bool) error {
	includePattern := regexp.MustCompile(`^@include\s+(.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := includePattern.FindStringSubmatch(line); matches != nil {
			includePath := strings.TrimSpace(matches[1])

			// Handle section references (file.md#Section)
			var filePath string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
			} else {
				filePath = includePath
			}

			// Resolve the full source path
			var fullSourcePath = filepath.Join(baseDir, filePath)

			// Skip if we've already processed this file
			if seen[fullSourcePath] {
				continue
			}
			seen[fullSourcePath] = true

			// Add dependency
			dep := IncludeDependency{
				SourcePath: fullSourcePath,
				TargetPath: filePath, // Keep relative path for target
			}
			*dependencies = append(*dependencies, dep)

			// Read the included file and process its includes recursively
			var includedContent []byte
			var err error
			includedContent, err = os.ReadFile(fullSourcePath)

			if err != nil {
				// If we can't read the file, add it anyway but don't recurse
				continue
			}

			// Extract markdown content from the included file
			markdownContent, err := parser.ExtractMarkdownContent(string(includedContent))
			if err != nil {
				continue // Skip if we can't extract markdown
			}

			// Recursively process includes in the included file
			includedDir := filepath.Dir(fullSourcePath)
			if err := collectIncludesRecursive(markdownContent, includedDir, workflowsDir, dependencies, seen); err != nil {
				// Log error but continue processing other includes
				fmt.Printf("Warning: Error processing includes in %s: %v\n", fullSourcePath, err)
			}
		}
	}

	return scanner.Err()
}

// copyIncludeDependenciesWithForce copies all include dependencies to the target directory with force option
func copyIncludeDependenciesWithForce(dependencies []IncludeDependency, githubWorkflowsDir string, force bool) error {
	for _, dep := range dependencies {
		// Create the target path in .github/workflows
		targetPath := filepath.Join(githubWorkflowsDir, dep.TargetPath)

		// Create target directory if it doesn't exist
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}

		// Read source content
		var sourceContent []byte
		var err error
		sourceContent, err = os.ReadFile(dep.SourcePath)

		if err != nil {
			fmt.Printf("Warning: Failed to read include file %s: %v\n", dep.SourcePath, err)
			continue
		}

		// Check if target file already exists
		if existingContent, err := os.ReadFile(targetPath); err == nil {
			// File exists, compare contents
			if string(existingContent) == string(sourceContent) {
				// Contents are the same, skip
				continue
			}

			// Contents are different
			if !force {
				fmt.Printf("Include file %s already exists with different content, skipping (use --force to overwrite)\n", dep.TargetPath)
				continue
			}

			// Force is enabled, overwrite
			fmt.Printf("Overwriting existing include file: %s\n", dep.TargetPath)
		}

		// Write to target
		if err := os.WriteFile(targetPath, sourceContent, 0644); err != nil {
			return fmt.Errorf("failed to write include file %s: %w", targetPath, err)
		}

		fmt.Printf("Copied include file: %s\n", dep.TargetPath)
	}

	return nil
}

// InstallPackage installs agent workflows from a GitHub repository
func InstallPackage(repoSpec string, local bool, verbose bool) error {
	if verbose {
		fmt.Printf("Installing package: %s\n", repoSpec)
	}

	// Parse repository specification (org/repo[@version])
	repo, version, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	if verbose {
		fmt.Printf("Repository: %s\n", repo)
		if version != "" {
			fmt.Printf("Version: %s\n", version)
		} else {
			fmt.Printf("Version: main (default)\n")
		}
	}

	// Get packages directory based on local flag
	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Printf("Installing to local packages directory: %s\n", packagesDir)
		} else {
			fmt.Printf("Installing to global packages directory: %s\n", packagesDir)
		}
	}

	// Create packages directory
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create packages directory: %w", err)
	}

	// Create target directory for this repository
	targetDir := filepath.Join(packagesDir, repo)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Check if package already exists
	if _, err := os.Stat(targetDir); err == nil {
		entries, err := os.ReadDir(targetDir)
		if err == nil && len(entries) > 0 {
			fmt.Printf("Package %s already exists. Updating...\n", repo)
			// Remove existing content
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("failed to remove existing package: %w", err)
			}
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to recreate package directory: %w", err)
			}
		}
	}

	// Download workflows from the repository
	if err := downloadWorkflows(repo, version, targetDir, verbose); err != nil {
		return fmt.Errorf("failed to download workflows: %w", err)
	}

	fmt.Printf("Successfully installed package: %s\n", repo)
	return nil
}

// UninstallPackage removes an installed package
func UninstallPackage(repoSpec string, local bool, verbose bool) error {
	if verbose {
		fmt.Printf("Uninstalling package: %s\n", repoSpec)
	}

	// Parse repository specification (only org/repo part, ignore version)
	repo, _, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	// Get packages directory based on local flag
	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Printf("Uninstalling from local packages directory: %s\n", packagesDir)
		} else {
			fmt.Printf("Uninstalling from global packages directory: %s\n", packagesDir)
		}
	}

	// Check if package exists
	targetDir := filepath.Join(packagesDir, repo)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("Package %s is not installed.\n", repo)
		return nil
	}

	// Remove the package directory
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	fmt.Printf("Successfully uninstalled package: %s\n", repo)
	return nil
}

// ListPackages lists all installed packages
func ListPackages(local bool, verbose bool) error {
	if verbose {
		fmt.Printf("Listing installed packages...\n")
	}

	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Printf("Looking in local packages directory: %s\n", packagesDir)
		} else {
			fmt.Printf("Looking in global packages directory: %s\n", packagesDir)
		}
	}

	if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
		if local {
			fmt.Println("No local packages directory found.")
		} else {
			fmt.Println("No global packages directory found.")
		}
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	// Find all installed packages
	packages, err := findInstalledPackages(packagesDir)
	if err != nil {
		return fmt.Errorf("failed to scan packages: %w", err)
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed.")
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	for _, pkg := range packages {
		count := len(pkg.Workflows)
		if pkg.CommitSHA != "" {
			// Truncate commit SHA to first 8 characters for display
			shortSHA := pkg.CommitSHA
			if len(shortSHA) > 8 {
				shortSHA = shortSHA[:8]
			}
			if count == 1 {
				fmt.Printf("%s@%s (%d agent)\n", pkg.Name, shortSHA, count)
			} else {
				fmt.Printf("%s@%s (%d agents)\n", pkg.Name, shortSHA, count)
			}
		} else {
			if count == 1 {
				fmt.Printf("%s (%d agent)\n", pkg.Name, count)
			} else {
				fmt.Printf("%s (%d agents)\n", pkg.Name, count)
			}
		}

		if verbose {
			fmt.Printf("  Location: %s\n", pkg.Path)
			fmt.Printf("  Workflows:\n")
			for _, workflow := range pkg.Workflows {
				fmt.Printf("    - %s\n", workflow)
			}
			fmt.Println()
		}
	}

	return nil
}

// Package represents an installed package
type Package struct {
	Name      string
	Path      string
	Workflows []string
	CommitSHA string
}

// parseRepoSpec parses repository specification like "org/repo@version" or "org/repo@branch" or "org/repo@commit"
func parseRepoSpec(repoSpec string) (repo, version string, err error) {
	parts := strings.SplitN(repoSpec, "@", 2)
	repo = parts[0]

	// Validate repository format (org/repo)
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return "", "", fmt.Errorf("repository must be in format 'org/repo'")
	}

	if len(parts) == 2 {
		version = parts[1]
	}

	return repo, version, nil
}

// downloadWorkflows downloads all .md files from the workflows directory of a GitHub repository
func downloadWorkflows(repo, version, targetDir string, verbose bool) error {
	if verbose {
		fmt.Printf("Downloading workflows from %s/workflows...\n", repo)
	}

	// Create a temporary directory for cloning
	tempDir, err := os.MkdirTemp("", "gh-aw-clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare clone arguments
	cloneArgs := []string{"repo", "clone", repo, tempDir}
	if version != "" && version != "main" {
		cloneArgs = append(cloneArgs, "--", "--branch", version)
	}

	if verbose {
		fmt.Printf("Cloning repository: gh %s\n", strings.Join(cloneArgs, " "))
	}

	// Clone the repository
	_, stdErr, err := gh.Exec(cloneArgs...)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w (stderr: %s)", err, stdErr.String())
	}

	// Get the current commit SHA from the cloned repository
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	commitBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit SHA: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitBytes))

	if verbose {
		fmt.Printf("Repository commit SHA: %s\n", commitSHA)
	}

	// Check if workflows directory exists
	workflowsDir := filepath.Join(tempDir, "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("workflows directory not found in repository %s", repo)
	}

	// Copy all .md files from workflows directory to target
	if err := copyMarkdownFiles(workflowsDir, targetDir, verbose); err != nil {
		return err
	}

	// Store the commit SHA in a metadata file
	metadataFile := filepath.Join(targetDir, ".aw-metadata")
	metadataContent := fmt.Sprintf("commit_sha=%s\n", commitSHA)
	if err := os.WriteFile(metadataFile, []byte(metadataContent), 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	if verbose {
		fmt.Printf("Stored commit SHA in metadata file: %s\n", metadataFile)
	}

	return nil
}

// copyMarkdownFiles recursively copies markdown files from source to target directory
func copyMarkdownFiles(sourceDir, targetDir string, verbose bool) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a markdown file
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create target file path
		targetFile := filepath.Join(targetDir, relPath)

		// Create target directory if needed
		targetFileDir := filepath.Dir(targetFile)
		if err := os.MkdirAll(targetFileDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %w", targetFileDir, err)
		}

		// Copy file
		if verbose {
			fmt.Printf("Copying: %s -> %s\n", relPath, targetFile)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read source file %s: %w", path, err)
		}

		if err := os.WriteFile(targetFile, content, 0644); err != nil {
			return fmt.Errorf("failed to write target file %s: %w", targetFile, err)
		}

		return nil
	})
}

// findInstalledPackages finds all installed packages
func findInstalledPackages(packagesDir string) ([]Package, error) {
	var packages []Package

	// Walk through the packages directory
	err := filepath.Walk(packagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root packages directory
		if path == packagesDir {
			return nil
		}

		// Look for org/repo directory structure
		relPath, err := filepath.Rel(packagesDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) == 2 && info.IsDir() {
			// This is an org/repo directory
			packageName := filepath.Join(parts[0], parts[1])

			// Find workflows in this package
			workflows, err := findWorkflowsInPackage(path)
			if err != nil {
				return err
			}

			// Read commit SHA from metadata file
			commitSHA := readCommitSHAFromMetadata(path)

			packages = append(packages, Package{
				Name:      packageName,
				Path:      path,
				Workflows: workflows,
				CommitSHA: commitSHA,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort packages by name
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return packages, nil
}

// findWorkflowsInPackage finds all workflow files in a package directory
// Only includes files at the top level, excluding files in subdirectories (components)
func findWorkflowsInPackage(packageDir string) ([]string, error) {
	var workflows []string

	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, err := filepath.Rel(packageDir, path)
			if err != nil {
				return err
			}

			// Only include files at the top level (no subdirectories)
			if !strings.Contains(relPath, string(filepath.Separator)) {
				workflows = append(workflows, strings.TrimSuffix(relPath, ".md"))
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(workflows)
	return workflows, nil
}

// WorkflowSourceInfo contains information about where a workflow was found
type WorkflowSourceInfo struct {
	IsPackage          bool
	PackagePath        string
	QualifiedName      string
	NeedsQualifiedName bool
	SourcePath         string
}

// findAndReadWorkflow finds and reads a workflow from multiple sources
func findAndReadWorkflow(workflowPath, workflowsDir string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	if verbose {
		fmt.Printf("Looking for workflow: %s\n", workflowPath)
		fmt.Printf("Using workflows directory: %s\n", workflowsDir)
	}

	//Try local workflows (existing behavior)
	content, path, err := readWorkflowFile(workflowPath, workflowsDir)
	if err == nil {
		if verbose {
			fmt.Printf("Found workflow in local components\n")
		}
		return content, &WorkflowSourceInfo{
			IsPackage:  false,
			SourcePath: path,
		}, nil
	}

	// If not found in local, try packages
	if verbose {
		fmt.Printf("Workflow not found in local .github/workflows or local components, searching packages...\n")
	}

	return findWorkflowInPackages(workflowPath, verbose)
}

// findWorkflowInPackages searches for a workflow in installed packages
func findWorkflowInPackages(workflowPath string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	// Try both local and global packages
	locations := []bool{true, false} // local first, then global

	// Remove .md extension if present for searching
	workflowName := strings.TrimSuffix(workflowPath, ".md")

	for _, local := range locations {
		packagesDir, err := getPackagesDir(local)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to get packages directory (local=%v): %v\n", local, err)
			}
			continue
		}

		locationName := "global"
		if local {
			locationName = "local"
		}

		if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
			if verbose {
				fmt.Printf("No %s packages directory found at %s\n", locationName, packagesDir)
			}
			continue
		}

		if verbose {
			fmt.Printf("Searching %s packages in %s for workflow: %s\n", locationName, packagesDir, workflowName)
		}

		// Check if workflow name contains org/repo prefix
		if strings.Contains(workflowName, "/") {
			// Fully qualified name: org/repo/workflow_name
			content, sourceInfo, err := findQualifiedWorkflowInPackages(workflowName, packagesDir, verbose)
			if err == nil {
				return content, sourceInfo, nil
			}
			if verbose {
				fmt.Printf("Qualified workflow not found in %s packages: %v\n", locationName, err)
			}
		} else {
			// Simple name: workflow_name - search all packages
			content, sourceInfo, err := findUnqualifiedWorkflowInPackages(workflowName, packagesDir, verbose)
			if err == nil {
				return content, sourceInfo, nil
			}
			if verbose {
				fmt.Printf("Unqualified workflow not found in %s packages: %v\n", locationName, err)
			}
		}
	}

	return nil, nil, fmt.Errorf("workflow not found in components and no packages installed")
}

// findQualifiedWorkflowInPackages finds a workflow using fully qualified name
func findQualifiedWorkflowInPackages(qualifiedName, packagesDir string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	parts := strings.Split(qualifiedName, "/")
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("qualified workflow name must be in format 'org/repo/workflow_name'")
	}

	org := parts[0]
	repo := parts[1]
	workflowName := strings.Join(parts[2:], "/") // Support nested workflows

	packagePath := filepath.Join(packagesDir, org, repo)
	workflowFile := filepath.Join(packagePath, workflowName+".md")

	if verbose {
		fmt.Printf("Looking for qualified workflow: %s\n", workflowFile)
	}

	content, err := os.ReadFile(workflowFile)
	if err != nil {
		return nil, nil, fmt.Errorf("workflow '%s' not found in package '%s/%s'", workflowName, org, repo)
	}

	// Check if there would be a conflict with existing workflows in .github/workflows
	simpleFilename := workflowName
	if strings.Contains(workflowName, "/") {
		// For nested workflows, use just the last part as the simple name
		parts := strings.Split(workflowName, "/")
		simpleFilename = parts[len(parts)-1]
	}

	return content, &WorkflowSourceInfo{
		IsPackage:          true,
		PackagePath:        packagePath,
		QualifiedName:      simpleFilename,
		NeedsQualifiedName: false,
		SourcePath:         workflowFile,
	}, nil
}

// findUnqualifiedWorkflowInPackages finds a workflow by name across all packages
func findUnqualifiedWorkflowInPackages(workflowName, packagesDir string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	var matches []WorkflowMatch

	// Search all packages for workflows with this name
	err := filepath.Walk(packagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			// Check if this is the workflow we're looking for
			baseName := strings.TrimSuffix(info.Name(), ".md")
			if baseName == workflowName {
				// Get package info from path
				relPath, err := filepath.Rel(packagesDir, path)
				if err != nil {
					return err
				}

				pathParts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
				if len(pathParts) >= 2 {
					org := pathParts[0]
					repo := pathParts[1]
					matches = append(matches, WorkflowMatch{
						Path:        path,
						PackageName: fmt.Sprintf("%s/%s", org, repo),
						Org:         org,
						Repo:        repo,
					})
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error searching packages: %w", err)
	}

	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("workflow '%s' not found in any package", workflowName)
	}

	if len(matches) == 1 {
		// Single match, use it
		match := matches[0]
		content, err := os.ReadFile(match.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read workflow file: %w", err)
		}

		if verbose {
			fmt.Printf("Found workflow '%s' in package '%s'\n", workflowName, match.PackageName)
		}

		return content, &WorkflowSourceInfo{
			IsPackage:          true,
			PackagePath:        filepath.Dir(match.Path),
			QualifiedName:      workflowName,
			NeedsQualifiedName: false,
			SourcePath:         match.Path,
		}, nil
	}

	// Multiple matches - require disambiguation
	fmt.Printf("Multiple workflows named '%s' found:\n", workflowName)
	for _, match := range matches {
		fmt.Printf("  - %s/%s\n", match.PackageName, workflowName)
	}
	fmt.Printf("\nPlease specify the full path: "+constants.CLIExtensionPrefix+" add <org/repo/%s>\n", workflowName)
	fmt.Printf("Or use a different name: "+constants.CLIExtensionPrefix+" add %s -n <custom-name>\n", workflowName)

	return nil, nil, fmt.Errorf("ambiguous workflow name - specify full path or use -n flag for custom name")
}

// WorkflowMatch represents a workflow match in package search
type WorkflowMatch struct {
	Path        string
	PackageName string
	Org         string
	Repo        string
}

// collectIncludeDependenciesFromSource collects include dependencies based on source type
func collectIncludeDependenciesFromSource(content string, sourceInfo *WorkflowSourceInfo, verbose bool) ([]IncludeDependency, error) {
	if sourceInfo.IsPackage {
		// For package sources, use package-aware dependency collection
		return collectPackageIncludeDependencies(content, sourceInfo.PackagePath, verbose)
	}

	workflowsDir := getWorkflowsDir()

	return collectIncludeDependencies(content, sourceInfo.SourcePath, workflowsDir)
}

// collectPackageIncludeDependencies collects dependencies for package-based workflows
func collectPackageIncludeDependencies(content, packagePath string, verbose bool) ([]IncludeDependency, error) {
	var dependencies []IncludeDependency
	seen := make(map[string]bool)

	if verbose {
		fmt.Printf("Collecting package dependencies from: %s\n", packagePath)
	}

	err := collectPackageIncludesRecursive(content, packagePath, &dependencies, seen, verbose)
	return dependencies, err
}

// collectPackageIncludesRecursive recursively processes @include directives in package content
func collectPackageIncludesRecursive(content, baseDir string, dependencies *[]IncludeDependency, seen map[string]bool, verbose bool) error {
	includePattern := regexp.MustCompile(`^@include\s+(.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := includePattern.FindStringSubmatch(line); matches != nil {
			includePath := strings.TrimSpace(matches[1])

			// Handle section references (file.md#Section)
			var filePath string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
			} else {
				filePath = includePath
			}

			// Resolve the full source path relative to base directory
			fullSourcePath := filepath.Join(baseDir, filePath)

			// Skip if we've already processed this file
			if seen[fullSourcePath] {
				continue
			}
			seen[fullSourcePath] = true

			// Add dependency
			dep := IncludeDependency{
				SourcePath: fullSourcePath,
				TargetPath: filePath, // Keep relative path for target
			}
			*dependencies = append(*dependencies, dep)

			if verbose {
				fmt.Printf("Found include dependency: %s -> %s\n", fullSourcePath, filePath)
			}

			// Read the included file and process its includes recursively
			includedContent, err := os.ReadFile(fullSourcePath)
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Could not read include file %s: %v\n", fullSourcePath, err)
				}
				continue
			}

			// Extract markdown content from the included file
			markdownContent, err := parser.ExtractMarkdownContent(string(includedContent))
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Could not extract markdown from %s: %v\n", fullSourcePath, err)
				}
				continue
			}

			// Recursively process includes in the included file
			includedDir := filepath.Dir(fullSourcePath)
			if err := collectPackageIncludesRecursive(markdownContent, includedDir, dependencies, seen, verbose); err != nil {
				if verbose {
					fmt.Printf("Warning: Error processing includes in %s: %v\n", fullSourcePath, err)
				}
			}
		}
	}

	return scanner.Err()
}

// copyIncludeDependenciesFromSourceWithForce copies dependencies based on source type with force option
func copyIncludeDependenciesFromSourceWithForce(dependencies []IncludeDependency, githubWorkflowsDir string, sourceInfo *WorkflowSourceInfo, verbose bool, force bool) error {
	if sourceInfo.IsPackage {
		// For package sources, copy from local filesystem
		return copyIncludeDependenciesFromPackageWithForce(dependencies, githubWorkflowsDir, verbose, force)
	}
	return copyIncludeDependenciesWithForce(dependencies, githubWorkflowsDir, force)
}

// copyIncludeDependenciesFromPackageWithForce copies include dependencies from package filesystem with force option
func copyIncludeDependenciesFromPackageWithForce(dependencies []IncludeDependency, githubWorkflowsDir string, verbose bool, force bool) error {
	for _, dep := range dependencies {
		// Create the target path in .github/workflows
		targetPath := filepath.Join(githubWorkflowsDir, dep.TargetPath)

		// Create target directory if it doesn't exist
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}

		// Read source content from package
		sourceContent, err := os.ReadFile(dep.SourcePath)
		if err != nil {
			fmt.Printf("Warning: Failed to read include file %s: %v\n", dep.SourcePath, err)
			continue
		}

		// Check if target file already exists
		if existingContent, err := os.ReadFile(targetPath); err == nil {
			// File exists, compare contents
			if string(existingContent) == string(sourceContent) {
				// Contents are the same, skip
				if verbose {
					fmt.Printf("Include file %s already exists with same content, skipping\n", dep.TargetPath)
				}
				continue
			}

			// Contents are different
			if !force {
				fmt.Printf("Include file %s already exists with different content, skipping (use --force to overwrite)\n", dep.TargetPath)
				continue
			}

			// Force is enabled, overwrite
			fmt.Printf("Overwriting existing include file: %s\n", dep.TargetPath)
		}

		// Write to target
		if err := os.WriteFile(targetPath, sourceContent, 0644); err != nil {
			return fmt.Errorf("failed to write include file %s: %w", targetPath, err)
		}

		if verbose {
			fmt.Printf("Copied include file: %s -> %s\n", dep.SourcePath, targetPath)
		}
	}

	return nil
}

// cleanupOrphanedIncludes removes include files that are no longer used by any workflow
func cleanupOrphanedIncludes(verbose bool) error {
	// Get all remaining markdown files
	mdFiles, err := getMarkdownWorkflowFiles()
	if err != nil {
		// No markdown files means we can clean up all includes
		if verbose {
			fmt.Printf("No markdown files found, cleaning up all includes\n")
		}
		return cleanupAllIncludes(verbose)
	}

	// Collect all include dependencies from remaining workflows
	usedIncludes := make(map[string]bool)

	for _, mdFile := range mdFiles {
		content, err := os.ReadFile(mdFile)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Could not read %s for include analysis: %v\n", mdFile, err)
			}
			continue
		}

		// Find includes used by this workflow
		includes, err := findIncludesInContent(string(content), filepath.Dir(mdFile), verbose)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Could not analyze includes in %s: %v\n", mdFile, err)
			}
			continue
		}

		for _, include := range includes {
			usedIncludes[include] = true
		}
	}

	// Find all include files in .github/workflows
	workflowsDir := ".github/workflows"
	var allIncludes []string

	err = filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			// This is an include file
			relPath, err := filepath.Rel(workflowsDir, path)
			if err != nil {
				return err
			}
			allIncludes = append(allIncludes, relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan include files: %w", err)
	}

	// Remove unused includes
	for _, include := range allIncludes {
		if !usedIncludes[include] {
			includePath := filepath.Join(workflowsDir, include)
			if err := os.Remove(includePath); err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to remove orphaned include %s: %v\n", include, err)
				}
			} else {
				fmt.Printf("Removed orphaned include: %s\n", include)
			}
		}
	}

	return nil
}

// cleanupAllIncludes removes all include files when no workflows remain
func cleanupAllIncludes(verbose bool) error {
	workflowsDir := ".github/workflows"

	err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			if err := os.Remove(path); err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to remove include %s: %v\n", path, err)
				}
			} else {
				relPath, _ := filepath.Rel(workflowsDir, path)
				fmt.Printf("Removed include: %s\n", relPath)
			}
		}

		return nil
	})

	return err
}

// findIncludesInContent finds all @include references in content
func findIncludesInContent(content, baseDir string, verbose bool) ([]string, error) {
	_ = baseDir // unused parameter for now, keeping for potential future use
	_ = verbose // unused parameter for now, keeping for potential future use
	var includes []string
	includePattern := regexp.MustCompile(`^@include\s+(.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := includePattern.FindStringSubmatch(line); matches != nil {
			includePath := strings.TrimSpace(matches[1])

			// Handle section references (file.md#Section)
			var filePath string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
			} else {
				filePath = includePath
			}

			includes = append(includes, filePath)
		}
	}

	return includes, scanner.Err()
}

// listPackageWorkflows lists workflows from installed packages
func listPackageWorkflows(verbose bool) error {
	// Check both local and global packages
	locations := []bool{true, false} // local first, then global
	var allPackages []Package

	for _, local := range locations {
		packagesDir, err := getPackagesDir(local)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to get packages directory (local=%v): %v\n", local, err)
			}
			continue
		}

		locationName := "global"
		if local {
			locationName = "local"
		}

		if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
			if verbose {
				fmt.Printf("No %s packages directory found at %s\n", locationName, packagesDir)
			}
			continue
		}

		if verbose {
			fmt.Printf("Searching for workflows in %s packages...\n", locationName)
		}

		// Find all installed packages
		packages, err := findInstalledPackages(packagesDir)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to scan %s packages: %v\n", locationName, err)
			}
			continue
		}

		// Mark packages with their location
		for i := range packages {
			if local {
				packages[i].Name = packages[i].Name + " (local)"
			} else {
				packages[i].Name = packages[i].Name + " (global)"
			}
		}

		allPackages = append(allPackages, packages...)
	}

	if len(allPackages) == 0 {
		fmt.Println("No workflows or packages found.")
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	fmt.Println("Available workflows from packages:")
	fmt.Println("==================================")

	for _, pkg := range allPackages {
		if verbose {
			fmt.Printf("Package: %s\n", pkg.Name)
		}

		for _, workflow := range pkg.Workflows {
			// Read the workflow file to get its title
			workflowFile := filepath.Join(pkg.Path, workflow+".md")
			workflowName, err := extractWorkflowNameFromFile(workflowFile)
			if err != nil || workflowName == "" {
				fmt.Printf("  %-30s (from %s)\n", workflow, pkg.Name)
			} else {
				fmt.Printf("  %-30s - %s (from %s)\n", workflow, workflowName, pkg.Name)
			}
		}
	}

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add <workflow>      - Add workflow from any package")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add <workflow> -n <name> - Add workflow with specific name")
	fmt.Println("  " + constants.CLIExtensionPrefix + " list --packages     - List installed packages")

	return nil
}

// readCommitSHAFromMetadata reads the commit SHA from the package metadata file
func readCommitSHAFromMetadata(packagePath string) string {
	metadataFile := filepath.Join(packagePath, ".aw-metadata")
	content, err := os.ReadFile(metadataFile)
	if err != nil {
		return "" // No metadata file or error reading it
	}

	// Parse the metadata file for commit_sha=<value>
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if commitSHA, found := strings.CutPrefix(line, "commit_sha="); found {
			return commitSHA
		}
	}

	return "" // commit_sha not found in metadata
}

// resolveWorkflowFile resolves a file or workflow name to an actual file path
func resolveWorkflowFile(fileOrWorkflowName string, verbose bool) (string, error) {
	// First, try to use it as a direct file path
	if _, err := os.Stat(fileOrWorkflowName); err == nil {
		if verbose {
			fmt.Printf("Found workflow file at path: %s\n", fileOrWorkflowName)
		}
		return fileOrWorkflowName, nil
	}

	// If it's not a direct file path, try to resolve it as a workflow name
	if verbose {
		fmt.Printf("File not found at %s, trying to resolve as workflow name...\n", fileOrWorkflowName)
	}

	// Add .md extension if not present
	workflowPath := fileOrWorkflowName
	if !strings.HasSuffix(workflowPath, ".md") {
		workflowPath += ".md"
	}

	if verbose {
		fmt.Printf("Looking for workflow file: %s\n", workflowPath)
	}

	workflowsDir := getWorkflowsDir()

	// Try to find the workflow from multiple sources
	sourceContent, sourceInfo, err := findAndReadWorkflow(workflowPath, workflowsDir, verbose)
	if err != nil {
		return "", fmt.Errorf("workflow '%s' not found in local .github/workflows, components or packages", fileOrWorkflowName)
	}

	// If we found the workflow in packages,
	if sourceInfo.IsPackage {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "workflow-*.md")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary file: %w", err)
		}

		if _, err := tmpFile.Write(sourceContent); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("failed to write temporary file: %w", err)
		}
		tmpFile.Close()

		if verbose {
			fmt.Printf("Created temporary workflow file: %s\n", tmpFile.Name())
		}

		return tmpFile.Name(), nil
	} else {
		// It's a local file, return the source path
		return sourceInfo.SourcePath, nil
	}
}

// RunWorkflowOnGitHub runs an agentic workflow on GitHub Actions
func RunWorkflowOnGitHub(workflowIdOrName string, verbose bool) error {
	if workflowIdOrName == "" {
		return fmt.Errorf("workflow name or ID is required")
	}

	if verbose {
		fmt.Printf("Running workflow on GitHub Actions: %s\n", workflowIdOrName)
	}

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is required but not available")
	}

	// Try to resolve the workflow file path to find the corresponding .lock.yml file
	workflowFile, err := resolveWorkflowFile(workflowIdOrName, verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve workflow: %w", err)
	}

	// Check if we created a temporary file that needs cleanup
	if strings.HasPrefix(workflowFile, os.TempDir()) {
		defer func() {
			if err := os.Remove(workflowFile); err != nil && verbose {
				fmt.Printf("Warning: Failed to clean up temporary file %s: %v\n", workflowFile, err)
			}
		}()
	}

	// Check if the workflow is runnable (has workflow_dispatch trigger)
	runnable, err := IsRunnable(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to check if workflow %s is runnable: %w", workflowFile, err)
	}

	if !runnable {
		return fmt.Errorf("workflow '%s' cannot be run on GitHub Actions - it must have 'workflow_dispatch' trigger", workflowIdOrName)
	}

	// Determine the lock file name based on the workflow source
	var lockFileName string

	// Always resolve the workflow to get source info for proper lock file naming
	workflowsDir := getWorkflowsDir()

	_, sourceInfo, err := findAndReadWorkflow(workflowIdOrName+".md", workflowsDir, verbose)
	if err != nil {
		return fmt.Errorf("failed to find workflow source info: %w", err)
	}

	filename := strings.TrimSuffix(filepath.Base(workflowIdOrName), ".md")
	if sourceInfo.IsPackage && sourceInfo.NeedsQualifiedName {
		// For package workflows that need qualified names, use the qualified name
		filename = sourceInfo.QualifiedName
	} else if sourceInfo.IsPackage {
		// For package workflows that don't need qualified names but are from packages,
		// we need to check what lock file actually exists
		// Try the unqualified name first, then fall back to checking existing lock files
		unqualifiedLock := filename + ".lock.yml"
		unqualifiedPath := filepath.Join(".github/workflows", unqualifiedLock)

		if _, err := os.Stat(unqualifiedPath); os.IsNotExist(err) {
			// Look for any lock file that might match this workflow from packages
			if foundLock := findMatchingLockFile(filename, verbose); foundLock != "" {
				filename = strings.TrimSuffix(foundLock, ".lock.yml")
			}
		}
	}
	lockFileName = filename + ".lock.yml"

	// Check if the lock file exists in .github/workflows
	lockFilePath := filepath.Join(".github/workflows", lockFileName)
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		return fmt.Errorf("workflow lock file '%s' not found in .github/workflows - run '"+constants.CLIExtensionPrefix+" compile' first", lockFileName)
	}

	if verbose {
		fmt.Printf("Using lock file: %s\n", lockFileName)
	}

	// Execute gh workflow run command and capture output
	cmd := exec.Command("gh", "workflow", "run", lockFileName)

	if verbose {
		fmt.Printf("Executing: gh workflow run %s\n", lockFileName)
	}

	// Capture both stdout and stderr
	stdout, err := cmd.Output()
	if err != nil {
		// If there's an error, try to get stderr for better error reporting
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "%s", exitError.Stderr)
		}
		return fmt.Errorf("failed to run workflow on GitHub Actions: %w", err)
	}

	// Display the output from gh workflow run
	output := strings.TrimSpace(string(stdout))
	if output != "" {
		fmt.Println(output)
	}

	fmt.Printf("Successfully triggered workflow: %s\n", lockFileName)

	// Try to get the latest run for this workflow to show a direct link
	if runURL, err := getLatestWorkflowRunURL(lockFileName, verbose); err == nil && runURL != "" {
		fmt.Printf("\n🔗 View workflow run: %s\n", runURL)
	} else if verbose && err != nil {
		fmt.Printf("Note: Could not get workflow run URL: %v\n", err)
	}

	return nil
}

// IsRunnable checks if a workflow can be run locally (has schedule or workflow_dispatch trigger)
func IsRunnable(markdownPath string) (bool, error) {
	// Read the file
	contentBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// Extract frontmatter
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return false, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Check if 'on' section is present
	onSection, exists := result.Frontmatter["on"]
	if !exists {
		// If no 'on' section, it defaults to runnable triggers (schedule, workflow_dispatch)
		return true, nil
	}

	// Convert to string to analyze
	onStr := fmt.Sprintf("%v", onSection)
	onStrLower := strings.ToLower(onStr)

	// Check for schedule or workflow_dispatch triggers
	hasSchedule := strings.Contains(onStrLower, "schedule") || strings.Contains(onStrLower, "cron")
	hasWorkflowDispatch := strings.Contains(onStrLower, "workflow_dispatch")

	return hasSchedule || hasWorkflowDispatch, nil
}

// findMatchingLockFile searches for existing lock files that might match the given workflow name
func findMatchingLockFile(workflowName string, verbose bool) string {
	workflowsDir := getWorkflowsDir()

	// Look for any .lock.yml files that might correspond to this workflow
	lockFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	if err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to search for lock files: %v\n", err)
		}
		return ""
	}

	if verbose {
		fmt.Printf("Searching for lock files matching workflow '%s'\n", workflowName)
	}

	// Look for exact matches first, then partial matches
	for _, lockFile := range lockFiles {
		baseName := filepath.Base(lockFile)
		lockName := strings.TrimSuffix(baseName, ".lock.yml")

		// Check if the lock file ends with the workflow name (for qualified names)
		if strings.HasSuffix(lockName, "_"+workflowName) {
			if verbose {
				fmt.Printf("Found matching lock file (suffix match): %s\n", baseName)
			}
			return baseName
		}
	}

	// If no suffix match, look for any lock file containing the workflow name
	for _, lockFile := range lockFiles {
		baseName := filepath.Base(lockFile)
		lockName := strings.TrimSuffix(baseName, ".lock.yml")

		if strings.Contains(lockName, workflowName) {
			if verbose {
				fmt.Printf("Found matching lock file (contains match): %s\n", baseName)
			}
			return baseName
		}
	}

	if verbose {
		fmt.Printf("No matching lock file found for workflow '%s'\n", workflowName)
	}
	return ""
}

// getLatestWorkflowRunURL gets the URL for the most recent run of the specified workflow
func getLatestWorkflowRunURL(lockFileName string, verbose bool) (string, error) {
	if verbose {
		fmt.Printf("Getting latest run URL for workflow: %s\n", lockFileName)
	}

	// Start spinner for network operation
	spinner := console.NewSpinner("Getting latest workflow run...")
	if !verbose {
		spinner.Start()
	}

	// Get the most recent run for this workflow
	cmd := exec.Command("gh", "run", "list", "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion")
	output, err := cmd.Output()

	// Stop spinner
	if !verbose {
		spinner.Stop()
	}

	if err != nil {
		return "", fmt.Errorf("failed to get workflow runs: %w", err)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("no runs found for workflow")
	}

	// Parse the JSON output
	var runs []struct {
		URL        string `json:"url"`
		DatabaseID int64  `json:"databaseId"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	}

	if err := json.Unmarshal(output, &runs); err != nil {
		return "", fmt.Errorf("failed to parse workflow run data: %w", err)
	}

	if len(runs) == 0 {
		return "", fmt.Errorf("no runs found")
	}

	run := runs[0]
	if verbose {
		fmt.Printf("Found run %d with status: %s\n", run.DatabaseID, run.Status)
	}

	return run.URL, nil
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

// getCurrentBranch gets the current git branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("could not determine current branch")
	}

	return branch, nil
}

// createAndSwitchBranch creates a new branch and switches to it
func createAndSwitchBranch(branchName string, verbose bool) error {
	if verbose {
		fmt.Printf("Creating and switching to branch: %s\n", branchName)
	}

	cmd := exec.Command("git", "checkout", "-b", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create and switch to branch %s: %w", branchName, err)
	}

	return nil
}

// switchBranch switches to the specified branch
func switchBranch(branchName string, verbose bool) error {
	if verbose {
		fmt.Printf("Switching to branch: %s\n", branchName)
	}

	cmd := exec.Command("git", "checkout", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	return nil
}

// commitChanges commits all staged changes with the given message
func commitChanges(message string, verbose bool) error {
	if verbose {
		fmt.Printf("Committing changes with message: %s\n", message)
	}

	cmd := exec.Command("git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// pushBranch pushes the specified branch to origin
func pushBranch(branchName string, verbose bool) error {
	if verbose {
		fmt.Printf("Pushing branch: %s\n", branchName)
	}

	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	return nil
}

// createPR creates a pull request using GitHub CLI
func createPR(branchName, title, body string, verbose bool) error {
	if verbose {
		fmt.Printf("Creating PR: %s\n", title)
	}

	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body, "--head", branchName)
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

// NewWorkflow creates a new workflow markdown file with template content
func NewWorkflow(workflowName string, verbose bool, force bool) error {
	if verbose {
		fmt.Printf("Creating new workflow: %s\n", workflowName)
	}

	// Get current working directory for .github/workflows
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	githubWorkflowsDir := filepath.Join(workingDir, ".github", "workflows")
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Construct the destination file path
	destFile := filepath.Join(githubWorkflowsDir, workflowName+".md")

	// Check if destination file already exists
	if _, err := os.Stat(destFile); err == nil && !force {
		return fmt.Errorf("workflow file '%s' already exists. Use --force to overwrite", destFile)
	}

	// Create the template content
	template := createWorkflowTemplate(workflowName)

	// Write the template to file
	if err := os.WriteFile(destFile, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file '%s': %w", destFile, err)
	}

	fmt.Printf("Created new workflow: %s\n", destFile)
	fmt.Printf("Edit the file to customize your workflow, then run '" + constants.CLIExtensionPrefix + " compile' to generate the GitHub Actions workflow.\n")

	return nil
}

// createWorkflowTemplate generates a concise workflow template with essential options
func createWorkflowTemplate(workflowName string) string {
	return `---
# Trigger - when should this workflow run?
on:
  workflow_dispatch:  # Manual trigger

# Alternative triggers (uncomment to use):
# on:
#   issues:
#     types: [opened, reopened]
#   pull_request:
#     types: [opened, synchronize]
#   schedule:
#     - cron: "0 9 * * 1"  # Every Monday at 9 AM UTC

# Permissions - what can this workflow access?
permissions:
  contents: read
  issues: write
  pull-requests: write

# Tools - what APIs and tools can the AI use?
tools:
  github:
    allowed:
      - get_issue
      - add_issue_comment
      - create_issue
      - get_pull_request
      - get_file_contents

# Advanced options (uncomment to use):
# engine: claude  # AI engine (default: claude)
# timeout_minutes: 30  # Max runtime (default: 15)
# runs-on: ubuntu-latest  # Runner type (default: ubuntu-latest)

---

# ` + workflowName + `

Describe what you want the AI to do when this workflow runs.

## Instructions

Replace this section with specific instructions for the AI. For example:

1. Read the issue description and comments
2. Analyze the request and gather relevant information
3. Provide a helpful response or take appropriate action

Be clear and specific about what the AI should accomplish.

## Notes

- Run ` + "`" + constants.CLIExtensionPrefix + " compile`" + ` to generate the GitHub Actions workflow
- See https://github.com/githubnext/gh-aw/blob/main/docs/index.md for complete configuration options and tools documentation
`
}
