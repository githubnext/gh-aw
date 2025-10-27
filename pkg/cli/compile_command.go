package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

var compileLog = logger.New("cli:compile_command")

// CompileWorkflowWithValidation compiles a workflow with always-on YAML validation for CLI usage
func CompileWorkflowWithValidation(compiler *workflow.Compiler, filePath string, verbose bool) error {
	// Compile the workflow first
	if err := compiler.CompileWorkflow(filePath); err != nil {
		return err
	}

	// Always validate that the generated lock file is valid YAML (CLI requirement)
	lockFile := strings.TrimSuffix(filePath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); err != nil {
		// Lock file doesn't exist (likely due to no-emit), skip YAML validation
		return nil
	}

	compileLog.Print("Validating generated lock file YAML syntax")

	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		return fmt.Errorf("failed to read generated lock file for validation: %w", err)
	}

	// Validate the lock file is valid YAML
	var yamlValidationTest any
	if err := yaml.Unmarshal(lockContent, &yamlValidationTest); err != nil {
		return fmt.Errorf("generated lock file is not valid YAML: %w", err)
	}

	return nil
}

// CompileWorkflowDataWithValidation compiles from already-parsed WorkflowData with validation
// This avoids re-parsing when the workflow data has already been parsed
func CompileWorkflowDataWithValidation(compiler *workflow.Compiler, workflowData *workflow.WorkflowData, filePath string, verbose bool) error {
	// Compile the workflow using already-parsed data
	if err := compiler.CompileWorkflowData(workflowData, filePath); err != nil {
		return err
	}

	// Always validate that the generated lock file is valid YAML (CLI requirement)
	lockFile := strings.TrimSuffix(filePath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); err != nil {
		// Lock file doesn't exist (likely due to no-emit), skip YAML validation
		return nil
	}

	compileLog.Print("Validating generated lock file YAML syntax")

	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		return fmt.Errorf("failed to read generated lock file for validation: %w", err)
	}

	// Validate the lock file is valid YAML
	var yamlValidationTest any
	if err := yaml.Unmarshal(lockContent, &yamlValidationTest); err != nil {
		return fmt.Errorf("generated lock file is not valid YAML: %w", err)
	}

	return nil
}

// CompileConfig holds configuration options for compiling workflows
type CompileConfig struct {
	MarkdownFiles        []string // Files to compile (empty for all files)
	Verbose              bool     // Enable verbose output
	EngineOverride       string   // Override AI engine setting
	Validate             bool     // Enable schema validation
	Watch                bool     // Enable watch mode
	WorkflowDir          string   // Custom workflow directory
	SkipInstructions     bool     // Deprecated: Instructions are no longer written during compilation
	NoEmit               bool     // Validate without generating lock files
	Purge                bool     // Remove orphaned lock files
	TrialMode            bool     // Enable trial mode (suppress safe outputs)
	TrialLogicalRepoSlug string   // Target repository for trial mode
	Strict               bool     // Enable strict mode validation
	Dependabot           bool     // Generate Dependabot manifests for npm dependencies
	ForceOverwrite       bool     // Force overwrite of existing files (dependabot.yml)
	Zizmor               bool     // Run zizmor security scanner on generated .lock.yml files
}

// CompilationStats tracks the results of workflow compilation
type CompilationStats struct {
	Total           int
	Errors          int
	Warnings        int
	FailedWorkflows []string // Names of workflows that failed compilation
}

func CompileWorkflows(config CompileConfig) ([]*workflow.WorkflowData, error) {
	markdownFiles := config.MarkdownFiles
	verbose := config.Verbose
	engineOverride := config.EngineOverride
	validate := config.Validate
	watch := config.Watch
	workflowDir := config.WorkflowDir
	noEmit := config.NoEmit
	purge := config.Purge
	trialMode := config.TrialMode
	trialLogicalRepoSlug := config.TrialLogicalRepoSlug
	strict := config.Strict
	dependabot := config.Dependabot
	forceOverwrite := config.ForceOverwrite
	zizmor := config.Zizmor

	compileLog.Printf("Starting workflow compilation: files=%d, validate=%v, watch=%v, noEmit=%v, dependabot=%v, zizmor=%v", len(markdownFiles), validate, watch, noEmit, dependabot, zizmor)

	// Track compilation statistics
	stats := &CompilationStats{}

	// Validate dependabot flag usage
	if dependabot {
		if len(markdownFiles) > 0 {
			return nil, fmt.Errorf("--dependabot flag cannot be used with specific workflow files")
		}
		if workflowDir != "" && workflowDir != ".github/workflows" {
			return nil, fmt.Errorf("--dependabot flag cannot be used with custom --workflows-dir")
		}
	}

	// Validate purge flag usage
	if purge && len(markdownFiles) > 0 {
		return nil, fmt.Errorf("--purge flag can only be used when compiling all markdown files (no specific files specified)")
	}

	// Validate and set default for workflow directory
	if workflowDir == "" {
		workflowDir = ".github/workflows"
		compileLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		// Ensure the path is relative
		if filepath.IsAbs(workflowDir) {
			return nil, fmt.Errorf("workflows-dir must be a relative path, got: %s", workflowDir)
		}
		// Clean the path to avoid issues with ".." or other problematic elements
		workflowDir = filepath.Clean(workflowDir)
		compileLog.Printf("Using custom workflow directory: %s", workflowDir)
	}

	// Create compiler with verbose flag and AI engine override
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	compileLog.Print("Created compiler instance")

	// Set validation based on the validate flag (false by default for compatibility)
	compiler.SetSkipValidation(!validate)

	// Set noEmit flag to validate without generating lock files
	compiler.SetNoEmit(noEmit)

	// Set strict mode if specified
	compiler.SetStrictMode(strict)

	// Set trial mode if specified
	if trialMode {
		compileLog.Printf("Enabling trial mode: repoSlug=%s", trialLogicalRepoSlug)
		compiler.SetTrialMode(true)
		if trialLogicalRepoSlug != "" {
			compiler.SetTrialLogicalRepoSlug(trialLogicalRepoSlug)
		}
	}

	if watch {
		// Watch mode: watch for file changes and recompile automatically
		// For watch mode, we only support a single file for now
		var markdownFile string
		if len(markdownFiles) > 0 {
			if len(markdownFiles) > 1 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Watch mode only supports a single file, using the first one"))
			}
			// Resolve the workflow file to get the full path
			resolvedFile, err := resolveWorkflowFile(markdownFiles[0], verbose)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve workflow '%s': %w", markdownFiles[0], err)
			}
			markdownFile = resolvedFile
		}
		return nil, watchAndCompileWorkflows(markdownFile, compiler, verbose)
	}

	var workflowDataList []*workflow.WorkflowData

	if len(markdownFiles) > 0 {
		// Compile specific workflow files
		var compiledCount int
		var errorCount int
		var errorMessages []string
		for _, markdownFile := range markdownFiles {
			stats.Total++
			// Resolve workflow ID or file path to actual file path
			compileLog.Printf("Resolving workflow file: %s", markdownFile)
			resolvedFile, err := resolveWorkflowFile(markdownFile, verbose)
			if err != nil {
				errMsg := fmt.Sprintf("failed to resolve workflow '%s': %v", markdownFile, err)
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
				errorMessages = append(errorMessages, err.Error())
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, markdownFile)
				continue
			}
			compileLog.Printf("Resolved to: %s", resolvedFile)

			// Parse workflow file to get data
			compileLog.Printf("Parsing workflow file: %s", resolvedFile)
			workflowData, err := compiler.ParseWorkflowFile(resolvedFile)
			if err != nil {
				errMsg := fmt.Sprintf("failed to parse workflow file %s: %v", resolvedFile, err)
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
				errorMessages = append(errorMessages, err.Error())
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(resolvedFile))
				continue
			}
			workflowDataList = append(workflowDataList, workflowData)

			compileLog.Printf("Starting compilation of %s", resolvedFile)
			if err := CompileWorkflowDataWithValidation(compiler, workflowData, resolvedFile, verbose); err != nil {
				// Always put error on a new line and don't wrap with "failed to compile workflow"
				fmt.Fprintln(os.Stderr, err.Error())
				errorMessages = append(errorMessages, err.Error())
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(resolvedFile))
				continue
			}
			compiledCount++
		}

		// Get warning count from compiler
		stats.Warnings = compiler.GetWarningCount()

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d workflow file(s)", compiledCount)))
		}

		// Ensure .gitattributes marks .lock.yml files as generated
		compileLog.Printf("Updating .gitattributes")
		if err := ensureGitAttributes(); err != nil {
			compileLog.Printf("Failed to update .gitattributes: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
			}
		} else {
			compileLog.Printf("Successfully updated .gitattributes")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .gitattributes to mark .lock.yml files as generated"))
			}
		}

		// Generate Dependabot manifests if requested
		if dependabot && !noEmit {
			// Resolve workflow directory path
			absWorkflowDir := workflowDir
			if !filepath.IsAbs(absWorkflowDir) {
				gitRoot, err := findGitRoot()
				if err == nil {
					absWorkflowDir = filepath.Join(gitRoot, workflowDir)
				}
			}

			if err := compiler.GenerateDependabotManifests(workflowDataList, absWorkflowDir, forceOverwrite); err != nil {
				if strict {
					return workflowDataList, fmt.Errorf("failed to generate Dependabot manifests: %w", err)
				}
				// Non-strict mode: just report as warning
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate Dependabot manifests: %v", err)))
			}
		}

		// Note: Instructions are only written by the init command
		// The compile command should not write instruction files

		// Print summary
		printCompilationSummary(stats)

		// Return error if any compilations failed
		if errorCount > 0 {
			// Return the first error message for backward compatibility with tests
			if len(errorMessages) > 0 {
				return workflowDataList, errors.New(errorMessages[0])
			}
			return workflowDataList, fmt.Errorf("compilation failed")
		}

		// Run zizmor security scanner if requested and compilation was successful
		if zizmor && !noEmit {
			// Resolve workflow directory path
			if workflowDir == "" {
				workflowDir = ".github/workflows"
			}
			absWorkflowDir := workflowDir
			if !filepath.IsAbs(absWorkflowDir) {
				gitRoot, err := findGitRoot()
				if err == nil {
					absWorkflowDir = filepath.Join(gitRoot, workflowDir)
				}
			}

			if err := runZizmor(absWorkflowDir, verbose); err != nil {
				return workflowDataList, fmt.Errorf("zizmor security scan failed: %w", err)
			}
		}

		return workflowDataList, nil
	}

	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileLog.Printf("Found git root: %s", gitRoot)

	// Compile all markdown files in the specified workflow directory relative to git root
	workflowsDir := filepath.Join(gitRoot, workflowDir)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}

	compileLog.Printf("Scanning for markdown files in %s", workflowsDir)
	if verbose {
		fmt.Printf("Scanning for markdown files in %s\n", workflowsDir)
	}

	// Find all markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		return nil, fmt.Errorf("no markdown files found in %s", workflowsDir)
	}

	compileLog.Printf("Found %d markdown files to compile", len(mdFiles))
	if verbose {
		fmt.Printf("Found %d markdown files to compile\n", len(mdFiles))
	}

	// Handle purge logic: collect existing .lock.yml files before compilation
	var existingLockFiles []string
	var expectedLockFiles []string
	if purge {
		// Find all existing .lock.yml files
		existingLockFiles, err = filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
		if err != nil {
			return nil, fmt.Errorf("failed to find existing lock files: %w", err)
		}

		// Create expected lock files list based on markdown files
		for _, mdFile := range mdFiles {
			lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"
			expectedLockFiles = append(expectedLockFiles, lockFile)
		}

		if verbose && len(existingLockFiles) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d existing .lock.yml files", len(existingLockFiles))))
		}
	}

	// Compile each file
	var errorCount int
	var successCount int
	for _, file := range mdFiles {
		stats.Total++
		// Parse workflow file to get data
		workflowData, err := compiler.ParseWorkflowFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("failed to parse workflow file %s: %v", file, err)))
			errorCount++
			stats.Errors++
			stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
			continue
		}
		workflowDataList = append(workflowDataList, workflowData)

		if err := CompileWorkflowDataWithValidation(compiler, workflowData, file, verbose); err != nil {
			// Print the error to stderr (errors from CompileWorkflow are already formatted)
			fmt.Fprintln(os.Stderr, err.Error())
			errorCount++
			stats.Errors++
			stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
			continue
		}
		successCount++
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d out of %d workflow files", successCount, len(mdFiles))))
	}

	// Handle purge logic: delete orphaned .lock.yml files
	if purge && len(existingLockFiles) > 0 {
		// Find lock files that should be deleted (exist but aren't expected)
		expectedLockFileSet := make(map[string]bool)
		for _, expected := range expectedLockFiles {
			expectedLockFileSet[expected] = true
		}

		var orphanedFiles []string
		for _, existing := range existingLockFiles {
			if !expectedLockFileSet[existing] {
				orphanedFiles = append(orphanedFiles, existing)
			}
		}

		// Delete orphaned lock files
		if len(orphanedFiles) > 0 {
			for _, orphanedFile := range orphanedFiles {
				if err := os.Remove(orphanedFile); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to remove orphaned lock file %s: %v", filepath.Base(orphanedFile), err)))
				} else {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Removed orphaned lock file: %s", filepath.Base(orphanedFile))))
				}
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Purged %d orphaned .lock.yml files", len(orphanedFiles))))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No orphaned .lock.yml files found to purge"))
		}
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .gitattributes to mark .lock.yml files as generated"))
	}

	// Generate Dependabot manifests if requested
	if dependabot && !noEmit {
		// Use absolute path for workflow directory
		absWorkflowDir := workflowsDir
		if !filepath.IsAbs(absWorkflowDir) {
			absWorkflowDir = filepath.Join(gitRoot, workflowDir)
		}

		if err := compiler.GenerateDependabotManifests(workflowDataList, absWorkflowDir, forceOverwrite); err != nil {
			if strict {
				return workflowDataList, fmt.Errorf("failed to generate Dependabot manifests: %w", err)
			}
			// Non-strict mode: just report as warning
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate Dependabot manifests: %v", err)))
		}
	}

	// Note: Instructions are only written by the init command
	// The compile command should not write instruction files

	// Print summary
	printCompilationSummary(stats)

	// Return error if any compilations failed
	if errorCount > 0 {
		return workflowDataList, fmt.Errorf("compilation failed")
	}

	// Run zizmor security scanner if requested and compilation was successful
	if zizmor && !noEmit {
		if err := runZizmor(workflowsDir, verbose); err != nil {
			return workflowDataList, fmt.Errorf("zizmor security scan failed: %w", err)
		}
	}

	return workflowDataList, nil
}

// watchAndCompileWorkflows watches for changes to workflow files and recompiles them automatically
func watchAndCompileWorkflows(markdownFile string, compiler *workflow.Compiler, verbose bool) error {
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

	// Also watch subdirectories for include files (recursive watching)
	err = filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors but continue walking
		}
		if info.IsDir() && path != workflowsDir {
			// Add subdirectories to the watcher
			if err := watcher.Add(path); err != nil {
				compileLog.Printf("Failed to watch subdirectory %s: %v", path, err)
			} else {
				compileLog.Printf("Watching subdirectory: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		compileLog.Printf("Failed to walk subdirectories: %v", err)
	}

	// Always emit the begin pattern for task integration
	if markdownFile != "" {
		fmt.Printf("Watching for file changes to %s...\n", markdownFile)
	} else {
		fmt.Printf("Watching for file changes in %s...\n", workflowsDir)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop watching.")
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
		fmt.Fprintln(os.Stderr, "Watching for file changes")
		if verbose {
			fmt.Fprintln(os.Stderr, "🔨 Initial compilation of all workflow files...")
		}
		stats, err := compileAllWorkflowFiles(compiler, workflowsDir, verbose)
		if err != nil {
			// Always show initial compilation errors, not just in verbose mode
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Initial compilation failed: %v", err)))
		}
		// Print summary instead of just "Recompiled"
		printCompilationSummary(stats)
	} else {
		// Reset warning count before compilation
		compiler.ResetWarningCount()

		// Track compilation statistics for single file
		stats := &CompilationStats{Total: 1}

		fmt.Fprintln(os.Stderr, "Watching for file changes")
		if verbose {
			fmt.Fprintf(os.Stderr, "🔨 Initial compilation of %s...\n", markdownFile)
		}
		if err := CompileWorkflowWithValidation(compiler, markdownFile, verbose); err != nil {
			// Always show initial compilation errors on new line without wrapping
			fmt.Fprintln(os.Stderr, err.Error())
			stats.Errors++
		}

		// Get warning count from compiler
		stats.Warnings = compiler.GetWarningCount()

		// Print summary instead of just "Recompiled"
		printCompilationSummary(stats)
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

			compileLog.Printf("Detected change: %s (%s)", event.Name, event.Op.String())
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
					compileModifiedFiles(compiler, filesToCompile, verbose)
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			compileLog.Printf("Watcher error: %v", err)
			if verbose {
				fmt.Printf("⚠️  Watcher error: %v\n", err)
			}

		case <-sigChan:
			if verbose {
				fmt.Fprintln(os.Stderr, "\n🛑 Stopping watch mode...")
			}
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return nil
		}
	}
}

// compileAllWorkflowFiles compiles all markdown files in the workflows directory
func compileAllWorkflowFiles(compiler *workflow.Compiler, workflowsDir string, verbose bool) (*CompilationStats, error) {
	// Reset warning count before compilation
	compiler.ResetWarningCount()

	// Track compilation statistics
	stats := &CompilationStats{}

	// Find all markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return stats, fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		if verbose {
			fmt.Printf("No markdown files found in %s\n", workflowsDir)
		}
		return stats, nil
	}

	// Compile each file
	for _, file := range mdFiles {
		stats.Total++

		compileLog.Printf("Compiling: %s", file)
		if verbose {
			fmt.Printf("🔨 Compiling: %s\n", file)
		}
		if err := CompileWorkflowWithValidation(compiler, file, verbose); err != nil {
			// Always show compilation errors on new line
			fmt.Fprintln(os.Stderr, err.Error())
			stats.Errors++
			stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
		} else {
			compileLog.Printf("Successfully compiled: %s", file)
		}
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
		}
	}

	return stats, nil
}

// compileModifiedFiles compiles a list of modified markdown files
func compileModifiedFiles(compiler *workflow.Compiler, files []string, verbose bool) {
	if len(files) == 0 {
		return
	}

	// Clear screen before emitting new output in watch mode
	console.ClearScreen()

	fmt.Fprintln(os.Stderr, "Watching for file changes")
	if verbose {
		fmt.Fprintf(os.Stderr, "🔨 Compiling %d modified file(s)...\n", len(files))
	}

	// Reset warning count before compilation
	compiler.ResetWarningCount()

	// Track compilation statistics
	stats := &CompilationStats{}

	for _, file := range files {
		// Check if file still exists (might have been deleted between detection and compilation)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			compileLog.Printf("File %s was deleted, skipping compilation", file)
			continue
		}

		stats.Total++

		compileLog.Printf("Compiling: %s", file)
		if verbose {
			fmt.Fprintf(os.Stderr, "🔨 Compiling: %s\n", file)
		}

		if err := CompileWorkflowWithValidation(compiler, file, verbose); err != nil {
			// Always show compilation errors on new line
			fmt.Fprintln(os.Stderr, err.Error())
			stats.Errors++
			stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
		} else {
			compileLog.Printf("Successfully compiled: %s", file)
		}
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Ensure .gitattributes marks .lock.yml files as generated
	if err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
		}
	}

	// Print summary instead of just "Recompiled"
	printCompilationSummary(stats)
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

// printCompilationSummary prints a summary of the compilation results
func printCompilationSummary(stats *CompilationStats) {
	if stats.Total == 0 {
		return
	}

	summary := fmt.Sprintf("Compiled %d workflow(s): %d error(s), %d warning(s)",
		stats.Total, stats.Errors, stats.Warnings)

	// Use different formatting based on whether there were errors
	if stats.Errors > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(summary))
		// List the failed workflows
		if len(stats.FailedWorkflows) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Failed workflows:"))
			for _, workflow := range stats.FailedWorkflows {
				fmt.Fprintf(os.Stderr, "  - %s\n", workflow)
			}
		}
	} else if stats.Warnings > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(summary))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(summary))
	}
}

// zizmorFinding represents a single finding from zizmor JSON output
type zizmorFinding struct {
	Ident          string `json:"ident"`
	Determinations struct {
		Severity string `json:"severity"`
	} `json:"determinations"`
	Locations []struct {
		Symbolic struct {
			Key struct {
				Local struct {
					GivenPath string `json:"given_path"`
				} `json:"Local"`
			} `json:"key"`
		} `json:"symbolic"`
	} `json:"locations"`
}

// runZizmor runs the zizmor security scanner on generated .lock.yml files using Docker
func runZizmor(workflowsDir string, verbose bool) error {
	compileLog.Print("Running zizmor security scanner")

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running zizmor security scanner on generated .lock.yml files..."))
	}

	// Find git root to get the absolute path for Docker volume mount
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Get the absolute path of the workflows directory
	var absWorkflowsDir string
	if filepath.IsAbs(workflowsDir) {
		absWorkflowsDir = workflowsDir
	} else {
		absWorkflowsDir = filepath.Join(gitRoot, workflowsDir)
	}

	compileLog.Printf("Running zizmor on directory: %s", absWorkflowsDir)

	// Build the Docker command with JSON output for easier parsing
	// docker run --rm -v "$(pwd)":/workdir -w /workdir ghcr.io/zizmorcore/zizmor:latest --format json .
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workdir", gitRoot),
		"-w", "/workdir",
		"ghcr.io/zizmorcore/zizmor:latest",
		"--format", "json",
		".",
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Parse and reformat the output
	if err := parseAndDisplayZizmorOutput(stdout.String(), stderr.String(), verbose); err != nil {
		compileLog.Printf("Failed to parse zizmor output: %v", err)
		// Fall back to showing raw output
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	}

	// Check if the error is due to findings (expected) or actual failure
	if err != nil {
		// zizmor uses exit codes to indicate findings:
		// 0 = no findings
		// 10-13 = findings at different severity levels
		// 14 = findings with mixed severities
		// Other codes = actual errors
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			compileLog.Printf("Zizmor exited with code %d", exitCode)
			// Exit codes 10-14 indicate findings, not failures
			// Treat these as success but log them
			if exitCode >= 10 && exitCode <= 14 {
				return nil
			}
			// Other exit codes are actual errors
			return fmt.Errorf("zizmor failed with exit code %d", exitCode)
		}
		// Non-ExitError errors (e.g., command not found)
		return fmt.Errorf("zizmor failed: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Zizmor security scan completed - no findings"))
	}

	return nil
}

// parseAndDisplayZizmorOutput parses zizmor JSON output and displays it in the desired format
func parseAndDisplayZizmorOutput(stdout, stderr string, verbose bool) error {
	// Count findings per file
	fileFindings := make(map[string]int)

	// Parse stderr for "completed" messages to get list of files
	completedFiles := []string{}
	scanner := bufio.NewScanner(strings.NewReader(stderr))
	for scanner.Scan() {
		line := scanner.Text()
		// Look for lines like: " INFO audit: zizmor: 🌈 completed ./.github/workflows/pdf-summary.lock.yml"
		if strings.Contains(line, "INFO audit: zizmor: 🌈 completed") {
			parts := strings.Split(line, "completed ")
			if len(parts) == 2 {
				filePath := strings.TrimSpace(parts[1])
				completedFiles = append(completedFiles, filePath)
				// Initialize count to 0
				fileFindings[filePath] = 0
			}
		}
	}

	// Parse JSON findings from stdout
	var findings []zizmorFinding
	if stdout != "" && strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		if err := json.Unmarshal([]byte(stdout), &findings); err != nil {
			return fmt.Errorf("failed to parse zizmor JSON output: %w", err)
		}

		// Count findings per file - each finding counts as 1 regardless of how many locations it has
		for _, finding := range findings {
			// Track which files this finding affects
			affectedFiles := make(map[string]bool)
			for _, location := range finding.Locations {
				filePath := location.Symbolic.Key.Local.GivenPath
				if filePath != "" && !affectedFiles[filePath] {
					affectedFiles[filePath] = true
					fileFindings[filePath]++
				}
			}
		}
	}

	// Display reformatted output for each completed file
	for _, filePath := range completedFiles {
		count := fileFindings[filePath]
		// Format: 🌈 zizmor xx warnings in <filepath>
		warningText := "warnings"
		if count == 1 {
			warningText = "warning"
		}
		fmt.Fprintf(os.Stderr, "🌈 zizmor %d %s in %s\n", count, warningText, filePath)
	}

	return nil
}
