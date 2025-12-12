package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var compileOrchestratorLog = logger.New("cli:compile_orchestrator")

// CompileWorkflows compiles workflows based on the provided configuration
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
	poutine := config.Poutine
	actionlint := config.Actionlint
	jsonOutput := config.JSONOutput

	compileOrchestratorLog.Printf("Starting workflow compilation: files=%d, validate=%v, watch=%v, noEmit=%v, dependabot=%v, zizmor=%v, poutine=%v, actionlint=%v, jsonOutput=%v", len(markdownFiles), validate, watch, noEmit, dependabot, zizmor, poutine, actionlint, jsonOutput)

	// Track compilation statistics
	stats := &CompilationStats{}

	// Track validation results for JSON output
	var validationResults []ValidationResult

	// Validate configuration
	if err := validateCompileConfig(config); err != nil {
		return nil, err
	}

	// Validate and set default for workflow directory
	if workflowDir == "" {
		workflowDir = ".github/workflows"
		compileOrchestratorLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		// Clean the path to avoid issues with ".." or other problematic elements
		workflowDir = filepath.Clean(workflowDir)
		compileOrchestratorLog.Printf("Using custom workflow directory: %s", workflowDir)
	}

	// Create compiler with verbose flag and AI engine override
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	compileOrchestratorLog.Print("Created compiler instance")

	// Set validation based on the validate flag (false by default for compatibility)
	compiler.SetSkipValidation(!validate)
	compileOrchestratorLog.Printf("Validation enabled: %v", validate)

	// Set noEmit flag to validate without generating lock files
	compiler.SetNoEmit(noEmit)
	if noEmit {
		compileOrchestratorLog.Print("No-emit mode enabled: validating without generating lock files")
	}

	// Set strict mode if specified
	compiler.SetStrictMode(strict)

	// Set trial mode if specified
	if trialMode {
		compileOrchestratorLog.Printf("Enabling trial mode: repoSlug=%s", trialLogicalRepoSlug)
		compiler.SetTrialMode(true)
		if trialLogicalRepoSlug != "" {
			compiler.SetTrialLogicalRepoSlug(trialLogicalRepoSlug)
		}
	}

	// Set refresh stop time flag
	compiler.SetRefreshStopTime(config.RefreshStopTime)
	if config.RefreshStopTime {
		compileOrchestratorLog.Print("Stop time refresh enabled: will regenerate stop-after times")
	}

	// Set action mode if specified
	if config.ActionMode != "" {
		mode := workflow.ActionMode(config.ActionMode)
		if !mode.IsValid() {
			return nil, fmt.Errorf("invalid action mode '%s'. Must be 'inline', 'dev', or 'release'", config.ActionMode)
		}
		compiler.SetActionMode(mode)
		compileOrchestratorLog.Printf("Action mode set to: %s", mode)
	} else {
		// Use auto-detection
		mode := workflow.DetectActionMode()
		compiler.SetActionMode(mode)
		compileOrchestratorLog.Printf("Action mode auto-detected: %s", mode)
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
		compileOrchestratorLog.Printf("Compiling %d specific workflow files", len(markdownFiles))
		// Compile specific workflow files
		var compiledCount int
		var errorCount int
		var errorMessages []string
		for _, markdownFile := range markdownFiles {
			stats.Total++

			// Initialize validation result for this workflow
			result := ValidationResult{
				Workflow: markdownFile,
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			}

			// Resolve workflow ID or file path to actual file path
			compileOrchestratorLog.Printf("Resolving workflow file: %s", markdownFile)
			resolvedFile, err := resolveWorkflowFile(markdownFile, verbose)
			if err != nil {
				if !jsonOutput {
					// Print the error directly - it already contains suggestions and formatting
					fmt.Fprintln(os.Stderr, err.Error())
				}
				errorMessages = append(errorMessages, err.Error())
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, markdownFile)

				// Add to validation results
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:    "resolution_error",
					Message: err.Error(),
				})
				validationResults = append(validationResults, result)
				continue
			}
			compileOrchestratorLog.Printf("Resolved to: %s", resolvedFile)

			// Update result with resolved file name
			result.Workflow = filepath.Base(resolvedFile)

			// Handle campaign spec files separately from regular workflows
			if strings.HasSuffix(resolvedFile, ".campaign.md") {
				// Validate the campaign spec file and referenced workflows instead of
				// compiling it as a regular workflow YAML.
				spec, problems, vErr := campaign.ValidateSpecFromFile(resolvedFile)
				if vErr != nil {
					errMsg := fmt.Sprintf("failed to validate campaign spec %s: %v", resolvedFile, vErr)
					if !jsonOutput {
						fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
					}
					errorMessages = append(errorMessages, vErr.Error())
					errorCount++
					stats.Errors++
					stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(resolvedFile))

					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Type:    "campaign_validation_error",
						Message: vErr.Error(),
					})
					validationResults = append(validationResults, result)
					continue
				}

				// Also ensure that workflows referenced by the campaign spec exist
				workflowsDir := filepath.Dir(resolvedFile)
				workflowProblems := campaign.ValidateWorkflowsExist(spec, workflowsDir)
				problems = append(problems, workflowProblems...)

				if len(problems) > 0 {
					for _, p := range problems {
						if !jsonOutput {
							fmt.Fprintln(os.Stderr, console.FormatErrorMessage(p))
						}
						result.Valid = false
						result.Errors = append(result.Errors, ValidationError{
							Type:    "campaign_validation_error",
							Message: p,
						})
					}
					errorMessages = append(errorMessages, problems[0])
					errorCount++
					stats.Errors++
					stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(resolvedFile))
				} else if verbose && !jsonOutput {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Validated campaign spec %s", filepath.Base(resolvedFile))))
				}

				validationResults = append(validationResults, result)
				continue
			}

			lockFile := strings.TrimSuffix(resolvedFile, ".md") + ".lock.yml"
			if !noEmit {
				result.CompiledFile = lockFile
			}

			// Parse workflow file to get data
			compileOrchestratorLog.Printf("Parsing workflow file: %s", resolvedFile)
			workflowData, err := compiler.ParseWorkflowFile(resolvedFile)
			if err != nil {
				errMsg := fmt.Sprintf("failed to parse workflow file %s: %v", resolvedFile, err)
				if !jsonOutput {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
				}
				errorMessages = append(errorMessages, err.Error())
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(resolvedFile))

				// Add to validation results
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:    "parse_error",
					Message: err.Error(),
				})
				validationResults = append(validationResults, result)
				continue
			}
			workflowDataList = append(workflowDataList, workflowData)

			compileOrchestratorLog.Printf("Starting compilation of %s", resolvedFile)
			if err := CompileWorkflowDataWithValidation(compiler, workflowData, resolvedFile, verbose && !jsonOutput, zizmor && !noEmit, poutine && !noEmit, actionlint && !noEmit, strict, validate && !noEmit); err != nil {
				// Always put error on a new line and don't wrap with "failed to compile workflow"
				if !jsonOutput {
					fmt.Fprintln(os.Stderr, err.Error())
				}
				errorMessages = append(errorMessages, err.Error())
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(resolvedFile))

				// Add to validation results
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:    "compilation_error",
					Message: err.Error(),
				})
				validationResults = append(validationResults, result)
				continue
			}
			compiledCount++

			// Add successful validation result
			validationResults = append(validationResults, result)
		}

		// Get warning count from compiler
		stats.Warnings = compiler.GetWarningCount()

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully compiled %d workflow file(s)", compiledCount)))
		}

		// Get the action cache once for use in multiple places
		actionCache := compiler.GetSharedActionCache()
		hasActionCacheEntries := actionCache != nil && len(actionCache.Entries) > 0

		// Ensure .gitattributes marks .lock.yml files as generated
		// Only update if we successfully compiled workflows or have action cache entries
		if compiledCount > 0 || hasActionCacheEntries {
			compileOrchestratorLog.Printf("Updating .gitattributes (compiled=%d, actionCache=%v)", compiledCount, hasActionCacheEntries)
			if err := ensureGitAttributes(); err != nil {
				compileOrchestratorLog.Printf("Failed to update .gitattributes: %v", err)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
				}
			} else {
				compileOrchestratorLog.Printf("Successfully updated .gitattributes")
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .gitattributes to mark .lock.yml files as generated"))
				}
			}
		} else {
			compileOrchestratorLog.Print("Skipping .gitattributes update (no compiled workflows and no action cache entries)")
		}

		// Generate Dependabot manifests if requested
		if dependabot && !noEmit {
			compileOrchestratorLog.Print("Generating Dependabot manifests for compiled workflows")
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

		// Validate campaign specs if they exist
		if err := validateCampaigns(workflowDir, verbose); err != nil {
			if strict {
				return workflowDataList, fmt.Errorf("campaign validation failed: %w", err)
			}
			// Non-strict mode: just report as warning
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Campaign validation: %v", err)))
		}

		// Output JSON if requested
		if jsonOutput {
			jsonBytes, err := json.MarshalIndent(validationResults, "", "  ")
			if err != nil {
				return workflowDataList, fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			// Print summary for text output
			printCompilationSummary(stats)
		}

		// Save the action cache after all compilations
		if actionCache != nil {
			if err := actionCache.Save(); err != nil {
				compileOrchestratorLog.Printf("Failed to save action cache: %v", err)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save action cache: %v", err)))
				}
			} else {
				compileOrchestratorLog.Print("Action cache saved successfully")
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Action cache saved to %s", actionCache.GetCachePath())))
				}
			}
		}

		// Return error if any compilations failed
		if errorCount > 0 {
			// Return the first error message for backward compatibility with tests
			if len(errorMessages) > 0 {
				return workflowDataList, errors.New(errorMessages[0])
			}
			return workflowDataList, fmt.Errorf("compilation failed")
		}

		return workflowDataList, nil
	}

	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil, fmt.Errorf("compile without arguments requires being in a git repository: %w", err)
	}
	compileOrchestratorLog.Printf("Found git root: %s", gitRoot)

	// Compile all markdown files in the specified workflow directory relative to git root
	workflowsDir := filepath.Join(gitRoot, workflowDir)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}

	compileOrchestratorLog.Printf("Scanning for markdown files in %s", workflowsDir)
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

	compileOrchestratorLog.Printf("Found %d markdown files to compile", len(mdFiles))
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

	// Compile each file (including .campaign.md files)
	var errorCount int
	var successCount int
	for _, file := range mdFiles {
		stats.Total++

		// Initialize validation result for this workflow
		result := ValidationResult{
			Workflow: filepath.Base(file),
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
		}

		// Handle campaign spec files separately from regular workflows
		if strings.HasSuffix(file, ".campaign.md") {
			// Validate the campaign spec file and referenced workflows instead of
			// compiling it as a regular workflow YAML.
			spec, problems, vErr := campaign.ValidateSpecFromFile(file)
			if vErr != nil {
				if !jsonOutput {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("failed to validate campaign spec %s: %v", file, vErr)))
				}
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))

				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:    "campaign_validation_error",
					Message: vErr.Error(),
				})
				validationResults = append(validationResults, result)
				continue
			}

			workflowsDir := filepath.Dir(file)
			workflowProblems := campaign.ValidateWorkflowsExist(spec, workflowsDir)
			problems = append(problems, workflowProblems...)

			if len(problems) > 0 {
				for _, p := range problems {
					if !jsonOutput {
						fmt.Fprintln(os.Stderr, console.FormatErrorMessage(p))
					}
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Type:    "campaign_validation_error",
						Message: p,
					})
				}
				// Treat campaign spec problems as compilation errors for this file
				errorCount++
				stats.Errors++
				stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
			} else if verbose && !jsonOutput {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Validated campaign spec %s", filepath.Base(file))))
			}

			validationResults = append(validationResults, result)
			continue
		}

		lockFile := strings.TrimSuffix(file, ".md") + ".lock.yml"
		if !noEmit {
			result.CompiledFile = lockFile
		}

		// Parse workflow file to get data
		workflowData, err := compiler.ParseWorkflowFile(file)
		if err != nil {
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("failed to parse workflow file %s: %v", file, err)))
			}
			errorCount++
			stats.Errors++
			stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))

			// Add to validation results
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "parse_error",
				Message: err.Error(),
			})
			validationResults = append(validationResults, result)
			continue
		}
		workflowDataList = append(workflowDataList, workflowData)

		if err := CompileWorkflowDataWithValidation(compiler, workflowData, file, verbose && !jsonOutput, zizmor && !noEmit, poutine && !noEmit, actionlint && !noEmit, strict, validate && !noEmit); err != nil {
			// Print the error to stderr (errors from CompileWorkflow are already formatted)
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			errorCount++
			stats.Errors++
			stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))

			// Add to validation results
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "compilation_error",
				Message: err.Error(),
			})
			validationResults = append(validationResults, result)
			continue
		}
		successCount++

		// Add successful validation result
		validationResults = append(validationResults, result)
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

	// Get the action cache once for use in multiple places
	actionCache := compiler.GetSharedActionCache()
	hasActionCacheEntries := actionCache != nil && len(actionCache.Entries) > 0

	// Ensure .gitattributes marks .lock.yml files as generated
	// Only update if we successfully compiled workflows or have action cache entries
	if successCount > 0 || hasActionCacheEntries {
		compileOrchestratorLog.Printf("Updating .gitattributes (compiled=%d, actionCache=%v)", successCount, hasActionCacheEntries)
		if err := ensureGitAttributes(); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .gitattributes to mark .lock.yml files as generated"))
		}
	} else {
		compileOrchestratorLog.Print("Skipping .gitattributes update (no compiled workflows and no action cache entries)")
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

	// Generate maintenance workflow if any workflow uses expires field
	if !noEmit {
		absWorkflowDir := workflowsDir
		if !filepath.IsAbs(absWorkflowDir) {
			absWorkflowDir = filepath.Join(gitRoot, workflowDir)
		}

		if err := workflow.GenerateMaintenanceWorkflow(workflowDataList, absWorkflowDir, verbose); err != nil {
			if strict {
				return workflowDataList, fmt.Errorf("failed to generate maintenance workflow: %w", err)
			}
			// Non-strict mode: just report as warning
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate maintenance workflow: %v", err)))
		}
	}

	// Note: Instructions are only written by the init command
	// The compile command should not write instruction files

	// Validate campaign specs if they exist
	if err := validateCampaigns(workflowDir, verbose); err != nil {
		if strict {
			return workflowDataList, fmt.Errorf("campaign validation failed: %w", err)
		}
		// Non-strict mode: just report as warning
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Campaign validation: %v", err)))
	}

	// Output JSON if requested
	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(validationResults, "", "  ")
		if err != nil {
			return workflowDataList, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// Print summary for text output
		printCompilationSummary(stats)
	}

	// Save the action cache after all compilations
	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileOrchestratorLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save action cache: %v", err)))
			}
		} else {
			compileOrchestratorLog.Print("Action cache saved successfully")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Action cache saved to %s", actionCache.GetCachePath())))
			}
		}
	}

	// Return error if any compilations failed
	if errorCount > 0 {
		return workflowDataList, fmt.Errorf("compilation failed")
	}

	return workflowDataList, nil
}
