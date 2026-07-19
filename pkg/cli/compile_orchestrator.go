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

	if err := compileWorkflowsCheckContext(ctx); err != nil {
		return nil, err
	}
	compileWorkflowsSetDefaultGHHost()

	if err := compileWorkflowsValidateConfig(config); err != nil {
		return nil, err
	}

	compileWorkflowsInitActionlint(config)

	// Track compilation statistics
	stats := &CompilationStats{}

	// Track validation results for JSON output
	var validationResults []ValidationResult

	// Set up workflow directory (using default if not specified)
	workflowDir := compileWorkflowsWorkflowDir(config.WorkflowDir)

	// Preprocess args: expand directory paths and GitHub URLs to constituent workflow files
	var err error
	config.MarkdownFiles, err = compileWorkflowsResolveArgs(config.MarkdownFiles, config.Verbose)
	if err != nil {
		return nil, err
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
		return compileWorkflowsWatch(ctx, config, compiler)
	}

	// Compile specific files or all files in directory
	if len(config.MarkdownFiles) > 0 {
		// Compile specific workflow files
		return compileSpecificFiles(ctx, compiler, config, stats, &validationResults)
	}

	// Compile all workflow files in directory
	return compileAllFilesInDirectory(ctx, compiler, config, workflowDir, stats, &validationResults)
}

func compileWorkflowsCheckContext(ctx context.Context) error {
	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
		return nil
	}
}

func compileWorkflowsSetDefaultGHHost() {
	if os.Getenv("GH_HOST") != "" { //nolint:osgetenvlibrary
		return
	}
	if detectedHost := getHostFromOriginRemote(); detectedHost != "github.com" && detectedHost != "" {
		compileOrchestratorLog.Printf("Auto-detected GHES host from git remote: %s", detectedHost)
		workflow.SetDefaultGHHost(detectedHost)
	} else if detectedHost == "github.com" {
		workflow.SetDefaultGHHost("")
	}
}

func compileWorkflowsValidateConfig(config CompileConfig) error {
	// Validate configuration
	if err := validateCompileConfig(config); err != nil {
		return err
	}

	// Validate action mode if specified
	return validateActionModeConfig(config.ActionMode)
}

func compileWorkflowsInitActionlint(config CompileConfig) {
	// Initialize actionlint statistics if actionlint is enabled
	if config.Actionlint && !config.NoEmit {
		initActionlintStats()
	}
}

func compileWorkflowsWorkflowDir(workflowDir string) string {
	if workflowDir == "" {
		workflowDir = constants.GetWorkflowDir()
		compileOrchestratorLog.Printf("Using default workflow directory: %s", workflowDir)
		return workflowDir
	}
	workflowDir = filepath.Clean(workflowDir)
	compileOrchestratorLog.Printf("Using custom workflow directory: %s", workflowDir)
	return workflowDir
}

func compileWorkflowsResolveArgs(markdownFiles []string, verbose bool) ([]string, error) {
	// Preprocess args: expand directory paths and GitHub URLs to constituent workflow files
	if len(markdownFiles) == 0 {
		return markdownFiles, nil
	}
	return resolveCompileArgs(markdownFiles, verbose)
}

func compileWorkflowsWatch(ctx context.Context, config CompileConfig, compiler *workflow.Compiler) ([]*workflow.WorkflowData, error) {
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
