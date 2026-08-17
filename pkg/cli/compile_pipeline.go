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
	"cmp"
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

// batchLockFiles collects the compiled lock files that each batched post-compilation
// tool should analyze.
type batchLockFiles struct {
	actionlint []string
	zizmor     []string
	dirTools   []string // lock files for directory-based tools (poutine, runner-guard)
	syft       []string // lock files for syft container image SBOM scanning
	grype      []string // lock files for grype container image vulnerability scanning
	grant      []string // lock files for grant container image license scanning
	yamllint   []string // lock files for yamllint YAML linter
	shellcheck []string // lock files for shellcheck run step linting
}

// collect records a compiled lock file for each batch tool enabled in config.
func (b *batchLockFiles) collect(config CompileConfig, lockFile string) {
	if config.NoEmit || lockFile == "" {
		return
	}
	if _, err := os.Stat(lockFile); err != nil {
		return
	}
	if config.Actionlint {
		b.actionlint = append(b.actionlint, lockFile)
	}
	if config.Zizmor {
		b.zizmor = append(b.zizmor, lockFile)
	}
	if config.Poutine || config.RunnerGuard {
		b.dirTools = append(b.dirTools, lockFile)
	}
	if config.Syft {
		b.syft = append(b.syft, lockFile)
	}
	if config.Grype {
		b.grype = append(b.grype, lockFile)
	}
	if config.Grant {
		b.grant = append(b.grant, lockFile)
	}
	if config.Yamllint {
		b.yamllint = append(b.yamllint, lockFile)
	}
	if config.shellcheckEnabled() {
		b.shellcheck = append(b.shellcheck, lockFile)
	}
}

// batchToolsResult reports the outcome of the batched post-compilation tools.
// fatalErr aborts the compilation; strictGrantErr is surfaced only when other
// compilation errors were recorded; extraErrors counts additional error entries.
type batchToolsResult struct {
	extraErrors    int
	strictGrantErr error
	fatalErr       error
}

// runBatchTools runs the batched post-compilation tools over the collected lock files.
// dirToolsDir is the directory scanned by directory-based tools; when empty it is
// derived from the first collected lock file.
func (b *batchLockFiles) runBatchTools(
	ctx context.Context,
	config CompileConfig,
	dirToolsDir string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) batchToolsResult {
	var result batchToolsResult
	toolVerbose := config.Verbose && !config.JSONOutput

	// Run batch actionlint on all collected lock files
	if config.Actionlint && !config.NoEmit && len(b.actionlint) > 0 {
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := RunActionlintOnFiles(ctx, b.actionlint, toolVerbose, config.Strict); err != nil {
			if config.Strict {
				result.fatalErr = err
				return result
			}
		}
	}

	// Run batch zizmor on all collected lock files
	if config.Zizmor && !config.NoEmit && len(b.zizmor) > 0 {
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := RunZizmorOnFiles(b.zizmor, toolVerbose, config.Strict); err != nil {
			// Always fail on high/critical severity findings (zizmor returns errors for those
			// regardless of strict mode). In strict mode, all findings are errors.
			result.fatalErr = err
			return result
		}
	}

	// Run batch poutine and runner-guard once on the workflow directory
	if len(b.dirTools) > 0 && dirToolsDir == "" {
		dirToolsDir = filepath.Dir(b.dirTools[0])
	}
	for _, dirTool := range []struct {
		name    string
		enabled bool
		run     func(string, bool, bool) error
	}{
		{"poutine", config.Poutine, RunPoutineOnDirectory},
		{"runner-guard", config.RunnerGuard, RunRunnerGuardOnDirectory},
	} {
		if !dirTool.enabled || config.NoEmit || len(b.dirTools) == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := runBatchDirectoryTool(dirTool.name, dirToolsDir, toolVerbose, config.Strict, dirTool.run); err != nil {
			if config.Strict {
				result.fatalErr = err
				return result
			}
		}
	}

	// Run the container image scanners on the compiled lock files.
	for _, imageTool := range []struct {
		enabled   bool
		lockFiles []string
		run       func([]string, bool, bool) error
	}{
		{config.Syft, b.syft, RunSyftOnLockFiles},
		{config.Grype, b.grype, RunGrypeOnLockFiles},
	} {
		if !imageTool.enabled || config.NoEmit || len(imageTool.lockFiles) == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := imageTool.run(imageTool.lockFiles, toolVerbose, config.Strict); err != nil {
			if config.Strict {
				result.fatalErr = err
				return result
			}
		}
	}

	// Run grant license scanner on container images referenced in the compiled lock files.
	if config.Grant && !config.NoEmit && len(b.grant) > 0 {
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := RunGrantOnLockFiles(b.grant, toolVerbose, config.Strict); err != nil {
			if config.Strict {
				result.extraErrors++
				stats.Errors++
				// Grant is a post-compilation tool, not a workflow; record it only in
				// validationResults (for JSON output) without adding to FailureDetails.
				*validationResults = append(*validationResults, ValidationResult{
					Workflow: "grant",
					Valid:    false,
					Errors: []ValidationIssue{{
						Type:    "grant_error",
						Message: err.Error(),
					}},
				})
				result.strictGrantErr = err
			}
		}
	}

	// Run yamllint on all collected lock files.
	if config.Yamllint && !config.NoEmit && len(b.yamllint) > 0 {
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := runBatchYamllintOnFiles(b.yamllint, toolVerbose, config.Strict); err != nil {
			if config.Strict {
				result.fatalErr = err
				return result
			}
		}
	}

	// Run shellcheck on run step scripts in all collected lock files.
	if config.shellcheckEnabled() && !config.NoEmit && len(b.shellcheck) > 0 {
		if err := ctx.Err(); err != nil {
			result.fatalErr = err
			return result
		}
		if err := RunShellcheckOnLockFiles(ctx, b.shellcheck, toolVerbose, config.Strict); err != nil {
			if config.Strict {
				result.fatalErr = err
				return result
			}
		}
	}

	return result
}

// recordCompiledFileResult folds a single file's compilation result into the stats,
// failure counters, and batch tool lock file lists. sourceFile is the file path used
// for failure tracking.
func recordCompiledFileResult(
	fileResult compileWorkflowFileResult,
	sourceFile string,
	config CompileConfig,
	stats *CompilationStats,
	lockFiles *batchLockFiles,
	compiledCount, errorCount *int,
) {
	if !fileResult.success {
		// Collect error messages from validation result for display in summary
		var errMsgs []string
		for _, verr := range fileResult.validationResult.Errors {
			errMsgs = append(errMsgs, verr.Message)
		}
		if len(errMsgs) == 0 {
			errMsgs = []string{fallbackCompilationErrorMessage}
		}
		*errorCount++
		stats.Errors += len(errMsgs)
		trackWorkflowFailure(stats, sourceFile, len(errMsgs), errMsgs)
		return
	}
	*compiledCount++
	stats.Succeeded++
	lockFiles.collect(config, fileResult.lockFile)
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

	var workflowDataList []*workflow.WorkflowData
	var compiledCount int
	var errorCount int
	lockFiles := &batchLockFiles{}

	// Compile each specified file
	for _, markdownFile := range config.MarkdownFiles {
		// Respect context cancellation between files (e.g. Ctrl+C)
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessageStderr("Operation cancelled"))
			return workflowDataList, ctx.Err()
		default:
		}

		stats.Total++

		// Resolve workflow ID or file path to actual file path
		compileOrchestrationLog.Printf("Resolving workflow file: %s", markdownFile)
		resolvedFile, err := resolveWorkflowFile(markdownFile, config.Verbose)
		if err != nil {
			// Don't print error here - it will be displayed in the compilation summary
			// The error is stored in ValidationResult for JSON output and returned for main to display
			errorCount++
			stats.Errors++
			trackWorkflowFailure(stats, markdownFile, 1, []string{err.Error()})
			*validationResults = append(*validationResults, ValidationResult{
				Workflow: markdownFile,
				Valid:    false,
				Errors: []ValidationIssue{{
					Type:    "resolution_error",
					Message: err.Error(),
				}},
				Warnings: []ValidationIssue{},
			})
			continue
		}
		compileOrchestrationLog.Printf("Resolved to: %s", resolvedFile)

		// Compile regular workflow file (disable per-file security tools)
		fileResult := compileWorkflowFile(
			ctx, compiler, resolvedFile, compileWorkflowFileOptions{
				verbose:    config.Verbose,
				jsonOutput: config.JSONOutput,
				noEmit:     config.NoEmit,
				strict:     config.Strict,
				validate:   shouldValidate,
				// zizmor, poutine, actionlint disabled per-file (batched instead)
			},
		)

		recordCompiledFileResult(fileResult, resolvedFile, config, stats, lockFiles, &compiledCount, &errorCount)
		if fileResult.success && fileResult.workflowData != nil {
			workflowDataList = append(workflowDataList, fileResult.workflowData)
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	batchResult := lockFiles.runBatchTools(ctx, config, "", stats, validationResults)
	if batchResult.fatalErr != nil {
		return workflowDataList, batchResult.fatalErr
	}
	errorCount += batchResult.extraErrors

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Aggregate and display batch-mode notices (experimental features, Copilot tip)
	displayBatchCompilationNotices(compiler, config)

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
		if batchResult.strictGrantErr != nil {
			return workflowDataList, batchResult.strictGrantErr
		}
		return workflowDataList, errors.New("compilation failed")
	}

	return workflowDataList, nil
}

// resolveWorkflowsDirectory locates the git root and the workflow markdown files
// to compile inside workflowDir.
func resolveWorkflowsDirectory(workflowDir string, verbose bool) (gitRoot string, workflowsDir string, mdFiles []string, err error) {
	// Find git root for consistent behavior
	gitRoot, err = gitutil.FindGitRoot()
	if err != nil {
		return "", "", nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileOrchestrationLog.Printf("Found git root: %s", gitRoot)

	// Compile all markdown files in the specified workflow directory
	workflowsDir = filepath.Join(gitRoot, workflowDir)
	if _, statErr := os.Stat(workflowsDir); os.IsNotExist(statErr) {
		return "", "", nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}

	compileOrchestrationLog.Printf("Scanning for markdown files in %s", workflowsDir)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr("Scanning for markdown files in "+workflowsDir))
	}

	// Find and filter markdown files (shared helper keeps logic in one place)
	mdFiles, err = getMarkdownWorkflowFiles(workflowsDir)
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

	compileOrchestrationLog.Printf("Found %d markdown files to compile", len(mdFiles))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(fmt.Sprintf("Found %d markdown files to compile", len(mdFiles))))
	}

	return gitRoot, workflowsDir, mdFiles, nil
}

// directoryCompilationOutcome carries the per-directory compilation counters needed
// by the post-compilation reporting steps.
type directoryCompilationOutcome struct {
	successCount int
	errorCount   int
	mdFiles      []string
	gitRoot      string
	workflowsDir string
	purgeData    *purgeTrackingData
}

// finalizeDirectoryCompilation runs the display, purge, post-processing and output
// steps that follow a directory compilation.
func finalizeDirectoryCompilation(
	ctx context.Context,
	compiler *workflow.Compiler,
	config CompileConfig,
	workflowDataList []*workflow.WorkflowData,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
	outcome directoryCompilationOutcome,
) error {
	// Emit recommendation when many slash commands are present without centralized strategy.
	displayCentralizedSlashCommandRecommendation(compiler, workflowDataList, config.JSONOutput)

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	displayBatchCompilationNotices(compiler, config)

	// Display schedule warnings
	displayScheduleWarnings(compiler, config.JSONOutput)

	// Display safe update warnings (emitted as prompts for the calling agent)
	displaySafeUpdateWarnings(compiler, config.JSONOutput)

	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessageStderr(fmt.Sprintf("Successfully compiled %d out of %d workflow files", outcome.successCount, len(outcome.mdFiles))))
	}

	// Handle purge logic if requested
	if config.Purge && outcome.purgeData != nil {
		runPurgeOperations(outcome.workflowsDir, outcome.purgeData, config.Verbose)
	}

	// Post-processing
	if err := runPostProcessingForDirectory(ctx, compiler, workflowDataList, config, outcome.workflowsDir, outcome.gitRoot, outcome.successCount, outcome.errorCount); err != nil {
		return err
	}

	// Output results.
	// Populate MarkdownFiles so that outputResults can collect per-workflow stats
	// (e.g. schedule heatmap) even when the caller did not specify explicit files.
	if config.Stats && len(config.MarkdownFiles) == 0 {
		config.MarkdownFiles = outcome.mdFiles
	}
	return outputResults(stats, validationResults, config)
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
	gitRoot, workflowsDir, mdFiles, err := resolveWorkflowsDirectory(workflowDir, config.Verbose)
	if err != nil {
		return nil, err
	}

	batchMode := !config.Verbose && len(mdFiles) > 1
	compiler.SetBatchMode(batchMode)
	compiler.SetQuiet(batchMode)

	// Handle purge logic: collect existing files before compilation
	var purgeData *purgeTrackingData
	if config.Purge {
		purgeData, err = collectPurgeData(workflowsDir, mdFiles, config.Verbose)
		if err != nil {
			return nil, fmt.Errorf("failed to collect existing files for purge: %w", err)
		}
	}

	// Enable validation automatically when force-refresh-action-pins is used
	// to verify all resolved action SHAs are valid
	shouldValidate := config.Validate || config.ForceRefreshActionPins
	if config.ForceRefreshActionPins && !config.Validate {
		compileOrchestrationLog.Print("Automatically enabling action SHA validation due to --force-refresh-action-pins")
	}

	// Compile each file
	var workflowDataList []*workflow.WorkflowData
	var successCount int
	var errorCount int
	lockFiles := &batchLockFiles{}

	for _, file := range mdFiles {
		// Respect context cancellation between files (e.g. Ctrl+C)
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessageStderr("Operation cancelled"))
			return workflowDataList, ctx.Err()
		default:
		}

		stats.Total++

		// Compile regular workflow file (disable per-file security tools)
		fileResult := compileWorkflowFile(
			ctx, compiler, file, compileWorkflowFileOptions{
				verbose:    config.Verbose,
				jsonOutput: config.JSONOutput,
				noEmit:     config.NoEmit,
				strict:     config.Strict,
				validate:   shouldValidate,
				// zizmor, poutine, actionlint disabled per-file (batched instead)
			},
		)

		recordCompiledFileResult(fileResult, file, config, stats, lockFiles, &successCount, &errorCount)
		if fileResult.success && fileResult.workflowData != nil {
			workflowDataList = append(workflowDataList, fileResult.workflowData)
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	batchResult := lockFiles.runBatchTools(ctx, config, workflowsDir, stats, validationResults)
	if batchResult.fatalErr != nil {
		return workflowDataList, batchResult.fatalErr
	}
	errorCount += batchResult.extraErrors

	if err := finalizeDirectoryCompilation(ctx, compiler, config, workflowDataList, stats, validationResults, directoryCompilationOutcome{
		successCount: successCount,
		errorCount:   errorCount,
		mdFiles:      mdFiles,
		gitRoot:      gitRoot,
		workflowsDir: workflowsDir,
		purgeData:    purgeData,
	}); err != nil {
		return workflowDataList, err
	}

	// Return error if any compilations failed
	if errorCount > 0 {
		if batchResult.strictGrantErr != nil {
			return workflowDataList, batchResult.strictGrantErr
		}
		return workflowDataList, errors.New("compilation failed")
	}

	return workflowDataList, nil
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
				return cmp.Compare(b.count, a.count)
			}
			return cmp.Compare(a.name, b.name)
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
func collectPurgeData(workflowsDir string, mdFiles []string, verbose bool) (*purgeTrackingData, error) {
	return collectPurgeDataWithPatterns(workflowsDir, mdFiles, verbose, "*.lock.yml", "*.invalid.yml")
}

// collectPurgeDataWithPatterns is the testable implementation of collectPurgeData.
// lockPattern and invalidPattern are appended to workflowsDir for the glob calls.
func collectPurgeDataWithPatterns(workflowsDir string, mdFiles []string, verbose bool, lockPattern, invalidPattern string) (*purgeTrackingData, error) {
	data := &purgeTrackingData{}

	// Find all existing files
	var err error
	data.existingLockFiles, err = filepath.Glob(filepath.Join(workflowsDir, lockPattern))
	if err != nil {
		return nil, fmt.Errorf("failed to glob existing .lock.yml files in %s: %w", workflowsDir, err)
	}
	data.existingInvalidFiles, err = filepath.Glob(filepath.Join(workflowsDir, invalidPattern))
	if err != nil {
		return nil, fmt.Errorf("failed to glob existing .invalid.yml files in %s: %w", workflowsDir, err)
	}

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

	return data, nil
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
