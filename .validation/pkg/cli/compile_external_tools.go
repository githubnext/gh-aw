// This file provides external tool runners for workflow compilation.
//
// This file contains functions that invoke external analysis tools
// (actionlint, zizmor, poutine, runner-guard) on compiled workflow files.
//
// # Organization Rationale
//
// These external tool runner functions are grouped here because they:
//   - Invoke third-party analysis tools (not compilation logic)
//   - Operate on compiled lock files as a post-compilation step
//   - Have a clear domain focus (external tooling integration)
//   - Keep compile_batch_operations.go focused on batch file management
//
// # Key Functions
//
// External Tool Runners:
//   - RunActionlintOnFiles() - Run actionlint on multiple lock files
//   - RunZizmorOnFiles() - Run zizmor on multiple lock files
//   - RunPoutineOnDirectory() - Run poutine security scanner on a directory
//   - RunRunnerGuardOnDirectory() - Run runner-guard taint analysis on a directory

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var compileExternalToolsLog = logger.New("cli:compile_external_tools")

// RunActionlintOnFiles runs actionlint on multiple lock files in a single batch.
// This is more efficient than running actionlint once per file.
func RunActionlintOnFiles(ctx context.Context, lockFiles []string, verbose bool, strict bool) error {
	return runBatchLockFileTool("actionlint", lockFiles, verbose, strict, func(files []string, runVerbose bool, runStrict bool) error {
		return runActionlintOnFiles(ctx, files, runVerbose, runStrict)
	})
}

// RunZizmorOnFiles runs zizmor on multiple lock files in a single batch.
// This is more efficient than running zizmor once per file.
func RunZizmorOnFiles(lockFiles []string, verbose bool, strict bool) error {
	return runBatchLockFileTool("zizmor", lockFiles, verbose, strict, runZizmorOnFiles)
}

// RunPoutineOnDirectory runs poutine security scanner once on a directory.
// Poutine scans all workflows in a directory, so it only needs to run once.
func RunPoutineOnDirectory(workflowDir string, verbose bool, strict bool) error {
	return runPoutineOnDirectory(workflowDir, verbose, strict)
}

// RunRunnerGuardOnDirectory runs runner-guard taint analysis scanner once on a directory.
// Runner-guard scans all workflows in a directory, so it only needs to run once.
func RunRunnerGuardOnDirectory(workflowDir string, verbose bool, strict bool) error {
	return runRunnerGuardOnDirectory(workflowDir, verbose, strict)
}

// runBatchLockFileTool runs a batch tool on lock files with uniform error handling
func runBatchLockFileTool(toolName string, lockFiles []string, verbose bool, strict bool, runner func([]string, bool, bool) error) error {
	if len(lockFiles) == 0 {
		compileExternalToolsLog.Printf("No lock files to process with %s", toolName)
		return nil
	}

	compileExternalToolsLog.Printf("Running batch %s on %d lock files", toolName, len(lockFiles))

	return handleBatchToolError(toolName, runner(lockFiles, verbose, strict), strict, verbose)
}

// runBatchDirectoryTool runs a directory-based batch tool with uniform error handling
func runBatchDirectoryTool(toolName string, workflowDir string, verbose bool, strict bool, runner func(string, bool, bool) error) error {
	compileExternalToolsLog.Printf("Running batch %s on directory: %s", toolName, workflowDir)

	return handleBatchToolError(toolName, runner(workflowDir, verbose, strict), strict, verbose)
}

// handleBatchToolError applies uniform strict/non-strict error handling for batch tool results.
// In strict mode, errors are returned wrapped. In non-strict mode, errors are logged as warnings.
func handleBatchToolError(toolName string, err error, strict, verbose bool) error {
	if err == nil {
		return nil
	}
	if strict {
		return fmt.Errorf("%s failed: %w", toolName, err)
	}
	// In non-strict mode, errors are warnings
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%s warnings: %v", toolName, err)))
	}
	return nil
}
