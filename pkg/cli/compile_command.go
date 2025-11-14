package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var compileLog = logger.New("cli:compile_command")

// CompileWorkflowWithValidation compiles a workflow with always-on YAML validation for CLI usage
func CompileWorkflowWithValidation(compiler *workflow.Compiler, filePath string, verbose bool, runZizmorPerFile bool, runPoutinePerFile bool, runActionlintPerFile bool, strict bool, validateActionSHAs bool) error {
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

	// Validate action SHAs if requested
	if validateActionSHAs {
		compileLog.Print("Validating action SHAs in lock file")
		// Use the compiler's shared action cache to benefit from cached resolutions
		actionCache := compiler.GetSharedActionCache()
		if err := workflow.ValidateActionSHAsInLockFile(lockFile, actionCache, verbose); err != nil {
			// Action SHA validation warnings are non-fatal
			compileLog.Printf("Action SHA validation completed with warnings: %v", err)
		}
	}

	// Run zizmor on the generated lock file if requested
	if runZizmorPerFile {
		if err := runZizmorOnFile(lockFile, verbose, strict); err != nil {
			return fmt.Errorf("zizmor security scan failed: %w", err)
		}
	}

	// Run poutine on the generated lock file if requested
	if runPoutinePerFile {
		if err := runPoutineOnFile(lockFile, verbose, strict); err != nil {
			return fmt.Errorf("poutine security scan failed: %w", err)
		}
	}

	// Run actionlint on the generated lock file if requested
	if runActionlintPerFile {
		if err := runActionlintOnFile(lockFile, verbose, strict); err != nil {
			return fmt.Errorf("actionlint linter failed: %w", err)
		}
	}

	return nil
}

// CompileWorkflowDataWithValidation compiles from already-parsed WorkflowData with validation
// This avoids re-parsing when the workflow data has already been parsed
func CompileWorkflowDataWithValidation(compiler *workflow.Compiler, workflowData *workflow.WorkflowData, filePath string, verbose bool, runZizmorPerFile bool, runPoutinePerFile bool, runActionlintPerFile bool, strict bool, validateActionSHAs bool) error {
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

	// Validate action SHAs if requested
	if validateActionSHAs {
		compileLog.Print("Validating action SHAs in lock file")
		// Use the compiler's shared action cache to benefit from cached resolutions
		actionCache := compiler.GetSharedActionCache()
		if err := workflow.ValidateActionSHAsInLockFile(lockFile, actionCache, verbose); err != nil {
			// Action SHA validation warnings are non-fatal
			compileLog.Printf("Action SHA validation completed with warnings: %v", err)
		}
	}

	// Run zizmor on the generated lock file if requested
	if runZizmorPerFile {
		if err := runZizmorOnFile(lockFile, verbose, strict); err != nil {
			return fmt.Errorf("zizmor security scan failed: %w", err)
		}
	}

	// Run poutine on the generated lock file if requested
	if runPoutinePerFile {
		if err := runPoutineOnFile(lockFile, verbose, strict); err != nil {
			return fmt.Errorf("poutine security scan failed: %w", err)
		}
	}

	// Run actionlint on the generated lock file if requested
	if runActionlintPerFile {
		if err := runActionlintOnFile(lockFile, verbose, strict); err != nil {
			return fmt.Errorf("actionlint linter failed: %w", err)
		}
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
	Poutine              bool     // Run poutine security scanner on generated .lock.yml files
	Actionlint           bool     // Run actionlint linter on generated .lock.yml files
	JSONOutput           bool     // Output validation results as JSON
}

// CompilationStats tracks the results of workflow compilation
type CompilationStats struct {
	Total           int
	Errors          int
	Warnings        int
	FailedWorkflows []string // Names of workflows that failed compilation
}

// ValidationError represents a single validation error or warning
type ValidationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationResult represents the validation result for a single workflow
type ValidationResult struct {
	Workflow     string            `json:"workflow"`
	Valid        bool              `json:"valid"`
	Errors       []ValidationError `json:"errors"`
	Warnings     []ValidationError `json:"warnings"`
	CompiledFile string            `json:"compiled_file,omitempty"`
}

// validateCompileConfig validates the configuration flags before compilation
// This is extracted for faster testing without full compilation
func validateCompileConfig(config CompileConfig) error {
	// Validate dependabot flag usage
	if config.Dependabot {
		if len(config.MarkdownFiles) > 0 {
			return fmt.Errorf("--dependabot flag cannot be used with specific workflow files")
		}
		if config.WorkflowDir != "" && config.WorkflowDir != ".github/workflows" {
			return fmt.Errorf("--dependabot flag cannot be used with custom --dir")
		}
	}

	// Validate purge flag usage
	if config.Purge && len(config.MarkdownFiles) > 0 {
		return fmt.Errorf("--purge flag can only be used when compiling all markdown files (no specific files specified)")
	}

	// Validate workflow directory path
	if config.WorkflowDir != "" && filepath.IsAbs(config.WorkflowDir) {
		return fmt.Errorf("--dir must be a relative path, got: %s", config.WorkflowDir)
	}

	return nil
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
	poutine := config.Poutine
	actionlint := config.Actionlint
	jsonOutput := config.JSONOutput

	compileLog.Printf("Starting workflow compilation: files=%d, validate=%v, watch=%v, noEmit=%v, dependabot=%v, zizmor=%v, poutine=%v, actionlint=%v, jsonOutput=%v", len(markdownFiles), validate, watch, noEmit, dependabot, zizmor, poutine, actionlint, jsonOutput)

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
		compileLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		// Clean the path to avoid issues with ".." or other problematic elements
		workflowDir = filepath.Clean(workflowDir)
		compileLog.Printf("Using custom workflow directory: %s", workflowDir)
	}

	// Create compiler with verbose flag and AI engine override
	compiler := workflow.NewCompiler(verbose, engineOverride, GetVersion())
	compileLog.Print("Created compiler instance")

	// Set validation based on the validate flag (false by default for compatibility)
	compiler.SetSkipValidation(!validate)
	compileLog.Printf("Validation enabled: %v", validate)

	// Set noEmit flag to validate without generating lock files
	compiler.SetNoEmit(noEmit)
	if noEmit {
		compileLog.Print("No-emit mode enabled: validating without generating lock files")
	}

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
		compileLog.Printf("Compiling %d specific workflow files", len(markdownFiles))
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
			compileLog.Printf("Resolving workflow file: %s", markdownFile)
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
			compileLog.Printf("Resolved to: %s", resolvedFile)

			// Update result with resolved file name
			result.Workflow = filepath.Base(resolvedFile)
			lockFile := strings.TrimSuffix(resolvedFile, ".md") + ".lock.yml"
			if !noEmit {
				result.CompiledFile = lockFile
			}

			// Parse workflow file to get data
			compileLog.Printf("Parsing workflow file: %s", resolvedFile)
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

			compileLog.Printf("Starting compilation of %s", resolvedFile)
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
		actionCache := compiler.GetSharedActionCache()
		if actionCache != nil {
			if err := actionCache.Save(); err != nil {
				compileLog.Printf("Failed to save action cache: %v", err)
				if verbose {
					fmt.Fprintf(os.Stderr, "⚠️  Failed to save action cache: %v\n", err)
				}
			} else {
				compileLog.Print("Action cache saved successfully")
				if verbose {
					fmt.Fprintf(os.Stderr, "✓ Action cache saved to %s\n", actionCache.GetCachePath())
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

		// Initialize validation result for this workflow
		result := ValidationResult{
			Workflow: filepath.Base(file),
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
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
	actionCache := compiler.GetSharedActionCache()
	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to save action cache: %v\n", err)
			}
		} else {
			compileLog.Print("Action cache saved successfully")
			if verbose {
				fmt.Fprintf(os.Stderr, "✓ Action cache saved to %s\n", actionCache.GetCachePath())
			}
		}
	}

	// Return error if any compilations failed
	if errorCount > 0 {
		return workflowDataList, fmt.Errorf("compilation failed")
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
		if err := CompileWorkflowWithValidation(compiler, markdownFile, verbose, false, false, false, false, false); err != nil {
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

// compileSingleFile compiles a single markdown workflow file and updates compilation statistics
// If checkExists is true, the function will check if the file exists before compiling
// Returns true if compilation was attempted (file exists or checkExists is false), false otherwise
func compileSingleFile(compiler *workflow.Compiler, file string, stats *CompilationStats, verbose bool, checkExists bool) bool {
	// Check if file exists if requested (for watch mode)
	if checkExists {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			compileLog.Printf("File %s was deleted, skipping compilation", file)
			return false
		}
	}

	stats.Total++

	compileLog.Printf("Compiling: %s", file)
	if verbose {
		fmt.Fprintf(os.Stderr, "🔨 Compiling: %s\n", file)
	}

	if err := CompileWorkflowWithValidation(compiler, file, verbose, false, false, false, false, false); err != nil {
		// Always show compilation errors on new line
		fmt.Fprintln(os.Stderr, err.Error())
		stats.Errors++
		stats.FailedWorkflows = append(stats.FailedWorkflows, filepath.Base(file))
	} else {
		compileLog.Printf("Successfully compiled: %s", file)
	}

	return true
}

// compileAllWorkflowFiles compiles all markdown files in the workflows directory
func compileAllWorkflowFiles(compiler *workflow.Compiler, workflowsDir string, verbose bool) (*CompilationStats, error) {
	compileLog.Printf("Compiling all workflow files in directory: %s", workflowsDir)
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
		compileLog.Printf("No markdown files found in %s", workflowsDir)
		if verbose {
			fmt.Printf("No markdown files found in %s\n", workflowsDir)
		}
		return stats, nil
	}

	compileLog.Printf("Found %d markdown files to compile", len(mdFiles))

	// Compile each file
	for _, file := range mdFiles {
		compileSingleFile(compiler, file, stats, verbose, false)
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Save the action cache after all compilations
	actionCache := compiler.GetSharedActionCache()
	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to save action cache: %v\n", err)
			}
		} else {
			compileLog.Print("Action cache saved successfully")
			if verbose {
				fmt.Fprintf(os.Stderr, "✓ Action cache saved to %s\n", actionCache.GetCachePath())
			}
		}
	}

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
		compileSingleFile(compiler, file, stats, verbose, true)
	}

	// Get warning count from compiler
	stats.Warnings = compiler.GetWarningCount()

	// Save the action cache after compilations
	actionCache := compiler.GetSharedActionCache()
	if actionCache != nil {
		if err := actionCache.Save(); err != nil {
			compileLog.Printf("Failed to save action cache: %v", err)
			if verbose {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to save action cache: %v\n", err)
			}
		} else {
			compileLog.Print("Action cache saved successfully")
		}
	}

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

// NewCompileCommand creates the compile command
func NewCompileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compile [markdown-file]...",
		Short: "Compile Markdown to YAML workflows",
		Long: `Compile one or more Markdown workflow files to YAML workflows.

If no files are specified, all Markdown files in .github/workflows will be compiled.

The --dependabot flag generates dependency manifests when dependencies are detected:
  - For npm: Creates package.json and package-lock.json (requires npm in PATH)
  - For Python: Creates requirements.txt for pip packages
  - For Go: Creates go.mod for go install/get packages
  - Creates .github/dependabot.yml with all detected ecosystems
  - Use --force to overwrite existing dependabot.yml
  - Cannot be used with specific workflow files or custom --dir
  - Only processes workflows in the default .github/workflows directory

Examples:
  ` + constants.CLIExtensionPrefix + ` compile                    # Compile all Markdown files
  ` + constants.CLIExtensionPrefix + ` compile ci-doctor    # Compile a specific workflow
  ` + constants.CLIExtensionPrefix + ` compile ci-doctor daily-plan  # Compile multiple workflows
  ` + constants.CLIExtensionPrefix + ` compile workflow.md        # Compile by file path
  ` + constants.CLIExtensionPrefix + ` compile --dir custom/workflows  # Compile from custom directory
  ` + constants.CLIExtensionPrefix + ` compile --watch ci-doctor     # Watch and auto-compile
  ` + constants.CLIExtensionPrefix + ` compile --trial --logical-repo owner/repo  # Compile for trial mode
  ` + constants.CLIExtensionPrefix + ` compile --dependabot        # Generate Dependabot manifests
  ` + constants.CLIExtensionPrefix + ` compile --dependabot --force  # Force overwrite existing dependabot.yml`,
		Run: func(cmd *cobra.Command, args []string) {
			engineOverride, _ := cmd.Flags().GetString("engine")
			validate, _ := cmd.Flags().GetBool("validate")
			watch, _ := cmd.Flags().GetBool("watch")
			dir, _ := cmd.Flags().GetString("dir")
			workflowsDir, _ := cmd.Flags().GetString("workflows-dir")
			noEmit, _ := cmd.Flags().GetBool("no-emit")
			purge, _ := cmd.Flags().GetBool("purge")
			strict, _ := cmd.Flags().GetBool("strict")
			trial, _ := cmd.Flags().GetBool("trial")
			logicalRepo, _ := cmd.Flags().GetString("logical-repo")
			dependabot, _ := cmd.Flags().GetBool("dependabot")
			forceOverwrite, _ := cmd.Flags().GetBool("force")
			zizmor, _ := cmd.Flags().GetBool("zizmor")
			poutine, _ := cmd.Flags().GetBool("poutine")
			actionlint, _ := cmd.Flags().GetBool("actionlint")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")
			if err := ValidateEngine(engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}

			// Handle --workflows-dir deprecation
			workflowDir := dir
			if workflowsDir != "" {
				if dir != "" {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage("cannot use both --dir and --workflows-dir flags"))
					os.Exit(1)
				}
				workflowDir = workflowsDir
			}
			config := CompileConfig{
				MarkdownFiles:        args,
				Verbose:              verbose,
				EngineOverride:       engineOverride,
				Validate:             validate,
				Watch:                watch,
				WorkflowDir:          workflowDir,
				SkipInstructions:     false, // Deprecated field, kept for backward compatibility
				NoEmit:               noEmit,
				Purge:                purge,
				TrialMode:            trial,
				TrialLogicalRepoSlug: logicalRepo,
				Strict:               strict,
				Dependabot:           dependabot,
				ForceOverwrite:       forceOverwrite,
				Zizmor:               zizmor,
				Poutine:              poutine,
				Actionlint:           actionlint,
				JSONOutput:           jsonOutput,
			}
			if _, err := CompileWorkflows(config); err != nil {
				errMsg := err.Error()
				// Check if error is already formatted (contains suggestions or starts with ✗)
				if strings.Contains(errMsg, "Suggestions:") || strings.HasPrefix(errMsg, "✗") {
					fmt.Fprintln(os.Stderr, errMsg)
				} else {
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
				}
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP("engine", "e", "", "Override AI engine (claude, codex, copilot, custom)")
	cmd.Flags().Bool("validate", false, "Enable GitHub Actions workflow schema validation, container image validation, and action SHA validation")
	cmd.Flags().BoolP("watch", "w", false, "Watch for changes to workflow files and recompile automatically")
	cmd.Flags().String("dir", "", "Relative directory containing workflows (default: .github/workflows)")
	cmd.Flags().String("workflows-dir", "", "Deprecated: use --dir instead")
	_ = cmd.Flags().MarkDeprecated("workflows-dir", "use --dir instead")
	cmd.Flags().Bool("no-emit", false, "Validate workflow without generating lock files")
	cmd.Flags().Bool("purge", false, "Delete .lock.yml files that were not regenerated during compilation (only when no specific files are specified)")
	cmd.Flags().Bool("strict", false, "Enable strict mode: require timeout, refuse write permissions, require network configuration")
	cmd.Flags().Bool("trial", false, "Enable trial mode compilation (modifies workflows for trial execution)")
	cmd.Flags().String("logical-repo", "", "Repository to simulate workflow execution against (for trial mode)")
	cmd.Flags().Bool("dependabot", false, "Generate dependency manifests (package.json, requirements.txt, go.mod) and Dependabot config when dependencies are detected")
	cmd.Flags().Bool("force", false, "Force overwrite of existing files (e.g., dependabot.yml)")
	cmd.Flags().Bool("zizmor", false, "Run zizmor security scanner on generated .lock.yml files")
	cmd.Flags().Bool("poutine", false, "Run poutine security scanner on generated .lock.yml files")
	cmd.Flags().Bool("actionlint", false, "Run actionlint linter on generated .lock.yml files")
	cmd.Flags().Bool("json", false, "Output validation results as JSON")

	return cmd
}
