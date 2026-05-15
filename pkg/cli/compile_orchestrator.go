package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileOrchestratorLog = logger.New("cli:compile_orchestrator")

// CompileWorkflows compiles workflows based on the provided configuration
func CompileWorkflows(ctx context.Context, config CompileConfig) ([]*workflow.WorkflowData, error) {
	compileOrchestratorLog.Printf("Starting workflow compilation: files=%d, validate=%v, watch=%v, noEmit=%v",
		len(config.MarkdownFiles), config.Validate, config.Watch, config.NoEmit)

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return nil, ctx.Err()
	default:
	}

	// Validate configuration
	if err := validateCompileConfig(config); err != nil {
		return nil, err
	}

	// Validate action mode if specified
	if err := validateActionModeConfig(config.ActionMode); err != nil {
		return nil, err
	}

	// Initialize actionlint statistics if actionlint is enabled
	if config.Actionlint && !config.NoEmit {
		initActionlintStats()
	}

	// Track compilation statistics
	stats := &CompilationStats{}

	// Track validation results for JSON output
	var validationResults []ValidationResult

	// Set up workflow directory (using default if not specified)
	workflowDir := config.WorkflowDir
	if workflowDir == "" {
		workflowDir = constants.GetWorkflowDir()
		compileOrchestratorLog.Printf("Using default workflow directory: %s", workflowDir)
	} else {
		workflowDir = filepath.Clean(workflowDir)
		compileOrchestratorLog.Printf("Using custom workflow directory: %s", workflowDir)
	}

	// Preprocess args: expand directory paths and GitHub URLs to constituent workflow files
	if len(config.MarkdownFiles) > 0 {
		expandedFiles, err := resolveCompileArgs(config.MarkdownFiles, config.Verbose)
		if err != nil {
			return nil, err
		}
		config.MarkdownFiles = expandedFiles
	}

	// Create and configure compiler
	compiler := createAndConfigureCompiler(config)
	compiler.SetContext(ctx)

	if err := validateRepositoryManifestForCompilation(config, stats, &validationResults); err != nil {
		if config.JSONOutput {
			if outputErr := outputResults(stats, &validationResults, config); outputErr != nil {
				return nil, outputErr
			}
		}
		return nil, err
	}

	// Handle watch mode (early return)
	if config.Watch {
		// Watch mode: watch for file changes and recompile automatically
		// For watch mode, we only support a single file for now
		var markdownFile string
		if len(config.MarkdownFiles) > 0 {
			if len(config.MarkdownFiles) > 1 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Watch mode only supports a single file, using the first one"))
			}
			// Resolve the workflow file to get the full path
			resolvedFile, err := resolveWorkflowFile(config.MarkdownFiles[0], config.Verbose)
			if err != nil {
				// Return error directly without wrapping - it already contains formatted message with suggestions
				return nil, err
			}
			markdownFile = resolvedFile
		}
		return nil, watchAndCompileWorkflows(ctx, markdownFile, compiler, config.Verbose)
	}

	// Compile specific files or all files in directory
	if len(config.MarkdownFiles) > 0 {
		// Compile specific workflow files
		return compileSpecificFiles(ctx, compiler, config, stats, &validationResults)
	}

	// Compile all workflow files in directory
	return compileAllFilesInDirectory(ctx, compiler, config, workflowDir, stats, &validationResults)
}
