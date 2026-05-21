// This file provides main orchestration logic for workflow compilation.
//
// This file contains the primary compilation orchestration functions that coordinate
// the compilation of specific files or all files in a directory.
//
// # Organization Rationale
//
// These orchestration functions are grouped here because they:
//   - Coordinate the overall compilation process
//   - Handle both specific file and directory-wide compilation
//   - Integrate all compilation phases (processing, validation, linting, post-processing)
//   - Keep the main CompileWorkflows function small and focused
//
// # Key Functions
//
// Compilation Orchestration:
//   - compileSpecificFiles() - Compile a list of specific workflow files
//   - compileAllFilesInDirectory() - Compile all workflows in a directory
//
// These functions handle the complete compilation pipeline for their respective scenarios.

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileOrchestrationLog = logger.New("cli:compile_pipeline")

const fallbackCompilationErrorMessage = "compilation failed (no detailed error message available)"

// compileResolvedFilesInParallel compiles a list of already-resolved workflow
// file paths, using up to workers goroutines.  Results are returned in the same
// order as the input slice.
//
// When workers <= 1 (or there is only one file) the compilation is sequential:
// no goroutines are launched and the compiler is used directly.  For workers > 1
// the compiler is forked once per file so each goroutine operates on independent
// mutable state; results are merged back into the parent compiler after all
// goroutines finish.
//
// compiler.WarmUp() must have been called before this function so that shared
// lazy-init paths (action cache, import cache, repo config) are fully
// initialised and not triggered from inside a goroutine.
func compileResolvedFilesInParallel(
	ctx context.Context,
	compiler *workflow.Compiler,
	resolvedFiles []string,
	workers int,
	opts compileWorkflowFileOptions,
) []compileWorkflowFileResult {
	n := len(resolvedFiles)
	if n == 0 {
		return nil
	}

	results := make([]compileWorkflowFileResult, n)

	if workers <= 1 || n == 1 {
		// Sequential path: fork/merge still applies so that the pattern is
		// consistent and parent state is not accidentally mutated mid-loop.
		for i, file := range resolvedFiles {
			child := compiler.Fork()
			results[i] = compileWorkflowFile(ctx, child, file, opts)
			compiler.Merge(child)
		}
		return results
	}

	// Parallel path: bounded goroutine pool.
	type indexedResult struct {
		idx   int
		res   compileWorkflowFileResult
		child *workflow.Compiler
	}

	jobs := make(chan int, n)
	for i := range resolvedFiles {
		jobs <- i
	}
	close(jobs)

	resultCh := make(chan indexedResult, n)

	numWorkers := workers
	if numWorkers > n {
		numWorkers = n
	}

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				child := compiler.Fork()
				res := compileWorkflowFile(ctx, child, resolvedFiles[idx], opts)
				resultCh <- indexedResult{idx: idx, res: res, child: child}
			}
		}()
	}

	// Close the result channel once all workers have finished.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results in completion order; merge child compiler state back into
	// the parent serially so no locking is required on the parent.
	for ir := range resultCh {
		results[ir.idx] = ir.res
		compiler.Merge(ir.child)
	}

	return results
}


func compileSpecificFiles(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) ([]*workflow.WorkflowData, error) {
	compileOrchestrationLog.Printf("Compiling %d specific workflow files", len(config.MarkdownFiles))

	// Enable validation automatically when force-refresh-action-pins is used
	// to verify all resolved action SHAs are valid
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	opts := compileWorkflowFileOptions{
		verbose:    config.Verbose,
		jsonOutput: config.JSONOutput,
		noEmit:     config.NoEmit,
		strict:     config.Strict,
		validate:   shouldValidate,
		// zizmor, poutine, actionlint disabled per-file (batched instead)
	}

	// Phase 1 (sequential): resolve every markdown file.  Record resolution
	// errors immediately; collect resolved paths for parallel compilation.
	type entry struct {
		markdownFile    string
		resolvedFile    string
		resolutionError *CompileValidationError
	}

	entries := make([]entry, len(config.MarkdownFiles))
	var toCompile []string
	var toCompileIdx []int

	for i, markdownFile := range config.MarkdownFiles {
		stats.Total++
		e := entry{markdownFile: markdownFile}

		compileOrchestrationLog.Printf("Resolving workflow file: %s", markdownFile)
		resolvedFile, err := resolveWorkflowFile(markdownFile, config.Verbose)
		if err != nil {
			compileOrchestrationLog.Printf("Resolution failed: %s: %v", markdownFile, err)
			stats.Errors++
			trackWorkflowFailure(stats, markdownFile, 1, []string{err.Error()})
			e.resolutionError = &CompileValidationError{
				Type:    "resolution_error",
				Message: err.Error(),
			}
		} else {
			compileOrchestrationLog.Printf("Resolved to: %s", resolvedFile)
			e.resolvedFile = resolvedFile
			toCompile = append(toCompile, resolvedFile)
			toCompileIdx = append(toCompileIdx, i)
		}
		entries[i] = e
	}

	// Phase 2 (parallel): compile all successfully resolved files.
	// WarmUp initialises lazy-loaded shared state before any goroutine is created.
	compiler.WarmUp()
	fileResults := compileResolvedFilesInParallel(ctx, compiler, toCompile, config.Workers, opts)

	// Phase 3 (sequential): aggregate results in original file order.
	fileResultByIdx := make(map[int]compileWorkflowFileResult, len(toCompileIdx))
	for j, entryIdx := range toCompileIdx {
		fileResultByIdx[entryIdx] = fileResults[j]
	}

	var workflowDataList []*workflow.WorkflowData
	var compiledCount int
	var errorCount int
	var lockFilesForActionlint []string
	var lockFilesForZizmor []string
	var lockFilesForDirTools []string

	for i, e := range entries {
		result := ValidationResult{
			Workflow: e.markdownFile,
			Valid:    true,
			Errors:   []CompileValidationError{},
			Warnings: []CompileValidationError{},
		}

		if e.resolutionError != nil {
			errorCount++
			result.Valid = false
			result.Errors = append(result.Errors, *e.resolutionError)
			*validationResults = append(*validationResults, result)
			continue
		}

		// Update result with resolved file name
		result.Workflow = filepath.Base(e.resolvedFile)
		fileResult := fileResultByIdx[i]

		if !fileResult.success {
			// Collect error messages from validation result for display in summary
			var errMsgs []string
			for _, verr := range fileResult.validationResult.Errors {
				errMsgs = append(errMsgs, verr.Message)
			}
			if len(errMsgs) == 0 {
				errMsgs = []string{fallbackCompilationErrorMessage}
			}
			errorCount++
			stats.Errors += len(errMsgs)
			trackWorkflowFailure(stats, e.resolvedFile, len(errMsgs), errMsgs)
		} else {
			compiledCount++
			if fileResult.workflowData != nil {
				workflowDataList = append(workflowDataList, fileResult.workflowData)
			}

			// Collect lock files for batch security tools
			if !config.NoEmit && fileResult.lockFile != "" {
				if _, err := os.Stat(fileResult.lockFile); err == nil {
					if config.Actionlint {
						lockFilesForActionlint = append(lockFilesForActionlint, fileResult.lockFile)
					}
					if config.Zizmor {
						lockFilesForZizmor = append(lockFilesForZizmor, fileResult.lockFile)
					}
					if config.Poutine || config.RunnerGuard {
						lockFilesForDirTools = append(lockFilesForDirTools, fileResult.lockFile)
					}
				}
			}
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	// Run batch actionlint on all collected lock files
	if config.Actionlint && !config.NoEmit && len(lockFilesForActionlint) > 0 {
		if err := RunActionlintOnFiles(lockFilesForActionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch zizmor on all collected lock files
	if config.Zizmor && !config.NoEmit && len(lockFilesForZizmor) > 0 {
		if err := RunZizmorOnFiles(lockFilesForZizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch poutine once on the workflow directory
	// Get the directory from the first lock file (all should be in same directory)
	if config.Poutine && !config.NoEmit && len(lockFilesForDirTools) > 0 {
		workflowDir := filepath.Dir(lockFilesForDirTools[0])
		if err := runBatchDirectoryTool("poutine", workflowDir, config.Verbose && !config.JSONOutput, config.Strict, RunPoutineOnDirectory); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch runner-guard once on the workflow directory
	// Get the directory from the first lock file (all should be in same directory)
	if config.RunnerGuard && !config.NoEmit && len(lockFilesForDirTools) > 0 {
		workflowDir := filepath.Dir(lockFilesForDirTools[0])
		if err := runBatchDirectoryTool("runner-guard", workflowDir, config.Verbose && !config.JSONOutput, config.Strict, RunRunnerGuardOnDirectory); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Display schedule warnings
	displayScheduleWarnings(compiler, config.JSONOutput)

	// Display safe update warnings (emitted as prompts for the calling agent)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)

	// Post-processing
	if err := runPostProcessing(compiler, workflowDataList, config, compiledCount); err != nil {
		return workflowDataList, err
	}

	// Output results
	if err := outputResults(stats, validationResults, config); err != nil {
		return workflowDataList, err
	}

	// Return error if any compilations failed
	// Don't return the detailed error message here since it's already printed in the summary
	// Returning a simple error prevents duplication in the output
	if errorCount > 0 {
		return workflowDataList, errors.New("compilation failed")
	}

	return workflowDataList, nil
}

// compileAllFilesInDirectory compiles all workflow files in a directory
func compileAllFilesInDirectory(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	workflowDir string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) ([]*workflow.WorkflowData, error) {
	// Find git root for consistent behavior
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileOrchestrationLog.Printf("Found git root: %s", gitRoot)

	// Compile all markdown files in the specified workflow directory
	workflowsDir := filepath.Join(gitRoot, workflowDir)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}

	compileOrchestrationLog.Printf("Scanning for markdown files in %s", workflowsDir)
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Scanning for markdown files in "+workflowsDir))
	}

	// Find and filter markdown files (shared helper keeps logic in one place)
	mdFiles, err := getMarkdownWorkflowFiles(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}
	mdFiles, err = filterMarkdownFilesWithFrontmatter(mdFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to filter markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		return nil, fmt.Errorf("no workflow markdown files found in %s (workflow files must start with a frontmatter opener on the first line)", workflowsDir)
	}

	compileOrchestrationLog.Printf("Found %d markdown files to compile", len(mdFiles))
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d markdown files to compile", len(mdFiles))))
	}

	// Handle purge logic: collect existing files before compilation
	var purgeData *purgeTrackingData
	if config.Purge {
		purgeData = collectPurgeData(workflowsDir, mdFiles, config.Verbose)
	}

	// Enable validation automatically when force-refresh-action-pins is used
	// to verify all resolved action SHAs are valid
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	stats.Total += len(mdFiles)

	opts := compileWorkflowFileOptions{
		verbose:    config.Verbose,
		jsonOutput: config.JSONOutput,
		noEmit:     config.NoEmit,
		strict:     config.Strict,
		validate:   shouldValidate,
		// zizmor, poutine, actionlint disabled per-file (batched instead)
	}

	// Phase 2 (parallel): compile all files.
	// WarmUp initialises lazy-loaded shared state before any goroutine is created.
	compiler.WarmUp()
	fileResults := compileResolvedFilesInParallel(ctx, compiler, mdFiles, config.Workers, opts)

	// Phase 3 (sequential): aggregate results in original file order.
	var workflowDataList []*workflow.WorkflowData
	var successCount int
	var errorCount int
	var lockFilesForActionlint []string
	var lockFilesForZizmor []string
	var lockFilesForDirTools []string // lock files for directory-based tools (poutine, runner-guard)

	for i, file := range mdFiles {
		fileResult := fileResults[i]

		if !fileResult.success {
			// Collect error messages from validation result
			var errMsgs []string
			for _, verr := range fileResult.validationResult.Errors {
				errMsgs = append(errMsgs, verr.Message)
			}
			if len(errMsgs) == 0 {
				errMsgs = []string{fallbackCompilationErrorMessage}
			}
			errorCount++
			stats.Errors += len(errMsgs)
			trackWorkflowFailure(stats, file, len(errMsgs), errMsgs)
		} else {
			successCount++
			if fileResult.workflowData != nil {
				workflowDataList = append(workflowDataList, fileResult.workflowData)
			}

			// Collect lock files for batch security tools
			if !config.NoEmit && fileResult.lockFile != "" {
				if _, err := os.Stat(fileResult.lockFile); err == nil {
					if config.Actionlint {
						lockFilesForActionlint = append(lockFilesForActionlint, fileResult.lockFile)
					}
					if config.Zizmor {
						lockFilesForZizmor = append(lockFilesForZizmor, fileResult.lockFile)
					}
					if config.Poutine || config.RunnerGuard {
						lockFilesForDirTools = append(lockFilesForDirTools, fileResult.lockFile)
					}
				}
			}
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	// Run batch actionlint
	if config.Actionlint && !config.NoEmit && len(lockFilesForActionlint) > 0 {
		if err := RunActionlintOnFiles(lockFilesForActionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch zizmor
	if config.Zizmor && !config.NoEmit && len(lockFilesForZizmor) > 0 {
		if err := RunZizmorOnFiles(lockFilesForZizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch poutine once on the workflow directory
	if config.Poutine && !config.NoEmit && len(lockFilesForDirTools) > 0 {
		if err := runBatchDirectoryTool("poutine", workflowsDir, config.Verbose && !config.JSONOutput, config.Strict, RunPoutineOnDirectory); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch runner-guard once on the workflow directory
	if config.RunnerGuard && !config.NoEmit && len(lockFilesForDirTools) > 0 {
		if err := runBatchDirectoryTool("runner-guard", workflowsDir, config.Verbose && !config.JSONOutput, config.Strict, RunRunnerGuardOnDirectory); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Emit recommendation when many slash commands are present without centralized strategy.
	displayCentralizedSlashCommandRecommendation(compiler, workflowDataList, config.JSONOutput)

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Display schedule warnings
	displayScheduleWarnings(compiler, config.JSONOutput)

	// Display safe update warnings (emitted as prompts for the calling agent)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)

	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d out of %d workflow files", successCount, len(mdFiles))))
	}

	// Handle purge logic if requested
	if config.Purge && purgeData != nil {
		runPurgeOperations(workflowsDir, purgeData, config.Verbose)
	}

	// Post-processing
	if err := runPostProcessingForDirectory(ctx, compiler, workflowDataList, config, workflowsDir, gitRoot, successCount); err != nil {
		return workflowDataList, err
	}

	// Output results.
	// Populate MarkdownFiles so that outputResults can collect per-workflow stats
	// (e.g. schedule heatmap) even when the caller did not specify explicit files.
	if config.Stats && len(config.MarkdownFiles) == 0 {
		config.MarkdownFiles = mdFiles
	}
	if err := outputResults(stats, validationResults, config); err != nil {
		return workflowDataList, err
	}

	// Return error if any compilations failed
	if errorCount > 0 {
		return workflowDataList, errors.New("compilation failed")
	}

	return workflowDataList, nil
}

// purgeTrackingData holds data needed for purge operations
type purgeTrackingData struct {
	existingLockFiles    []string
	existingInvalidFiles []string
	expectedLockFiles    []string
}

// collectPurgeData collects existing files for purge operations
func collectPurgeData(workflowsDir string, mdFiles []string, verbose bool) *purgeTrackingData {
	data := &purgeTrackingData{}

	// Find all existing files
	data.existingLockFiles, _ = filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	data.existingInvalidFiles, _ = filepath.Glob(filepath.Join(workflowsDir, "*.invalid.yml"))

	// Create expected files list
	for _, mdFile := range mdFiles {
		lockFile := stringutil.MarkdownToLockFile(mdFile)
		data.expectedLockFiles = append(data.expectedLockFiles, lockFile)
	}

	if verbose {
		if len(data.existingLockFiles) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d existing .lock.yml files", len(data.existingLockFiles))))
		}
		if len(data.existingInvalidFiles) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d existing .invalid.yml files", len(data.existingInvalidFiles))))
		}
	}

	return data
}

// runPurgeOperations runs all purge operations
func runPurgeOperations(workflowsDir string, data *purgeTrackingData, verbose bool) {
	// Errors from purge operations are logged but don't stop compilation
	_ = purgeOrphanedLockFiles(workflowsDir, data.expectedLockFiles, verbose)
	_ = purgeInvalidFiles(workflowsDir, verbose)
}

// runPostProcessing runs post-processing for specific files compilation
func runPostProcessing(
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	config CompileConfig,
	successCount int,
) error {
	// Get action cache
	actionCache := compiler.GetSharedActionCache()

	// Update .gitattributes (errors are non-fatal)
	_ = updateGitAttributes(successCount, actionCache, config.Verbose)

	// Generate Dependabot manifests if requested
	if config.Dependabot && !config.NoEmit {
		gitRoot, err := gitutil.FindGitRoot()
		if err == nil {
			absWorkflowDir := filepath.Join(gitRoot, config.WorkflowDir)
			if err := generateDependabotManifestsWrapper(compiler, workflowDataList, absWorkflowDir, config.ForceOverwrite, config.Strict); err != nil {
				if config.Strict {
					return err
				}
			}
		}
	}

	// Reconcile compiler-managed Dependabot ignore entries for compiler-emitted action refs.
	if !config.NoEmit {
		if gitRoot, err := gitutil.FindGitRoot(); err == nil {
			if err := compiler.ReconcileManagedDependabotIgnoresInRepo(gitRoot); err != nil {
				if config.Strict {
					return err
				}
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to reconcile compiler-managed Dependabot ignore entries: %v", err)))
			}
		}
	}

	// Generate maintenance workflow if needed
	// Only generate when compiling all workflows (not specific files)
	// Skip when using custom --dir option or when compiling specific files
	// Note: Maintenance workflow generation requires parsing all workflows in the directory
	// to check for expires fields, so we skip it when compiling specific files to avoid
	// unnecessary parsing and warnings from unrelated workflows

	// Prune stale gh-aw-actions entries before saving
	pruneStaleActionCacheEntries(compiler, actionCache)

	// Save action cache (errors are logged but non-fatal)
	_ = saveActionCache(actionCache, config.Verbose)

	return nil
}

// runPostProcessingForDirectory runs post-processing for directory compilation
func runPostProcessingForDirectory(
	ctx context.Context,
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	config CompileConfig,
	workflowsDir string,
	gitRoot string,
	successCount int,
) error {
	// Get action cache
	actionCache := compiler.GetSharedActionCache()

	// Update .gitattributes (errors are non-fatal)
	_ = updateGitAttributes(successCount, actionCache, config.Verbose)

	// Generate Dependabot manifests if requested
	if config.Dependabot && !config.NoEmit {
		absWorkflowDir := getAbsoluteWorkflowDir(workflowsDir, gitRoot)
		if err := generateDependabotManifestsWrapper(compiler, workflowDataList, absWorkflowDir, config.ForceOverwrite, config.Strict); err != nil {
			if config.Strict {
				return err
			}
		}
	}

	// Reconcile compiler-managed Dependabot ignore entries for compiler-emitted action refs.
	if !config.NoEmit {
		if err := compiler.ReconcileManagedDependabotIgnoresInRepo(gitRoot); err != nil {
			if config.Strict {
				return err
			}
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to reconcile compiler-managed Dependabot ignore entries: %v", err)))
		}
	}

	// Generate maintenance workflow if needed.
	// Skip maintenance workflow generation when using custom --dir option.
	// Keep invoking generators for empty workflowDataList so stale generated files are cleaned up.
	if !config.NoEmit && config.WorkflowDir == "" {
		absWorkflowDir := getAbsoluteWorkflowDir(workflowsDir, gitRoot)
		if err := generateMaintenanceWorkflowWrapper(ctx, compiler, workflowDataList, absWorkflowDir, gitRoot, config.Verbose, config.Strict); err != nil {
			if config.Strict {
				return err
			}
		}
		if err := generateCentralSlashCommandWorkflowWrapper(ctx, workflowDataList, absWorkflowDir, config.Strict); err != nil {
			if config.Strict {
				return err
			}
		}
	}

	// Prune stale gh-aw-actions entries before saving
	pruneStaleActionCacheEntries(compiler, actionCache)

	// Save action cache (errors are logged but non-fatal)
	_ = saveActionCache(actionCache, config.Verbose)

	return nil
}

// outputResults outputs compilation results in the requested format
func outputResults(
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	config CompileConfig,
) error {
	// Collect and display stats if requested
	if config.Stats && !config.NoEmit && !config.JSONOutput {
		var statsList []*WorkflowStats
		if len(config.MarkdownFiles) > 0 {
			statsList = collectWorkflowStatisticsWrapper(config.MarkdownFiles)
		}
		displayStatsTable(statsList)
		displayScheduleCalendar(statsList)
	}

	// Output JSON if requested
	if config.JSONOutput {
		jsonStr, err := formatValidationOutput(*validationResults)
		if err != nil {
			return err
		}
		fmt.Println(jsonStr)
	} else if !config.Stats {
		// Print summary for text output (skip if stats mode)
		printCompilationSummary(stats, config.ShowAllErrors)
	}

	// Display actionlint summary if enabled
	if config.Actionlint && !config.NoEmit && !config.JSONOutput {
		displayActionlintSummary()
	}

	return nil
}
