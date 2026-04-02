package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var addWorkflowCompilationLog = logger.New("cli:add_workflow_compilation")

// compileWorkflow compiles a workflow file without refreshing stop time.
// This is a convenience wrapper around compileWorkflowWithRefresh.
func compileWorkflow(filePath string, verbose bool, quiet bool, engineOverride string) error {
	return compileWorkflowWithRefresh(filePath, verbose, quiet, engineOverride, false)
}

// compileWorkflowWithRefresh compiles a workflow file with optional stop time refresh.
// This function handles the compilation process and ensures .gitattributes is updated.
func compileWorkflowWithRefresh(filePath string, verbose bool, quiet bool, engineOverride string, refreshStopTime bool) error {
	addWorkflowCompilationLog.Printf("Compiling workflow: file=%s, refresh_stop_time=%v, engine=%s", filePath, refreshStopTime, engineOverride)

	// Create compiler with auto-detected version and action mode
	compiler := workflow.NewCompiler(
		workflow.WithVerbose(verbose),
		workflow.WithEngineOverride(engineOverride),
	)

	compiler.SetRefreshStopTime(refreshStopTime)
	compiler.SetQuiet(quiet)
	if err := CompileWorkflowWithValidation(compiler, filePath, verbose, false, false, false, false, false); err != nil {
		addWorkflowCompilationLog.Printf("Compilation failed: %v", err)
		return err
	}

	addWorkflowCompilationLog.Print("Compilation completed successfully")

	// Ensure .gitattributes marks .lock.yml files as generated
	if _, err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
		}
	}

	// Note: Instructions are only written when explicitly requested via the compile command flag
	// This helper function is used in contexts where instructions should not be automatically written

	return nil
}

// compileWorkflowWithTracking compiles a workflow and tracks generated files.
// This is a convenience wrapper around compileWorkflowWithTrackingAndRefresh.
func compileWorkflowWithTracking(filePath string, verbose bool, quiet bool, engineOverride string, tracker *FileTracker) error {
	return compileWorkflowWithTrackingAndRefresh(filePath, verbose, quiet, engineOverride, tracker, false)
}

// compileWorkflowWithTrackingAndRefresh compiles a workflow, tracks generated files, and optionally refreshes stop time.
// This function ensures that the file tracker records all files created or modified during compilation.
func compileWorkflowWithTrackingAndRefresh(filePath string, verbose bool, quiet bool, engineOverride string, tracker *FileTracker, refreshStopTime bool) error {
	addWorkflowCompilationLog.Printf("Compiling workflow with tracking: file=%s, refresh_stop_time=%v", filePath, refreshStopTime)

	// Generate the expected lock file path
	lockFile := stringutil.MarkdownToLockFile(filePath)

	// Check if lock file exists before compilation
	lockFileExists := false
	if _, err := os.Stat(lockFile); err == nil {
		lockFileExists = true
	}

	addWorkflowCompilationLog.Printf("Lock file %s exists: %v", lockFile, lockFileExists)

	// Check if .gitattributes exists before compilation so we know whether to
	// use TrackCreated or TrackModified if ensureGitAttributes modifies it later.
	gitRoot, gitRootErr := gitutil.FindGitRoot()
	gitAttributesPath := ""
	gitAttributesExisted := false
	if gitRootErr == nil {
		gitAttributesPath = filepath.Join(gitRoot, ".gitattributes")
		if _, err := os.Stat(gitAttributesPath); err == nil {
			gitAttributesExisted = true
		}
	}

	// Track the lock file before compilation
	if lockFileExists {
		tracker.TrackModified(lockFile)
	} else {
		tracker.TrackCreated(lockFile)
	}

	// Create compiler with auto-detected version and action mode
	compiler := workflow.NewCompiler(
		workflow.WithVerbose(verbose),
		workflow.WithEngineOverride(engineOverride),
	)
	compiler.SetFileTracker(tracker)
	compiler.SetRefreshStopTime(refreshStopTime)
	compiler.SetQuiet(quiet)
	if err := CompileWorkflowWithValidation(compiler, filePath, verbose, false, false, false, false, false); err != nil {
		return err
	}

	// Ensure .gitattributes marks .lock.yml files as generated; only track it if it was actually
	// modified. Errors here are non-fatal — gitattributes update failure does not prevent the
	// compiled workflow from being usable.
	if updated, err := ensureGitAttributes(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
		}
	} else if updated && gitRootErr == nil {
		if gitAttributesExisted {
			tracker.TrackModified(gitAttributesPath)
		} else {
			tracker.TrackCreated(gitAttributesPath)
		}
	}

	return nil
}

// compileDispatchWorkflowDependencies compiles any dispatch-workflow .md dependencies of
// workflowFile that are present locally but lack a corresponding .lock.yml. This must be
// called before compiling the main workflow, because the dispatch-workflow validator
// requires every referenced .md workflow to have an up-to-date .lock.yml.
func compileDispatchWorkflowDependencies(workflowFile string, verbose, quiet bool, engineOverride string, tracker *FileTracker) {
	// Parse the merged safe-outputs to get the canonical list of dispatch-workflow names.
	compiler := workflow.NewCompiler()
	data, err := compiler.ParseWorkflowFile(workflowFile)
	if err != nil || data == nil || data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return
	}

	workflowsDir := filepath.Dir(workflowFile)

	for _, name := range data.SafeOutputs.DispatchWorkflow.Workflows {
		mdPath := filepath.Join(workflowsDir, name+".md")
		lockPath := stringutil.MarkdownToLockFile(mdPath)

		// Only compile if the .md is present but the .lock.yml is absent.
		if _, mdErr := os.Stat(mdPath); mdErr != nil {
			continue // .md doesn't exist locally
		}
		if _, lockErr := os.Stat(lockPath); lockErr == nil {
			continue // .lock.yml already exists, nothing to do
		}

		addWorkflowCompilationLog.Printf("Compiling dispatch-workflow dependency: %s", mdPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling dispatch-workflow dependency: "+mdPath))
		}

		var compileErr error
		if tracker != nil {
			compileErr = compileWorkflowWithTracking(mdPath, verbose, quiet, engineOverride, tracker)
		} else {
			compileErr = compileWorkflow(mdPath, verbose, quiet, engineOverride)
		}
		if compileErr != nil {
			// Best-effort: log and continue so the main workflow can still give a clear error.
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to compile dispatch-workflow dependency %s: %v", mdPath, compileErr)))
			}
		}
	}
}

// This function preserves the existing frontmatter formatting while adding the source field.
func addSourceToWorkflow(content, source string) (string, error) {
	// Use shared frontmatter logic that preserves formatting
	return addFieldToFrontmatter(content, "source", source)
}

// addEngineToWorkflow adds or updates the engine field in the workflow's frontmatter.
// This function preserves the existing frontmatter formatting while setting the engine field.
func addEngineToWorkflow(content, engine string) (string, error) {
	// Use shared frontmatter logic that preserves formatting
	return addFieldToFrontmatter(content, "engine", engine)
}
