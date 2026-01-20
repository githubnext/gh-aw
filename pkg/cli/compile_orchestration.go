// Package cli provides main orchestration logic for workflow compilation.
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/stringutil"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var compileOrchestrationLog = logger.New("cli:compile_orchestration")

// compileSpecificFiles compiles a specific list of workflow files
func compileSpecificFiles(
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

	var workflowDataList []*workflow.WorkflowData
	var compiledCount int
	var errorCount int
	var errorMessages []string
	var lockFilesForActionlint []string
	var lockFilesForZizmor []string
	var campaignFiles []string

	// Compile each specified file
	for _, markdownFile := range config.MarkdownFiles {
		stats.Total++

		// Initialize validation result
		result := ValidationResult{
			Workflow: markdownFile,
			Valid:    true,
			Errors:   []CompileValidationError{},
			Warnings: []CompileValidationError{},
		}

		// Resolve workflow ID or file path to actual file path
		compileOrchestrationLog.Printf("Resolving workflow file: %s", markdownFile)
		resolvedFile, err := resolveWorkflowFile(markdownFile, config.Verbose)
		if err != nil {
			if !config.JSONOutput {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
			}
			errorMessages = append(errorMessages, err.Error())
			errorCount++
			stats.Errors++
			trackWorkflowFailure(stats, markdownFile, 1)
			result.Valid = false
			result.Errors = append(result.Errors, CompileValidationError{
				Type:    "resolution_error",
				Message: err.Error(),
			})
			*validationResults = append(*validationResults, result)
			continue
		}
		compileOrchestrationLog.Printf("Resolved to: %s", resolvedFile)

		// Update result with resolved file name
		result.Workflow = filepath.Base(resolvedFile)

		// Handle campaign spec files separately
		if strings.HasSuffix(resolvedFile, ".campaign.md") {
			campaignFiles = append(campaignFiles, resolvedFile)
			campaignResult, success := processCampaignSpec(ProcessCampaignSpecOptions{
				Compiler:     compiler,
				ResolvedFile: resolvedFile,
				Verbose:      config.Verbose,
				JSONOutput:   config.JSONOutput,
				NoEmit:       config.NoEmit,
				Zizmor:       false,
				Poutine:      false,
				Actionlint:   false,
				Strict:       config.Strict,
				Validate:     shouldValidate,
			})
			if !success {
				errorCount++
				stats.Errors++
				trackWorkflowFailure(stats, resolvedFile, len(campaignResult.Errors))
				errorMessages = append(errorMessages, campaignResult.Errors[0].Message)
			}
			*validationResults = append(*validationResults, campaignResult)
			continue
		}

		// Compile regular workflow file (disable per-file security tools)
		fileResult := compileWorkflowFile(
			compiler, resolvedFile, config.Verbose, config.JSONOutput,
			config.NoEmit, false, false, false, // Disable per-file security tools
			config.Strict, shouldValidate,
		)

		if !fileResult.success {
			errorCount++
			stats.Errors++
			trackWorkflowFailure(stats, resolvedFile, 1)
			errorMessages = append(errorMessages, fileResult.validationResult.Errors[0].Message)
		} else {
			compiledCount++
			workflowDataList = append(workflowDataList, fileResult.workflowData)

			// Collect lock files for batch security tools
			if !config.NoEmit && fileResult.lockFile != "" {
				if _, err := os.Stat(fileResult.lockFile); err == nil {
					if config.Actionlint {
						lockFilesForActionlint = append(lockFilesForActionlint, fileResult.lockFile)
					}
					if config.Zizmor {
						lockFilesForZizmor = append(lockFilesForZizmor, fileResult.lockFile)
					}
				}
			}
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	// Run batch actionlint on all collected lock files
	if config.Actionlint && !config.NoEmit && len(lockFilesForActionlint) > 0 {
		if err := runBatchActionlint(lockFilesForActionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch zizmor on all collected lock files
	if config.Zizmor && !config.NoEmit && len(lockFilesForZizmor) > 0 {
		if err := runBatchZizmor(lockFilesForZizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch poutine once on the workflow directory
	// Get the directory from the first lock file (all should be in same directory)
	if config.Poutine && !config.NoEmit && len(lockFilesForZizmor) > 0 {
		workflowDir := filepath.Dir(lockFilesForZizmor[0])
		if err := runBatchPoutine(workflowDir, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Display schedule warnings
	displayScheduleWarnings(compiler, config.JSONOutput)

	// Post-processing
	if err := runPostProcessing(compiler, workflowDataList, config, compiledCount, campaignFiles); err != nil {
		return workflowDataList, err
	}

	// Output results
	if err := outputResults(stats, validationResults, config); err != nil {
		return workflowDataList, err
	}

	// Return error if any compilations failed
	if errorCount > 0 {
		if len(errorMessages) > 0 {
			return workflowDataList, errors.New(errorMessages[0])
		}
		return workflowDataList, fmt.Errorf("compilation failed")
	}

	return workflowDataList, nil
}

// compileAllFilesInDirectory compiles all workflow files in a directory
func compileAllFilesInDirectory(
	compiler *workflow.Compiler,
	config CompileConfig,
	workflowDir string,
	stats *CompilationStats,
	validationResults *[]ValidationResult,
) ([]*workflow.WorkflowData, error) {
	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Scanning for markdown files in %s", workflowsDir)))
	}

	// Find all markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		return nil, fmt.Errorf("no markdown files found in %s", workflowsDir)
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

	// Compile each file
	var workflowDataList []*workflow.WorkflowData
	var successCount int
	var errorCount int
	var lockFilesForActionlint []string
	var lockFilesForZizmor []string

	for _, file := range mdFiles {
		stats.Total++

		// Handle campaign spec files
		if strings.HasSuffix(file, ".campaign.md") {
			campaignResult, success := processCampaignSpec(ProcessCampaignSpecOptions{
				Compiler:     compiler,
				ResolvedFile: file,
				Verbose:      config.Verbose,
				JSONOutput:   config.JSONOutput,
				NoEmit:       config.NoEmit,
				Zizmor:       false,
				Poutine:      false,
				Actionlint:   false,
				Strict:       config.Strict,
				Validate:     shouldValidate,
			})
			if !success {
				errorCount++
				stats.Errors++
				trackWorkflowFailure(stats, file, len(campaignResult.Errors))
			}
			*validationResults = append(*validationResults, campaignResult)
			continue
		}

		// Compile regular workflow file (disable per-file security tools)
		fileResult := compileWorkflowFile(
			compiler, file, config.Verbose, config.JSONOutput,
			config.NoEmit, false, false, false, // Disable per-file security tools
			config.Strict, shouldValidate,
		)

		if !fileResult.success {
			errorCount++
			stats.Errors++
			trackWorkflowFailure(stats, file, 1)
		} else {
			successCount++
			workflowDataList = append(workflowDataList, fileResult.workflowData)

			// Collect lock files for batch security tools
			if !config.NoEmit && fileResult.lockFile != "" {
				if _, err := os.Stat(fileResult.lockFile); err == nil {
					if config.Actionlint {
						lockFilesForActionlint = append(lockFilesForActionlint, fileResult.lockFile)
					}
					if config.Zizmor {
						lockFilesForZizmor = append(lockFilesForZizmor, fileResult.lockFile)
					}
				}
			}
		}

		*validationResults = append(*validationResults, fileResult.validationResult)
	}

	// Run batch actionlint
	if config.Actionlint && !config.NoEmit && len(lockFilesForActionlint) > 0 {
		if err := runBatchActionlint(lockFilesForActionlint, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch zizmor
	if config.Zizmor && !config.NoEmit && len(lockFilesForZizmor) > 0 {
		if err := runBatchZizmor(lockFilesForZizmor, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Run batch poutine once on the workflow directory
	if config.Poutine && !config.NoEmit && len(lockFilesForZizmor) > 0 {
		if err := runBatchPoutine(workflowsDir, config.Verbose && !config.JSONOutput, config.Strict); err != nil {
			if config.Strict {
				return workflowDataList, err
			}
		}
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Display schedule warnings
	displayScheduleWarnings(compiler, config.JSONOutput)

	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d out of %d workflow files", successCount, len(mdFiles))))
	}

	// Handle purge logic if requested
	if config.Purge && purgeData != nil {
		runPurgeOperations(workflowsDir, purgeData, config.Verbose)
	}

	// Post-processing
	if err := runPostProcessingForDirectory(compiler, workflowDataList, config, workflowsDir, gitRoot, successCount); err != nil {
		return workflowDataList, err
	}

	// Output results
	if err := outputResults(stats, validationResults, config); err != nil {
		return workflowDataList, err
	}

	// Return error if any compilations failed
	if errorCount > 0 {
		return workflowDataList, fmt.Errorf("compilation failed")
	}

	return workflowDataList, nil
}

// purgeTrackingData holds data needed for purge operations
type purgeTrackingData struct {
	existingLockFiles                     []string
	existingInvalidFiles                  []string
	existingCampaignOrchestratorFiles     []string
	existingCampaignOrchestratorLockFiles []string
	expectedLockFiles                     []string
	expectedCampaignDefinitions           []string
}

// collectPurgeData collects existing files for purge operations
func collectPurgeData(workflowsDir string, mdFiles []string, verbose bool) *purgeTrackingData {
	data := &purgeTrackingData{}

	// Find all existing files
	data.existingLockFiles, _ = filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	data.existingInvalidFiles, _ = filepath.Glob(filepath.Join(workflowsDir, "*.invalid.yml"))
	data.existingCampaignOrchestratorFiles, _ = filepath.Glob(filepath.Join(workflowsDir, "*.campaign.g.md"))
	data.existingCampaignOrchestratorLockFiles, _ = filepath.Glob(filepath.Join(workflowsDir, "*.campaign.lock.yml"))

	// Create expected files list
	for _, mdFile := range mdFiles {
		lockFile := stringutil.MarkdownToLockFile(mdFile)
		data.expectedLockFiles = append(data.expectedLockFiles, lockFile)

		if strings.HasSuffix(mdFile, ".campaign.md") {
			data.expectedCampaignDefinitions = append(data.expectedCampaignDefinitions, mdFile)
		}
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
	_ = purgeOrphanedCampaignOrchestrators(workflowsDir, data.expectedCampaignDefinitions, verbose)
	_ = purgeOrphanedCampaignOrchestratorLockFiles(workflowsDir, data.expectedCampaignDefinitions, verbose)
}

// displayScheduleWarnings displays any schedule warnings from the compiler
func displayScheduleWarnings(compiler *workflow.Compiler, jsonOutput bool) {
	scheduleWarnings := compiler.GetScheduleWarnings()
	if len(scheduleWarnings) > 0 && !jsonOutput {
		for _, warning := range scheduleWarnings {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warning))
		}
	}
}

// runPostProcessing runs post-processing for specific files compilation
func runPostProcessing(
	compiler *workflow.Compiler,
	workflowDataList []*workflow.WorkflowData,
	config CompileConfig,
	successCount int,
	campaignFiles []string,
) error {
	// Get action cache
	actionCache := compiler.GetSharedActionCache()

	// Update .gitattributes (errors are non-fatal)
	_ = updateGitAttributes(successCount, actionCache, config.Verbose)

	// Generate Dependabot manifests if requested
	if config.Dependabot && !config.NoEmit {
		gitRoot, err := findGitRoot()
		if err == nil {
			absWorkflowDir := filepath.Join(gitRoot, config.WorkflowDir)
			if err := generateDependabotManifestsWrapper(compiler, workflowDataList, absWorkflowDir, config.ForceOverwrite, config.Strict); err != nil {
				if config.Strict {
					return err
				}
			}
		}
	}

	// Generate maintenance workflow if needed
	// When compiling specific files, we need to parse ALL workflows in the directory
	// to check for expires fields, not just the ones being compiled
	// Skip maintenance workflow generation when using custom --dir option
	if !config.NoEmit && config.WorkflowDir == "" {
		gitRoot, err := findGitRoot()
		if err == nil {
			// Use default workflow dir
			workflowDir := ".github/workflows"
			absWorkflowDir := filepath.Join(gitRoot, workflowDir)

			// Parse all workflow markdown files to check for expires fields
			allWorkflowData, parseErr := parseAllWorkflowsInDirectory(compiler, absWorkflowDir)
			if parseErr == nil {
				if err := generateMaintenanceWorkflowWrapper(compiler, allWorkflowData, absWorkflowDir, config.Verbose, config.Strict); err != nil {
					if config.Strict {
						return err
					}
				}
			} else if config.Verbose {
				// Log parse error but don't fail compilation
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse all workflows for maintenance generation: %v", parseErr)))
			}
		}
	}

	// Validate campaigns only if we're compiling campaign files
	// When compiling specific non-campaign workflows, skip campaign validation
	// When compiling specific campaign files, validate only those campaign files
	if len(campaignFiles) > 0 {
		if err := validateCampaignsWrapper(config.WorkflowDir, config.Verbose, config.Strict, campaignFiles); err != nil {
			if config.Strict {
				return err
			}
		}
	}

	// Save action cache (errors are logged but non-fatal)
	_ = saveActionCache(actionCache, config.Verbose)

	return nil
}

// runPostProcessingForDirectory runs post-processing for directory compilation
func runPostProcessingForDirectory(
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

	// Generate maintenance workflow if needed
	// Skip maintenance workflow generation when using custom --dir option
	if !config.NoEmit && config.WorkflowDir == "" {
		absWorkflowDir := getAbsoluteWorkflowDir(workflowsDir, gitRoot)
		if err := generateMaintenanceWorkflowWrapper(compiler, workflowDataList, absWorkflowDir, config.Verbose, config.Strict); err != nil {
			if config.Strict {
				return err
			}
		}
	}

	// Validate campaigns
	if err := validateCampaignsWrapper(config.WorkflowDir, config.Verbose, config.Strict, nil); err != nil {
		if config.Strict {
			return err
		}
	}

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
		formatStatsTable(statsList)
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
		formatCompilationSummary(stats)
	}

	// Display actionlint summary if enabled
	if config.Actionlint && !config.NoEmit && !config.JSONOutput {
		formatActionlintOutput()
	}

	return nil
}

// parseAllWorkflowsInDirectory parses all workflow markdown files in a directory
// This is used to check for expires fields across all workflows when generating
// the maintenance workflow, even when only specific files are being compiled
func parseAllWorkflowsInDirectory(compiler *workflow.Compiler, workflowsDir string) ([]*workflow.WorkflowData, error) {
	// Find all markdown files in the directory
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}

	var workflowDataList []*workflow.WorkflowData
	for _, file := range mdFiles {
		// Skip campaign specs and generated files
		if strings.HasSuffix(file, ".campaign.md") || strings.HasSuffix(file, ".campaign.g.md") {
			continue
		}

		// Parse the workflow file
		workflowData, err := compiler.ParseWorkflowFile(file)
		if err != nil {
			// Ignore parse errors - workflows might be incomplete
			continue
		}

		workflowDataList = append(workflowDataList, workflowData)
	}

	return workflowDataList, nil
}
