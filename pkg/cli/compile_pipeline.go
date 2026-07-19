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

type compileSpecificFilesState struct {
	workflowDataList       []*workflow.WorkflowData
	compiledCount          int
	errorCount             int
	lockFilesForActionlint []string
	lockFilesForZizmor     []string
	lockFilesForDirTools   []string
}

type compileAllFilesInDirectoryState struct {
	workflowDataList       []*workflow.WorkflowData
	successCount           int
	errorCount             int
	lockFilesForActionlint []string
	lockFilesForZizmor     []string
	lockFilesForDirTools   []string
}

// compileSpecificFiles compiles a specific list of workflow files
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

	state := &compileSpecificFilesState{}
	if err := compileSpecificFilesProcessAll(ctx, compiler, config, shouldValidate, stats, validationResults, state); err != nil {
		return state.workflowDataList, err
	}
	if err := compileSpecificFilesRunBatchTools(ctx, config, state); err != nil {
		return state.workflowDataList, err
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Display schedule warnings
	displayScheduleWarnings(compiler, config.JSONOutput)

	// Display safe update warnings (emitted as prompts for the calling agent)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)

	// Post-processing
	if err := runPostProcessing(compiler, state.workflowDataList, config, state.compiledCount); err != nil {
		return state.workflowDataList, err
	}

	// Output results
	if err := outputResults(stats, validationResults, config); err != nil {
		return state.workflowDataList, err
	}

	// Return error if any compilations failed
	// Don't return the detailed error message here since it's already printed in the summary
	// Returning a simple error prevents duplication in the output
	if state.errorCount > 0 {
		return state.workflowDataList, errors.New("compilation failed")
	}

	return state.workflowDataList, nil
}

func compileSpecificFilesProcessAll(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	shouldValidate bool,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	state *compileSpecificFilesState,
) error {
	for _, markdownFile := range config.MarkdownFiles {
		if err := ctx.Err(); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
			return err
		}
		compileSpecificFilesProcessOne(ctx, compiler, config, shouldValidate, stats, validationResults, state, markdownFile)
	}
	return nil
}

func compileSpecificFilesProcessOne(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	shouldValidate bool,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	state *compileSpecificFilesState,
	markdownFile string,
) {
	stats.Total++
	resolvedFile, ok := compileSpecificFilesResolveFile(config, stats, validationResults, state, markdownFile)
	if !ok {
		return
	}
	fileResult := compileWorkflowFile(ctx, compiler, resolvedFile, compileWorkflowFileOptions{
		verbose:    config.Verbose,
		jsonOutput: config.JSONOutput,
		noEmit:     config.NoEmit,
		strict:     config.Strict,
		validate:   shouldValidate,
	})
	compileSpecificFilesApplyResult(config, stats, state, resolvedFile, fileResult)
	*validationResults = append(*validationResults, fileResult.validationResult)
}

func compileSpecificFilesResolveFile(
	config CompileConfig,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	state *compileSpecificFilesState,
	markdownFile string,
) (string, bool) {
	result := ValidationResult{Workflow: markdownFile, Valid: true, Errors: []CompileValidationError{}, Warnings: []CompileValidationError{}}
	compileOrchestrationLog.Printf("Resolving workflow file: %s", markdownFile)
	resolvedFile, err := resolveWorkflowFile(markdownFile, config.Verbose)
	if err == nil {
		compileOrchestrationLog.Printf("Resolved to: %s", resolvedFile)
		return resolvedFile, true
	}
	state.errorCount++
	stats.Errors++
	trackWorkflowFailure(stats, markdownFile, 1, []string{err.Error()})
	result.Valid = false
	result.Errors = append(result.Errors, CompileValidationError{Type: "resolution_error", Message: err.Error()})
	*validationResults = append(*validationResults, result)
	return "", false
}

func compileSpecificFilesApplyResult(config CompileConfig, stats *CompilationStats, state *compileSpecificFilesState, resolvedFile string, fileResult compileWorkflowFileResult) {
	if !fileResult.success {
		errMsgs := compileSpecificFilesErrorMessages(fileResult)
		state.errorCount++
		stats.Errors += len(errMsgs)
		trackWorkflowFailure(stats, resolvedFile, len(errMsgs), errMsgs)
		return
	}
	state.compiledCount++
	if fileResult.workflowData != nil {
		state.workflowDataList = append(state.workflowDataList, fileResult.workflowData)
	}
	compileSpecificFilesCollectLockFiles(config, state, fileResult.lockFile)
}

func compileSpecificFilesErrorMessages(fileResult compileWorkflowFileResult) []string {
	var errMsgs []string
	for _, verr := range fileResult.validationResult.Errors {
		errMsgs = append(errMsgs, verr.Message)
	}
	if len(errMsgs) == 0 {
		errMsgs = []string{fallbackCompilationErrorMessage}
	}
	return errMsgs
}

func compileSpecificFilesCollectLockFiles(config CompileConfig, state *compileSpecificFilesState, lockFile string) {
	if config.NoEmit || lockFile == "" {
		return
	}
	if _, err := os.Stat(lockFile); err != nil {
		return
	}
	if config.Actionlint {
		state.lockFilesForActionlint = append(state.lockFilesForActionlint, lockFile)
	}
	if config.Zizmor {
		state.lockFilesForZizmor = append(state.lockFilesForZizmor, lockFile)
	}
	if config.Poutine || config.RunnerGuard {
		state.lockFilesForDirTools = append(state.lockFilesForDirTools, lockFile)
	}
}

func compileSpecificFilesRunBatchTools(ctx context.Context, config CompileConfig, state *compileSpecificFilesState) error {
	if config.Actionlint && !config.NoEmit && len(state.lockFilesForActionlint) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := RunActionlintOnFiles(ctx, state.lockFilesForActionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil && config.Strict {
			return err
		}
	}
	if config.Zizmor && !config.NoEmit && len(state.lockFilesForZizmor) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := RunZizmorOnFiles(state.lockFilesForZizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil && config.Strict {
			return err
		}
	}
	return compileSpecificFilesRunDirectoryTools(ctx, config, state)
}

func compileSpecificFilesRunDirectoryTools(ctx context.Context, config CompileConfig, state *compileSpecificFilesState) error {
	if len(state.lockFilesForDirTools) == 0 || config.NoEmit {
		return nil
	}
	workflowDir := filepath.Dir(state.lockFilesForDirTools[0])
	if config.Poutine {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runBatchDirectoryTool("poutine", workflowDir, config.Verbose && !config.JSONOutput, config.Strict, RunPoutineOnDirectory); err != nil && config.Strict {
			return err
		}
	}
	if config.RunnerGuard {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runBatchDirectoryTool("runner-guard", workflowDir, config.Verbose && !config.JSONOutput, config.Strict, RunRunnerGuardOnDirectory); err != nil && config.Strict {
			return err
		}
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
	gitRoot, workflowsDir, mdFiles, err := compileAllFilesInDirectoryInputs(config, workflowDir)
	if err != nil {
		return nil, err
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

	shouldValidate := compileAllFilesInDirectoryShouldValidate(config)
	state := &compileAllFilesInDirectoryState{}
	if err := compileAllFilesInDirectoryProcessAll(ctx, compiler, config, shouldValidate, mdFiles, stats, validationResults, state); err != nil {
		return state.workflowDataList, err
	}
	if err := compileAllFilesInDirectoryRunBatchTools(ctx, config, workflowsDir, state); err != nil {
		return state.workflowDataList, err
	}

	return compileAllFilesInDirectoryFinalize(compileAllFilesInDirectoryFinalizeParams{
		Ctx:               ctx,
		Compiler:          compiler,
		Config:            config,
		WorkflowsDir:      workflowsDir,
		GitRoot:           gitRoot,
		MDFiles:           mdFiles,
		PurgeData:         purgeData,
		Stats:             stats,
		ValidationResults: validationResults,
		State:             state,
	})
}

type compileAllFilesInDirectoryFinalizeParams struct {
	Ctx               context.Context
	Compiler          *workflow.Compiler
	Config            CompileConfig
	WorkflowsDir      string
	GitRoot           string
	MDFiles           []string
	PurgeData         *purgeTrackingData
	Stats             *CompilationStats
	ValidationResults *[]ValidationResult
	State             *compileAllFilesInDirectoryState
}

func compileAllFilesInDirectoryFinalize(p compileAllFilesInDirectoryFinalizeParams) ([]*workflow.WorkflowData, error) {
	// Emit recommendation when many slash commands are present without centralized strategy.
	displayCentralizedSlashCommandRecommendation(p.Compiler, p.State.workflowDataList, p.Config.JSONOutput)
	p.Stats.Warnings = p.Compiler.GetWarningCount()
	displayScheduleWarnings(p.Compiler, p.Config.JSONOutput)
	displaySafeUpdateWarnings(p.Compiler, p.Config.JSONOutput)
	if p.Config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d out of %d workflow files", p.State.successCount, len(p.MDFiles))))
	}
	if p.Config.Purge && p.PurgeData != nil {
		runPurgeOperations(p.WorkflowsDir, p.PurgeData, p.Config.Verbose)
	}
	if err := runPostProcessingForDirectory(p.Ctx, p.Compiler, p.State.workflowDataList, p.Config, p.WorkflowsDir, p.GitRoot, p.State.successCount, p.State.errorCount); err != nil {
		return p.State.workflowDataList, err
	}
	return compileAllFilesInDirectoryOutput(p.Config, p.MDFiles, p.Stats, p.ValidationResults, p.State)
}

func compileAllFilesInDirectoryOutput(
	config CompileConfig,
	mdFiles []string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	state *compileAllFilesInDirectoryState,
) ([]*workflow.WorkflowData, error) {
	// Output results.
	// Populate MarkdownFiles so that outputResults can collect per-workflow stats
	// (e.g. schedule heatmap) even when the caller did not specify explicit files.
	if config.Stats && len(config.MarkdownFiles) == 0 {
		config.MarkdownFiles = mdFiles
	}
	if err := outputResults(stats, validationResults, config); err != nil {
		return state.workflowDataList, err
	}

	// Return error if any compilations failed
	if state.errorCount > 0 {
		return state.workflowDataList, errors.New("compilation failed")
	}

	return state.workflowDataList, nil
}

func compileAllFilesInDirectoryInputs(config CompileConfig, workflowDir string) (string, string, []string, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", "", nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileOrchestrationLog.Printf("Found git root: %s", gitRoot)
	workflowsDir := filepath.Join(gitRoot, workflowDir)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return "", "", nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}
	compileOrchestrationLog.Printf("Scanning for markdown files in %s", workflowsDir)
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Scanning for markdown files in "+workflowsDir))
	}
	mdFiles, err := getMarkdownWorkflowFiles(workflowsDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to find markdown files: %w", err)
	}
	mdFiles, err = filterMarkdownFilesWithFrontmatter(mdFiles)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to filter markdown files: %w", err)
	}
	if len(mdFiles) == 0 {
		return "", "", nil, fmt.Errorf("no workflow markdown files found in %s (workflow files must start with a frontmatter opener on the first line)", workflowsDir)
	}
	return gitRoot, workflowsDir, mdFiles, nil
}

func compileAllFilesInDirectoryShouldValidate(config CompileConfig) bool {
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}
	return shouldValidate
}

func compileAllFilesInDirectoryProcessAll(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	shouldValidate bool,
	mdFiles []string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	state *compileAllFilesInDirectoryState,
) error {
	for _, file := range mdFiles {
		if err := ctx.Err(); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
			return err
		}
		compileAllFilesInDirectoryProcessOne(ctx, compiler, config, shouldValidate, stats, validationResults, state, file)
	}
	return nil
}

func compileAllFilesInDirectoryProcessOne(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	shouldValidate bool,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	state *compileAllFilesInDirectoryState,
	file string,
) {
	stats.Total++
	fileResult := compileWorkflowFile(ctx, compiler, file, compileWorkflowFileOptions{
		verbose:    config.Verbose,
		jsonOutput: config.JSONOutput,
		noEmit:     config.NoEmit,
		strict:     config.Strict,
		validate:   shouldValidate,
	})
	compileAllFilesInDirectoryApplyResult(config, stats, state, file, fileResult)
	*validationResults = append(*validationResults, fileResult.validationResult)
}

func compileAllFilesInDirectoryApplyResult(config CompileConfig, stats *CompilationStats, state *compileAllFilesInDirectoryState, file string, fileResult compileWorkflowFileResult) {
	if !fileResult.success {
		errMsgs := compileAllFilesInDirectoryErrorMessages(fileResult)
		state.errorCount++
		stats.Errors += len(errMsgs)
		trackWorkflowFailure(stats, file, len(errMsgs), errMsgs)
		return
	}
	state.successCount++
	if fileResult.workflowData != nil {
		state.workflowDataList = append(state.workflowDataList, fileResult.workflowData)
	}
	compileAllFilesInDirectoryCollectLockFiles(config, state, fileResult.lockFile)
}

func compileAllFilesInDirectoryErrorMessages(fileResult compileWorkflowFileResult) []string {
	var errMsgs []string
	for _, verr := range fileResult.validationResult.Errors {
		errMsgs = append(errMsgs, verr.Message)
	}
	if len(errMsgs) == 0 {
		errMsgs = []string{fallbackCompilationErrorMessage}
	}
	return errMsgs
}

func compileAllFilesInDirectoryCollectLockFiles(config CompileConfig, state *compileAllFilesInDirectoryState, lockFile string) {
	if config.NoEmit || lockFile == "" {
		return
	}
	if _, err := os.Stat(lockFile); err != nil {
		return
	}
	if config.Actionlint {
		state.lockFilesForActionlint = append(state.lockFilesForActionlint, lockFile)
	}
	if config.Zizmor {
		state.lockFilesForZizmor = append(state.lockFilesForZizmor, lockFile)
	}
	if config.Poutine || config.RunnerGuard {
		state.lockFilesForDirTools = append(state.lockFilesForDirTools, lockFile)
	}
}

func compileAllFilesInDirectoryRunBatchTools(ctx context.Context, config CompileConfig, workflowsDir string, state *compileAllFilesInDirectoryState) error {
	if config.Actionlint && !config.NoEmit && len(state.lockFilesForActionlint) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := RunActionlintOnFiles(ctx, state.lockFilesForActionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil && config.Strict {
			return err
		}
	}
	if config.Zizmor && !config.NoEmit && len(state.lockFilesForZizmor) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := RunZizmorOnFiles(state.lockFilesForZizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil && config.Strict {
			return err
		}
	}
	return compileAllFilesInDirectoryRunDirectoryTools(ctx, config, workflowsDir, state)
}

func compileAllFilesInDirectoryRunDirectoryTools(ctx context.Context, config CompileConfig, workflowsDir string, state *compileAllFilesInDirectoryState) error {
	if config.NoEmit || len(state.lockFilesForDirTools) == 0 {
		return nil
	}
	if config.Poutine {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runBatchDirectoryTool("poutine", workflowsDir, config.Verbose && !config.JSONOutput, config.Strict, RunPoutineOnDirectory); err != nil && config.Strict {
			return err
		}
	}
	if config.RunnerGuard {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runBatchDirectoryTool("runner-guard", workflowsDir, config.Verbose && !config.JSONOutput, config.Strict, RunRunnerGuardOnDirectory); err != nil && config.Strict {
			return err
		}
	}
	return nil
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

	if err := runPostProcessingForDirectoryDependabot(compiler, workflowDataList, config, workflowsDir, gitRoot); err != nil {
		return err
	}
	if err := runPostProcessingForDirectoryMaintenance(ctx, compiler, workflowDataList, config, workflowsDir, gitRoot); err != nil {
		return err
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

func runPostProcessingForDirectoryDependabot(
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	config CompileConfig,
	workflowsDir string,
	gitRoot string,
) error {
	if !config.Dependabot || config.NoEmit {
		return nil
	}
	absWorkflowDir := getAbsoluteWorkflowDir(workflowsDir, gitRoot)
	if err := generateDependabotManifestsWrapper(compiler, workflowDataList, absWorkflowDir, config.ForceOverwrite, config.Strict); err != nil && config.Strict {
		return err
	}
	if err := compiler.ReconcileManagedDependabotIgnoresInRepo(gitRoot); err != nil {
		if config.Strict {
			return err
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to reconcile compiler-managed Dependabot ignore entries: %v", err)))
	}
	return nil
}

func runPostProcessingForDirectoryMaintenance(
	ctx context.Context,
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	config CompileConfig,
	workflowsDir string,
	gitRoot string,
) error {
	if config.NoEmit || config.WorkflowDir != "" {
		return nil
	}
	absWorkflowDir := getAbsoluteWorkflowDir(workflowsDir, gitRoot)
	if err := generateMaintenanceWorkflowWrapper(ctx, compiler, workflowDataList, absWorkflowDir, gitRoot, config.Verbose, config.Strict); err != nil && config.Strict {
		return err
	}
	if err := generateCentralSlashCommandWorkflowWrapper(ctx, workflowDataList, absWorkflowDir, gitRoot, config.Strict); err != nil && config.Strict {
		return err
	}
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
