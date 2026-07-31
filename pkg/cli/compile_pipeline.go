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
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileOrchestrationLog = logger.New("cli:compile_pipeline")
var runBatchYamllintOnFiles = RunYamllintOnFiles

const fallbackCompilationErrorMessage = "compilation failed (no detailed error message available)"

// batchToolLockFiles groups generated lock-file paths by the security tool that will consume them.
type batchToolLockFiles struct {
	Actionlint []string
	Zizmor     []string
	DirTools   []string // poutine and runner-guard (directory-based tools)
	Syft       []string
	Grype      []string
	Grant      []string
	Yamllint   []string
}

// addLockFileToSets appends lockFile to each tool's slice based on the enabled tools in config.
func addLockFileToSets(config CompileConfig, lockFile string, sets *batchToolLockFiles) {
	if _, err := os.Stat(lockFile); err != nil {
		return
	}
	if config.Actionlint {
		sets.Actionlint = append(sets.Actionlint, lockFile)
	}
	if config.Zizmor {
		sets.Zizmor = append(sets.Zizmor, lockFile)
	}
	if config.Poutine || config.RunnerGuard {
		sets.DirTools = append(sets.DirTools, lockFile)
	}
	if config.Syft {
		sets.Syft = append(sets.Syft, lockFile)
	}
	if config.Grype {
		sets.Grype = append(sets.Grype, lockFile)
	}
	if config.Grant {
		sets.Grant = append(sets.Grant, lockFile)
	}
	if config.Yamllint {
		sets.Yamllint = append(sets.Yamllint, lockFile)
	}
}

// compileBatchLoopResult holds the accumulated output of compileBatchLoop.
type compileBatchLoopResult struct {
	workflowDataList []*workflow.WorkflowData
	compiledCount    int
	errorCount       int
}

// compileBatchLoop compiles each file in files, updating stats and validationResults in place.
// It returns early with the context error if the context is cancelled.
func compileBatchLoop(
	ctx context.Context,
	compiler *workflow.Compiler,
	files []string,
	config CompileConfig,
	shouldValidate bool,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	sets *batchToolLockFiles,
) (compileBatchLoopResult, error) {
	var res compileBatchLoopResult
	for _, file := range files {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessageStderr("Operation cancelled"))
			return res, ctx.Err()
		default:
		}
		stats.Total++
		fileResult := compileWorkflowFile(ctx, compiler, file, compileWorkflowFileOptions{
			verbose: config.Verbose, jsonOutput: config.JSONOutput,
			noEmit: config.NoEmit, strict: config.Strict, validate: shouldValidate,
		})
		if !fileResult.success {
			var errMsgs []string
			for _, verr := range fileResult.validationResult.Errors {
				errMsgs = append(errMsgs, verr.Message)
			}
			if len(errMsgs) == 0 {
				errMsgs = []string{fallbackCompilationErrorMessage}
			}
			res.errorCount++
			stats.Errors += len(errMsgs)
			trackWorkflowFailure(stats, file, len(errMsgs), errMsgs)
		} else {
			res.compiledCount++
			stats.Succeeded++
			if fileResult.workflowData != nil {
				res.workflowDataList = append(res.workflowDataList, fileResult.workflowData)
			}
			if !config.NoEmit && fileResult.lockFile != "" {
				addLockFileToSets(config, fileResult.lockFile, sets)
			}
		}
		*validationResults = append(*validationResults, fileResult.validationResult)
	}
	return res, nil
}

// runBatchContainerTools runs the container image security tools (syft, grype, grant).
// It returns a non-nil strictGrantErr when grant fails in strict mode.
func runBatchContainerTools(
	ctx context.Context,
	config CompileConfig,
	sets batchToolLockFiles,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) (strictGrantErr error, err error) {
	if config.Syft && !config.NoEmit && len(sets.Syft) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := RunSyftOnLockFiles(sets.Syft, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return nil, err
			}
		}
	}
	if config.Grype && !config.NoEmit && len(sets.Grype) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := RunGrypeOnLockFiles(sets.Grype, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return nil, err
			}
		}
	}
	if config.Grant && !config.NoEmit && len(sets.Grant) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := RunGrantOnLockFiles(sets.Grant, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				stats.Errors++
				*validationResults = append(*validationResults, ValidationResult{
					Workflow: "grant",
					Valid:    false,
					Errors: []CompileValidationError{{
						Type:    "grant_error",
						Message: err.Error(),
					}},
				})
				strictGrantErr = err
			}
		}
	}
	return strictGrantErr, nil
}

// runBatchSecurityTools runs all enabled batch security tools (actionlint, zizmor, poutine,
// runner-guard, syft, grype, grant, yamllint) against the collected lock-file sets.
// workflowDir is the directory passed to directory-scoped tools (poutine, runner-guard).
// On grant failure in strict mode it updates stats, appends a ValidationResult, and returns
// a non-nil strictGrantErr; a non-nil second return value signals a hard stop.
func runBatchSecurityTools(
	ctx context.Context,
	config CompileConfig,
	sets batchToolLockFiles,
	workflowDir string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) (strictGrantErr error, err error) {
	if config.Actionlint && !config.NoEmit && len(sets.Actionlint) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := RunActionlintOnFiles(ctx, sets.Actionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return nil, err
			}
		}
	}
	if config.Zizmor && !config.NoEmit && len(sets.Zizmor) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := RunZizmorOnFiles(sets.Zizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return nil, err
			}
		}
	}
	if config.Poutine && !config.NoEmit && len(sets.DirTools) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := runBatchDirectoryTool("poutine", workflowDir, config.Verbose && !config.JSONOutput, config.Strict, RunPoutineOnDirectory); err != nil {
			if config.Strict {
				return nil, err
			}
		}
	}
	if config.RunnerGuard && !config.NoEmit && len(sets.DirTools) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := runBatchDirectoryTool("runner-guard", workflowDir, config.Verbose && !config.JSONOutput, config.Strict, RunRunnerGuardOnDirectory); err != nil {
			if config.Strict {
				return nil, err
			}
		}
	}
	strictGrantErr, err = runBatchContainerTools(ctx, config, sets, stats, validationResults)
	if err != nil {
		return strictGrantErr, err
	}
	if config.Yamllint && !config.NoEmit && len(sets.Yamllint) > 0 {
		if err := ctx.Err(); err != nil {
			return strictGrantErr, err
		}
		if err := runBatchYamllintOnFiles(sets.Yamllint, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return strictGrantErr, err
			}
		}
	}
	return strictGrantErr, nil
}

// resolveWorkflowFilesForCompile resolves each file in config.MarkdownFiles to its on-disk path.
// Resolution errors are recorded in stats and validationResults; errorCount is the number of
// files that could not be resolved.
func resolveWorkflowFilesForCompile(
	ctx context.Context,
	config CompileConfig,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) (resolvedFiles []string, errorCount int) {
	for _, markdownFile := range config.MarkdownFiles {
		if ctx.Err() != nil {
			return resolvedFiles, errorCount
		}
		stats.Total++
		resolvedFile, err := resolveWorkflowFile(markdownFile, config.Verbose)
		if err != nil {
			errorCount++
			stats.Errors++
			trackWorkflowFailure(stats, markdownFile, 1, []string{err.Error()})
			*validationResults = append(*validationResults, ValidationResult{
				Workflow: markdownFile,
				Valid:    false,
				Errors:   []CompileValidationError{{Type: "resolution_error", Message: err.Error()}},
				Warnings: []CompileValidationError{},
			})
			continue
		}
		resolvedFiles = append(resolvedFiles, resolvedFile)
	}
	return
}

// discoverWorkflowMarkdownFiles finds and filters all workflow markdown files in workflowsDir.
func discoverWorkflowMarkdownFiles(workflowsDir string, config CompileConfig) ([]string, error) {
	compileOrchestrationLog.Printf("Scanning for markdown files in %s", workflowsDir)
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr("Scanning for markdown files in "+workflowsDir))
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
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(fmt.Sprintf("Found %d markdown files to compile", len(mdFiles))))
	}
	return mdFiles, nil
}

// resolveWorkflowsDirectory finds the git root, constructs the absolute workflows directory
// path, and discovers the markdown workflow files within it.
func resolveWorkflowsDirectory(workflowDir string, config CompileConfig) (workflowsDir, gitRoot string, mdFiles []string, err error) {
	gitRoot, err = gitutil.FindGitRoot()
	if err != nil {
		return "", "", nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileOrchestrationLog.Printf("Found git root: %s", gitRoot)
	workflowsDir = filepath.Join(gitRoot, workflowDir)
	if _, statErr := os.Stat(workflowsDir); os.IsNotExist(statErr) {
		return "", "", nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}
	mdFiles, err = discoverWorkflowMarkdownFiles(workflowsDir, config)
	return
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

	batchMode := !config.Verbose && len(config.MarkdownFiles) > 1
	compiler.SetBatchMode(batchMode)
	compiler.SetQuiet(batchMode)

	// Enable validation automatically when force-refresh-action-pins is used
	// to verify all resolved action SHAs are valid
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	// Resolve each markdown file to its actual path, collecting per-file resolution errors.
	resolvedFiles, errorCount := resolveWorkflowFilesForCompile(ctx, config, stats, validationResults)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var sets batchToolLockFiles
	batchRes, ctxErr := compileBatchLoop(ctx, compiler, resolvedFiles, config, shouldValidate, stats, validationResults, &sets)
	if ctxErr != nil {
		return batchRes.workflowDataList, ctxErr
	}
	errorCount += batchRes.errorCount
	workflowDir := ""
	if len(sets.DirTools) > 0 {
		workflowDir = filepath.Dir(sets.DirTools[0])
	}
	strictGrantErr, toolsErr := runBatchSecurityTools(ctx, config, sets, workflowDir, stats, validationResults)
	if toolsErr != nil {
		return batchRes.workflowDataList, toolsErr
	}

	stats.Warnings = compiler.GetWarningCount()
	displayBatchCompilationNotices(compiler, config)
	displayScheduleWarnings(compiler, config.JSONOutput)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)

	if err := runPostProcessing(compiler, batchRes.workflowDataList, config, batchRes.compiledCount); err != nil {
		return batchRes.workflowDataList, err
	}
	if err := outputResults(stats, validationResults, config); err != nil {
		return batchRes.workflowDataList, err
	}

	if errorCount > 0 {
		if strictGrantErr != nil {
			return batchRes.workflowDataList, strictGrantErr
		}
		return batchRes.workflowDataList, errors.New("compilation failed")
	}
	return batchRes.workflowDataList, nil
}

// compileDirDisplayPhase emits batch notices, warnings and the optional verbose success message.
func compileDirDisplayPhase(compiler *workflow.Compiler, batchRes compileBatchLoopResult, config CompileConfig, mdFileCount int) {
	displayCentralizedSlashCommandRecommendation(compiler, batchRes.workflowDataList, config.JSONOutput)
	displayBatchCompilationNotices(compiler, config)
	displayScheduleWarnings(compiler, config.JSONOutput)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessageStderr(fmt.Sprintf("Successfully compiled %d out of %d workflow files", batchRes.compiledCount, mdFileCount)))
	}
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
	workflowsDir, gitRoot, mdFiles, err := resolveWorkflowsDirectory(workflowDir, config)
	if err != nil {
		return nil, err
	}

	batchMode := !config.Verbose && len(mdFiles) > 1
	compiler.SetBatchMode(batchMode)
	compiler.SetQuiet(batchMode)

	var purgeData *purgeTrackingData
	if config.Purge {
		purgeData = collectPurgeData(workflowsDir, mdFiles, config.Verbose)
	}

	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	var sets batchToolLockFiles
	batchRes, ctxErr := compileBatchLoop(ctx, compiler, mdFiles, config, shouldValidate, stats, validationResults, &sets)
	if ctxErr != nil {
		return batchRes.workflowDataList, ctxErr
	}

	strictGrantErr, toolsErr := runBatchSecurityTools(ctx, config, sets, workflowsDir, stats, validationResults)
	if toolsErr != nil {
		return batchRes.workflowDataList, toolsErr
	}

	stats.Warnings = compiler.GetWarningCount()
	compileDirDisplayPhase(compiler, batchRes, config, len(mdFiles))

	if config.Purge && purgeData != nil {
		runPurgeOperations(workflowsDir, purgeData, config.Verbose)
	}

	if err := runPostProcessingForDirectory(ctx, compiler, batchRes.workflowDataList, config, workflowsDir, gitRoot, batchRes.compiledCount, batchRes.errorCount); err != nil {
		return batchRes.workflowDataList, err
	}

	if config.Stats && len(config.MarkdownFiles) == 0 {
		config.MarkdownFiles = mdFiles
	}
	if err := outputResults(stats, validationResults, config); err != nil {
		return batchRes.workflowDataList, err
	}

	if batchRes.errorCount > 0 {
		if strictGrantErr != nil {
			return batchRes.workflowDataList, strictGrantErr
		}
		return batchRes.workflowDataList, errors.New("compilation failed")
	}
	return batchRes.workflowDataList, nil
}

func displayBatchCompilationNotices(compiler *workflow.Compiler, config CompileConfig) {
	if config.JSONOutput || config.Verbose {
		return
	}

	featureUsage := compiler.GetExperimentalFeatureUsage()
	if len(featureUsage) > 0 {
		type featureCount struct {
			name  string
			count int
		}
		features := make([]featureCount, 0, len(featureUsage))
		for message, count := range featureUsage {
			features = append(features, featureCount{
				name:  strings.TrimPrefix(message, "Using experimental feature: "),
				count: count,
			})
		}
		slices.SortFunc(features, func(a, b featureCount) int {
			if a.count != b.count {
				if a.count > b.count {
					return -1
				}
				return 1
			}
			if a.name < b.name {
				return -1
			}
			if a.name > b.name {
				return 1
			}
			return 0
		})

		fmt.Fprintln(os.Stderr, console.FormatWarningMessageStderr("Experimental features in use:"))
		for _, feature := range features {
			fmt.Fprintln(os.Stderr, console.FormatListItemStderr(fmt.Sprintf("%s: %s", feature.name, formatWorkflowCount(feature.count))))
		}
	}

	if compiler.CopilotRequestsTipNeeded() {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(
			"Copilot token-based inference may be available: add permissions.copilot-requests: write. "+
				"See https://github.github.com/gh-aw/reference/billing/",
		))
	}
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
			fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(fmt.Sprintf("Found %d existing .lock.yml files", len(data.existingLockFiles))))
		}
		if len(data.existingInvalidFiles) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(fmt.Sprintf("Found %d existing .invalid.yml files", len(data.existingInvalidFiles))))
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
				fmt.Fprintln(os.Stderr, console.FormatWarningMessageStderr(fmt.Sprintf("Failed to reconcile compiler-managed Dependabot ignore entries: %v", err)))
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
			fmt.Fprintln(os.Stderr, console.FormatWarningMessageStderr(fmt.Sprintf("Failed to reconcile compiler-managed Dependabot ignore entries: %v", err)))
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
