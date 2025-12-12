package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var compileHelpersLog = logger.New("cli:compile_helpers")

// compileSingleFile compiles a single markdown workflow file and updates compilation statistics
// If checkExists is true, the function will check if the file exists before compiling
// Returns true if compilation was attempted (file exists or checkExists is false), false otherwise
func compileSingleFile(compiler *workflow.Compiler, file string, stats *CompilationStats, verbose bool, checkExists bool) bool {
	// Check if file exists if requested (for watch mode)
	if checkExists {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			compileHelpersLog.Printf("File %s was deleted, skipping compilation", file)
			return false
		}
	}

	stats.Total++

	compileHelpersLog.Printf("Compiling: %s", file)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Compiling: %s", file)))
	}

	if err := CompileWorkflowWithValidation(compiler, file, verbose, false, false, false, false, false); err != nil {
		// Always show compilation errors on new line
		fmt.Fprintln(os.Stderr, err.Error())
		stats.Errors++
		stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
	} else {
		compileHelpersLog.Printf("Successfully compiled: %s", file)
	}

	return true
}

// compileAllWorkflowFiles compiles all markdown files in the workflows directory
func compileAllWorkflowFiles(compiler *workflow.Compiler, workflowsDir string, verbose bool) (*CompilationStats, error) {
	compileHelpersLog.Printf("Compiling all workflow files in directory: %s", workflowsDir)
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
		compileHelpersLog.Printf("No markdown files found in %s", workflowsDir)
		if verbose {
			fmt.Printf("No markdown files found in %s\n", workflowsDir)
		}
		return stats, nil
	}

	compileHelpersLog.Printf("Found %d markdown files to compile", len(mdFiles))

	// Compile each file
	for _, file := range mdFiles {
		compileSingleFile(compiler, file, stats, verbose, false)
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Save the action cache after all compilations
	actionCache := compiler.GetSharedActionCache()
	hasActionCacheEntries := actionCache != nil && len(actionCache.Entries) > 0
	successCount := stats.Total - stats.Errors

	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileHelpersLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save action cache: %v", err)))
			}
		} else {
			compileHelpersLog.Print("Action cache saved successfully")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Action cache saved to %s", actionCache.GetCachePath())))
			}
		}
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	// Only update if we successfully compiled workflows or have action cache entries
	if successCount > 0 || hasActionCacheEntries {
		if err := ensureGitAttributes(); err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
			}
		}
	} else {
		compileHelpersLog.Print("Skipping .gitattributes update (no compiled workflows and no action cache entries)")
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
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Compiling %d modified file(s)...", len(files))))
	}

	// Reset warning count before compilation
	compiler.ResetWarningCount()

	// Track compilation statistics
	stats := &CompilationStats{}

	for _, file := range files {
		compileSingleFile(compiler, file, stats, verbose, true)
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Save the action cache after compilations
	actionCache := compiler.GetSharedActionCache()
	hasActionCacheEntries := actionCache != nil && len(actionCache.Entries) > 0
	successCount := stats.Total - stats.Errors

	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileHelpersLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save action cache: %v", err)))
			}
		} else {
			compileHelpersLog.Print("Action cache saved successfully")
		}
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	// Only update if we successfully compiled workflows or have action cache entries
	if successCount > 0 || hasActionCacheEntries {
		if err := ensureGitAttributes(); err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
			}
		}
	} else {
		compileHelpersLog.Print("Skipping .gitattributes update (no compiled workflows and no action cache entries)")
	}

	// Print summary instead of just "Recompiled"
	printCompilationSummary(stats)
}

// compileModifiedFilesWithDependencies compiles modified files and their dependencies using the dependency graph
func compileModifiedFilesWithDependencies(compiler *workflow.Compiler, depGraph *DependencyGraph, files []string, verbose bool) {
	if len(files) == 0 {
		return
	}

	// Clear screen before emitting new output in watch mode
	console.ClearScreen()

	// Use dependency graph to determine what needs to be recompiled
	var workflowsToCompile []string
	uniqueWorkflows := make(map[string]bool)

	for _, modifiedFile := range files {
		compileHelpersLog.Printf("Processing modified file: %s", modifiedFile)

		// Update the workflow in the dependency graph
		if err := depGraph.UpdateWorkflow(modifiedFile, compiler); err != nil {
			compileHelpersLog.Printf("Warning: failed to update workflow in dependency graph: %v", err)
		}

		// Get affected workflows from dependency graph
		affected := depGraph.GetAffectedWorkflows(modifiedFile)
		compileHelpersLog.Printf("File %s affects %d workflow(s)", modifiedFile, len(affected))

		// Add to unique set
		for _, workflow := range affected {
			if !uniqueWorkflows[workflow] {
				uniqueWorkflows[workflow] = true
				workflowsToCompile = append(workflowsToCompile, workflow)
			}
		}
	}

	fmt.Fprintln(os.Stderr, "Watching for file changes")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Recompiling %d workflow(s) affected by %d change(s)...", len(workflowsToCompile), len(files))))
	}

	// Reset warning count before compilation
	compiler.ResetWarningCount()

	// Track compilation statistics
	stats := &CompilationStats{}

	for _, file := range workflowsToCompile {
		compileSingleFile(compiler, file, stats, verbose, true)
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Save the action cache after compilations
	actionCache := compiler.GetSharedActionCache()
	hasActionCacheEntries := actionCache != nil && len(actionCache.Entries) > 0
	successCount := stats.Total - stats.Errors

	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileHelpersLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save action cache: %v", err)))
			}
		} else {
			compileHelpersLog.Print("Action cache saved successfully")
		}
	}

	// Ensure .gitattributes marks .lock.yml files as generated
	// Only update if we successfully compiled workflows or have action cache entries
	if successCount > 0 || hasActionCacheEntries {
		if err := ensureGitAttributes(); err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to update .gitattributes: %v\n", err)
			}
		}
	} else {
		compileHelpersLog.Print("Skipping .gitattributes update (no compiled workflows and no action cache entries)")
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
