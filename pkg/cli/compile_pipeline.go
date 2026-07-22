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

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileOrchestrationLog = logger.New("cli:compile_pipeline")

const fallbackCompilationErrorMessage = "compilation failed (no detailed error message available)"

// compileSpecificFiles compiles a specific list of workflow files
func compileSpecificFiles(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) ([]*workflow.WorkflowData, error) {
	compileOrchestrationLog.Printf("Compiling %d specific workflow files", len(config.MarkdownFiles))
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	results, err := processSpecificWorkflowFiles(ctx, compiler, config, stats, validationResults, shouldValidate)
	if err != nil {
		return results.workflowDataList, err
	}

	return finalizeCompileSpecificFiles(ctx, compiler, config, stats, validationResults, results)
}

type specificFileLockFiles struct {
	actionlint []string
	zizmor     []string
	dirTools   []string
}

type specificFileCompileResults struct {
	workflowDataList []*workflow.WorkflowData
	compiledCount    int
	errorCount       int
	lockFiles        specificFileLockFiles
}

func newSpecificFileLockFiles() specificFileLockFiles {
	return specificFileLockFiles{
		actionlint: []string{},
		zizmor:     []string{},
		dirTools:   []string{},
	}
}

func checkCompileSpecificFilesContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
		return nil
	}
}

func newCompileSpecificFilesValidationResult(workflowName string) ValidationResult {
	return ValidationResult{
		Workflow: workflowName,
		Valid:    true,
		Errors:   []CompileValidationError{},
		Warnings: []CompileValidationError{},
	}
}

func processSpecificWorkflowFiles(ctx context.Context, compiler *workflow.Compiler, config CompileConfig, stats *CompilationStats, validationResults *[]ValidationResult, shouldValidate bool) (specificFileCompileResults, error) {
	results := specificFileCompileResults{
		lockFiles: newSpecificFileLockFiles(),
	}
	for _, markdownFile := range config.MarkdownFiles {
		if err := processSpecificWorkflowFile(ctx, compiler, config, stats, validationResults, shouldValidate, markdownFile, &results); err != nil {
			return results, err
		}
	}
	return results, nil
}

func processSpecificWorkflowFile(ctx context.Context, compiler *workflow.Compiler, config CompileConfig, stats *CompilationStats, validationResults *[]ValidationResult, shouldValidate bool, markdownFile string, results *specificFileCompileResults) error {
	if err := checkCompileSpecificFilesContext(ctx); err != nil {
		return err
	}
	stats.Total++
	result := newCompileSpecificFilesValidationResult(markdownFile)
	resolvedFile, ok := resolveSpecificWorkflowFile(config.Verbose, markdownFile, stats, &result, &results.errorCount)
	if !ok {
		*validationResults = append(*validationResults, result)
		return nil
	}
	fileResult := compileSpecificWorkflowFile(ctx, compiler, config, shouldValidate, resolvedFile)
	recordSpecificWorkflowCompileResult(config, stats, resolvedFile, fileResult, results)
	*validationResults = append(*validationResults, fileResult.validationResult)
	return nil
}

func resolveSpecificWorkflowFile(verbose bool, markdownFile string, stats *CompilationStats, result *ValidationResult, errorCount *int) (string, bool) {
	compileOrchestrationLog.Printf("Resolving workflow file: %s", markdownFile)
	resolvedFile, err := resolveWorkflowFile(markdownFile, verbose)
	if err != nil {
		*errorCount += recordCompileSpecificFilesResolutionError(stats, result, markdownFile, err)
		return "", false
	}
	compileOrchestrationLog.Printf("Resolved to: %s", resolvedFile)
	result.Workflow = filepath.Base(resolvedFile)
	return resolvedFile, true
}

func compileSpecificWorkflowFile(ctx context.Context, compiler *workflow.Compiler, config CompileConfig, shouldValidate bool, resolvedFile string) compileWorkflowFileResult {
	return compileWorkflowFile(
		ctx, compiler, resolvedFile, compileWorkflowFileOptions{
			verbose:    config.Verbose,
			jsonOutput: config.JSONOutput,
			noEmit:     config.NoEmit,
			strict:     config.Strict,
			validate:   shouldValidate,
		},
	)
}

func recordSpecificWorkflowCompileResult(config CompileConfig, stats *CompilationStats, resolvedFile string, fileResult compileWorkflowFileResult, results *specificFileCompileResults) {
	if !fileResult.success {
		results.errorCount += recordCompileSpecificFilesFailure(stats, resolvedFile, fileResult.validationResult)
		return
	}
	results.compiledCount++
	if fileResult.workflowData != nil {
		results.workflowDataList = append(results.workflowDataList, fileResult.workflowData)
	}
	results.lockFiles.collect(config, fileResult.lockFile)
}

func recordCompileSpecificFilesResolutionError(stats *CompilationStats, result *ValidationResult, markdownFile string, err error) int {
	stats.Errors++
	trackWorkflowFailure(stats, markdownFile, 1, []string{err.Error()})
	result.Valid = false
	result.Errors = append(result.Errors, CompileValidationError{
		Type:    "resolution_error",
		Message: err.Error(),
	})
	return 1
}

func recordCompileSpecificFilesFailure(stats *CompilationStats, resolvedFile string, validationResult ValidationResult) int {
	errMsgs := collectCompileSpecificFilesErrorMessages(validationResult)
	stats.Errors += len(errMsgs)
	trackWorkflowFailure(stats, resolvedFile, len(errMsgs), errMsgs)
	return 1
}

func collectCompileSpecificFilesErrorMessages(validationResult ValidationResult) []string {
	errMsgs := make([]string, 0, len(validationResult.Errors))
	for _, validationErr := range validationResult.Errors {
		errMsgs = append(errMsgs, validationErr.Message)
	}
	if len(errMsgs) == 0 {
		return []string{fallbackCompilationErrorMessage}
	}
	return errMsgs
}

func (f *specificFileLockFiles) collect(config CompileConfig, lockFile string) {
	if config.NoEmit || lockFile == "" {
		return
	}
	if _, err := os.Stat(lockFile); err != nil {
		return
	}
	if config.Actionlint {
		f.actionlint = append(f.actionlint, lockFile)
	}
	if config.Zizmor {
		f.zizmor = append(f.zizmor, lockFile)
	}
	if config.Poutine || config.RunnerGuard {
		f.dirTools = append(f.dirTools, lockFile)
	}
}

func runSpecificFilesBatchTools(ctx context.Context, config CompileConfig, lockFiles specificFileLockFiles) error {
	if err := runSpecificFilesActionlint(ctx, config, lockFiles.actionlint); err != nil {
		return err
	}
	if err := runSpecificFilesZizmor(config, lockFiles.zizmor); err != nil {
		return err
	}
	if err := runSpecificFilesDirectoryTools(ctx, config, lockFiles.dirTools); err != nil {
		return err
	}
	return nil
}

func runSpecificFilesActionlint(ctx context.Context, config CompileConfig, lockFiles []string) error {
	if !config.Actionlint || config.NoEmit || len(lockFiles) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := RunActionlintOnFiles(ctx, lockFiles, config.Verbose && !config.JSONOutput, config.Strict); err != nil && config.Strict {
		return err
	}
	return nil
}

func runSpecificFilesZizmor(config CompileConfig, lockFiles []string) error {
	if !config.Zizmor || config.NoEmit || len(lockFiles) == 0 {
		return nil
	}
	if err := RunZizmorOnFiles(lockFiles, config.Verbose && !config.JSONOutput, config.Strict); err != nil && config.Strict {
		return err
	}
	return nil
}

func runSpecificFilesDirectoryTools(ctx context.Context, config CompileConfig, lockFiles []string) error {
	if config.NoEmit || len(lockFiles) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	workflowDir := filepath.Dir(lockFiles[0])
	if err := runSpecificFilesDirectoryTool(config.Poutine, "poutine", workflowDir, config, RunPoutineOnDirectory); err != nil {
		return err
	}
	if err := runSpecificFilesDirectoryTool(config.RunnerGuard, "runner-guard", workflowDir, config, RunRunnerGuardOnDirectory); err != nil {
		return err
	}
	return nil
}

func runSpecificFilesDirectoryTool(enabled bool, toolName, workflowDir string, config CompileConfig, runner func(string, bool, bool) error) error {
	if !enabled {
		return nil
	}
	if err := runBatchDirectoryTool(toolName, workflowDir, config.Verbose && !config.JSONOutput, config.Strict, runner); err != nil && config.Strict {
		return err
	}
	return nil
}

func finalizeCompileSpecificFiles(ctx context.Context, compiler *workflow.Compiler, config CompileConfig, stats *CompilationStats, validationResults *[]ValidationResult, results specificFileCompileResults) ([]*workflow.WorkflowData, error) {
	if err := runSpecificFilesBatchTools(ctx, config, results.lockFiles); err != nil {
		return results.workflowDataList, err
	}
	stats.Warnings = compiler.GetWarningCount()
	displayScheduleWarnings(compiler, config.JSONOutput)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)
	if err := runPostProcessing(compiler, results.workflowDataList, config, results.compiledCount); err != nil {
		return results.workflowDataList, err
	}
	if err := outputResults(stats, validationResults, config); err != nil {
		return results.workflowDataList, err
	}
	if results.errorCount > 0 {
		return results.workflowDataList, errors.New("compilation failed")
	}
	return results.workflowDataList, nil
}

type compileAllDirectoryContext struct {
	gitRoot      string
	workflowsDir string
	mdFiles      []string
	purgeData    *purgeTrackingData
}

type compileAllDirectoryResult struct {
	workflowDataList []*workflow.WorkflowData
	successCount     int
	errorCount       int
	lockFiles        specificFileLockFiles
}

func loadCompileAllDirectoryContext(config CompileConfig, workflowDir string) (*compileAllDirectoryContext, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileOrchestrationLog.Printf("Found git root: %s", gitRoot)

	workflowsDir := filepath.Join(gitRoot, workflowDir)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}

	mdFiles, err := findCompileAllMarkdownFiles(workflowsDir, config.Verbose)
	if err != nil {
		return nil, err
	}

	var purgeData *purgeTrackingData
	if config.Purge {
		purgeData = collectPurgeData(workflowsDir, mdFiles, config.Verbose)
	}

	return &compileAllDirectoryContext{
		gitRoot:      gitRoot,
		workflowsDir: workflowsDir,
		mdFiles:      mdFiles,
		purgeData:    purgeData,
	}, nil
}

func findCompileAllMarkdownFiles(workflowsDir string, verbose bool) ([]string, error) {
	compileOrchestrationLog.Printf("Scanning for markdown files in %s", workflowsDir)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Scanning for markdown files in "+workflowsDir))
	}

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
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d markdown files to compile", len(mdFiles))))
	}

	return mdFiles, nil
}

func compileAllDirectoryWorkflowFiles(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	mdFiles []string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) (*compileAllDirectoryResult, error) {
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	result := &compileAllDirectoryResult{
		workflowDataList: []*workflow.WorkflowData{},
		lockFiles:        newSpecificFileLockFiles(),
	}

	for _, file := range mdFiles {
		if err := checkCompileSpecificFilesContext(ctx); err != nil {
			return result, err
		}

		stats.Total++
		fileResult := compileWorkflowFile(
			ctx, compiler, file, compileWorkflowFileOptions{
				verbose:    config.Verbose,
				jsonOutput: config.JSONOutput,
				noEmit:     config.NoEmit,
				strict:     config.Strict,
				validate:   shouldValidate,
			},
		)

		if !fileResult.success {
			result.errorCount += recordCompileSpecificFilesFailure(stats, file, fileResult.validationResult)
		} else {
			result.successCount++
			if fileResult.workflowData != nil {
				result.workflowDataList = append(result.workflowDataList, fileResult.workflowData)
			}
			result.lockFiles.collect(config, fileResult.lockFile)
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	return result, nil
}

func runCompileAllDirectoryBatchTools(
	ctx context.Context,
	config CompileConfig,
	workflowsDir string,
	lockFiles specificFileLockFiles,
) error {
	if err := runSpecificFilesActionlint(ctx, config, lockFiles.actionlint); err != nil {
		return err
	}
	if err := runSpecificFilesZizmor(config, lockFiles.zizmor); err != nil {
		return err
	}
	if err := runCompileAllDirectoryTool(ctx, config.Poutine, "poutine", workflowsDir, config, RunPoutineOnDirectory, lockFiles.dirTools); err != nil {
		return err
	}
	if err := runCompileAllDirectoryTool(ctx, config.RunnerGuard, "runner-guard", workflowsDir, config, RunRunnerGuardOnDirectory, lockFiles.dirTools); err != nil {
		return err
	}
	return nil
}

func runCompileAllDirectoryTool(
	ctx context.Context,
	enabled bool,
	toolName string,
	workflowsDir string,
	config CompileConfig,
	runner func(string, bool, bool) error,
	lockFiles []string,
) error {
	if !enabled || config.NoEmit || len(lockFiles) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := runBatchDirectoryTool(toolName, workflowsDir, config.Verbose && !config.JSONOutput, config.Strict, runner); err != nil && config.Strict {
		return err
	}
	return nil
}

func finalizeCompileAllDirectoryResults(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	dirCtx *compileAllDirectoryContext,
	result *compileAllDirectoryResult,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) error {
	displayCentralizedSlashCommandRecommendation(compiler, result.workflowDataList, config.JSONOutput)
	stats.Warnings = compiler.GetWarningCount()
	displayScheduleWarnings(compiler, config.JSONOutput)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)

	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d out of %d workflow files", result.successCount, len(dirCtx.mdFiles))))
	}
	if config.Purge && dirCtx.purgeData != nil {
		runPurgeOperations(dirCtx.workflowsDir, dirCtx.purgeData, config.Verbose)
	}
	if err := runPostProcessingForDirectory(ctx, compiler, result.workflowDataList, config, dirCtx.workflowsDir, dirCtx.gitRoot, result.successCount, result.errorCount); err != nil {
		return err
	}

	if config.Stats && len(config.MarkdownFiles) == 0 {
		config.MarkdownFiles = dirCtx.mdFiles
	}
	if err := outputResults(stats, validationResults, config); err != nil {
		return err
	}
	if result.errorCount > 0 {
		return errors.New("compilation failed")
	}
	return nil
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
	dirCtx, err := loadCompileAllDirectoryContext(config, workflowDir)
	if err != nil {
		return nil, err
	}

	result, err := compileAllDirectoryWorkflowFiles(ctx, compiler, config, dirCtx.mdFiles, stats, validationResults)
	if err != nil {
		return result.workflowDataList, err
	}
	if err := runCompileAllDirectoryBatchTools(ctx, config, dirCtx.workflowsDir, result.lockFiles); err != nil {
		return result.workflowDataList, err
	}
	if err := finalizeCompileAllDirectoryResults(ctx, compiler, config, dirCtx, result, stats, validationResults); err != nil {
		return result.workflowDataList, err
	}

	return result.workflowDataList, nil
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

	// Generate Dependabot manifests and reconcile compiler-managed ignore entries if requested.
	if config.Dependabot && !config.NoEmit {
		if gitRoot, err := gitutil.FindGitRoot(); err == nil {
			absWorkflowDir := filepath.Join(gitRoot, config.WorkflowDir)
			if err := generateDependabotManifestsWrapper(compiler, workflowDataList, absWorkflowDir, config.ForceOverwrite, config.Strict); err != nil {
				if config.Strict {
					return err
				}
			}
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
	errorCount int,
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
	if config.Dependabot && !config.NoEmit {
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
		if err := generateCentralSlashCommandWorkflowWrapper(ctx, workflowDataList, absWorkflowDir, gitRoot, config.Strict); err != nil {
			if config.Strict {
				return err
			}
		}
	}

	// Prune stale gh-aw-actions entries before saving
	pruneStaleActionCacheEntries(compiler, actionCache)

	// Prune orphaned entries — entries for action versions no longer referenced
	// by any workflow in the directory (e.g. old pins left after a version bump).
	// Safe to call only after a full-directory compilation with zero compile errors.
	pruneOrphanedActionCacheEntries(compiler, actionCache, errorCount)

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
		fmt.Fprintln(os.Stdout, jsonStr)
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
