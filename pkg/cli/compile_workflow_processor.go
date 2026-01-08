// Package cli provides workflow file processing functions for compilation.
//
// This file contains functions that process individual workflow files and
// campaign specs, handling both regular workflows and campaign orchestrators.
//
// # Organization Rationale
//
// These workflow processing functions are grouped here because they:
//   - Handle per-file processing logic
//   - Process both regular workflows and campaign specs
//   - Have a clear domain focus (workflow file processing)
//   - Keep the main orchestrator focused on batch operations
//
// # Key Functions
//
// Workflow Processing:
//   - processWorkflowFile() - Process a single workflow markdown file
//   - processCampaignSpec() - Process a campaign spec file
//   - collectLockFilesForLinting() - Collect lock files for batch linting
//
// These functions abstract per-file processing, allowing the main compile
// orchestrator to focus on coordination while these handle file processing.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/stringutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var compileWorkflowProcessorLog = logger.New("cli:compile_workflow_processor")

// compileWorkflowFileResult represents the result of compiling a single workflow file
type compileWorkflowFileResult struct {
	workflowData     *workflow.WorkflowData
	lockFile         string
	validationResult ValidationResult
	success          bool
}

// compileWorkflowFile compiles a single workflow file (not a campaign spec)
// Returns the workflow data, lock file path, validation result, and success status
func compileWorkflowFile(
	compiler *workflow.Compiler,
	resolvedFile string,
	verbose bool,
	jsonOutput bool,
	noEmit bool,
	zizmor bool,
	poutine bool,
	actionlint bool,
	strict bool,
	validate bool,
) compileWorkflowFileResult {
	compileWorkflowProcessorLog.Printf("Processing workflow file: %s", resolvedFile)

	result := compileWorkflowFileResult{
		validationResult: ValidationResult{
			Workflow: filepath.Base(resolvedFile),
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
		},
		success: false,
	}

	// Generate lock file name, handling campaign orchestrators specially
	// Campaign orchestrators are named *.campaign.g.md (debug artifacts)
	// but should produce *.campaign.lock.yml (not *.campaign.g.lock.yml)
	var lockFile string
	if strings.HasSuffix(resolvedFile, ".campaign.g.md") {
		// For campaign orchestrators: example.campaign.g.md -> example.campaign.lock.yml
		lockFile = stringutil.CampaignOrchestratorToLockFile(resolvedFile)
	} else {
		lockFile = stringutil.MarkdownToLockFile(resolvedFile)
	}
	result.lockFile = lockFile
	if !noEmit {
		result.validationResult.CompiledFile = lockFile
	}

	// Parse workflow file to get data
	compileWorkflowProcessorLog.Printf("Parsing workflow file: %s", resolvedFile)

	// Set workflow identifier for schedule scattering (use repository-relative path for stability)
	relPath, err := getRepositoryRelativePath(resolvedFile)
	if err != nil {
		compileWorkflowProcessorLog.Printf("Warning: failed to get repository-relative path for %s: %v", resolvedFile, err)
		// Fallback to basename if we can't get relative path
		relPath = filepath.Base(resolvedFile)
	}
	compiler.SetWorkflowIdentifier(relPath)

	// Set repository slug for this specific file (may differ from CWD's repo)
	fileRepoSlug := getRepositorySlugFromRemoteForPath(resolvedFile)
	if fileRepoSlug != "" {
		compiler.SetRepositorySlug(fileRepoSlug)
		compileWorkflowProcessorLog.Printf("Repository slug for file set: %s", fileRepoSlug)
	}

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(resolvedFile)
	if err != nil {
		// Check if this is a shared workflow (not an error, just info)
		if sharedErr, ok := err.(*workflow.SharedWorkflowError); ok {
			if !jsonOutput {
				// Print info message instead of error
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(sharedErr.Error()))
			}
			// Mark as valid but skipped
			result.validationResult.Valid = true
			result.validationResult.Warnings = append(result.validationResult.Warnings, ValidationError{
				Type:    "shared_workflow",
				Message: "Skipped: Shared workflow component (missing 'on' field)",
			})
			result.success = true // Consider it successful, just skipped
			return result
		}

		errMsg := fmt.Sprintf("failed to parse workflow file %s: %v", resolvedFile, err)
		if !jsonOutput {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		}
		result.validationResult.Valid = false
		result.validationResult.Errors = append(result.validationResult.Errors, ValidationError{
			Type:    "parse_error",
			Message: err.Error(),
		})
		return result
	}
	result.workflowData = workflowData

	compileWorkflowProcessorLog.Printf("Starting compilation of %s", resolvedFile)

	// Compile the workflow
	// Disable per-file actionlint run (false instead of actionlint && !noEmit) - we'll batch them
	if err := CompileWorkflowDataWithValidation(compiler, workflowData, resolvedFile, verbose && !jsonOutput, zizmor && !noEmit, poutine && !noEmit, false, strict, validate && !noEmit); err != nil {
		// Always put error on a new line and don't wrap with "failed to compile workflow"
		if !jsonOutput {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		}
		result.validationResult.Valid = false
		result.validationResult.Errors = append(result.validationResult.Errors, ValidationError{
			Type:    "compilation_error",
			Message: err.Error(),
		})
		return result
	}

	result.success = true
	compileWorkflowProcessorLog.Printf("Successfully processed workflow file: %s", resolvedFile)
	return result
}

// ProcessCampaignSpecOptions holds the options for processCampaignSpec
type ProcessCampaignSpecOptions struct {
	Compiler     *workflow.Compiler
	ResolvedFile string
	Verbose      bool
	JSONOutput   bool
	NoEmit       bool
	Zizmor       bool
	Poutine      bool
	Actionlint   bool
	Strict       bool
	Validate     bool
}

// processCampaignSpec processes a campaign spec file
// Returns the validation result and success status
func processCampaignSpec(opts ProcessCampaignSpecOptions) (ValidationResult, bool) {
	compileWorkflowProcessorLog.Printf("Processing campaign spec file: %s", opts.ResolvedFile)

	result := ValidationResult{
		Workflow: filepath.Base(opts.ResolvedFile),
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Validate the campaign spec file and referenced workflows
	spec, problems, vErr := campaign.ValidateSpecFromFile(opts.ResolvedFile)
	if vErr != nil {
		errMsg := fmt.Sprintf("failed to validate campaign spec %s: %v", opts.ResolvedFile, vErr)
		if !opts.JSONOutput {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		}
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    "campaign_validation_error",
			Message: vErr.Error(),
		})
		return result, false
	}

	// Also ensure that workflows referenced by the campaign spec exist
	workflowsDir := filepath.Dir(opts.ResolvedFile)
	workflowProblems := campaign.ValidateWorkflowsExist(spec, workflowsDir)
	problems = append(problems, workflowProblems...)

	if len(problems) > 0 {
		for _, p := range problems {
			if !opts.JSONOutput {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(p))
			}
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    "campaign_validation_error",
				Message: p,
			})
		}
		return result, false
	}

	if opts.Verbose && !opts.JSONOutput {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Validated campaign spec %s", filepath.Base(opts.ResolvedFile))))
	}

	// Generate and compile the campaign orchestrator
	if _, genErr := generateAndCompileCampaignOrchestrator(GenerateCampaignOrchestratorOptions{
		Compiler:             opts.Compiler,
		Spec:                 spec,
		CampaignSpecPath:     opts.ResolvedFile,
		Verbose:              opts.Verbose && !opts.JSONOutput,
		NoEmit:               opts.NoEmit,
		RunZizmorPerFile:     opts.Zizmor && !opts.NoEmit,
		RunPoutinePerFile:    opts.Poutine && !opts.NoEmit,
		RunActionlintPerFile: opts.Actionlint && !opts.NoEmit,
		Strict:               opts.Strict,
		ValidateActionSHAs:   opts.Validate && !opts.NoEmit,
	}); genErr != nil {
		errMsg := fmt.Sprintf("failed to compile campaign orchestrator for %s: %v", filepath.Base(opts.ResolvedFile), genErr)
		if !opts.JSONOutput {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		}
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{Type: "campaign_orchestrator_error", Message: errMsg})
		return result, false
	}

	compileWorkflowProcessorLog.Printf("Successfully processed campaign spec: %s", opts.ResolvedFile)
	return result, true
}
