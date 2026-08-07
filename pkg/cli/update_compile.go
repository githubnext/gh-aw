package cli

import (
	"context"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var updateCompileLog = logger.New("cli:update_compile")

// updateCompilationError identifies failures from the shared compile command path.
type updateCompilationError struct {
	err error
}

func (e *updateCompilationError) Error() string {
	return e.err.Error()
}

func (e *updateCompilationError) Unwrap() error {
	return e.err
}

// compileWorkflowsForUpdate uses the same configuration and orchestration as the
// compile command, with only update's matching engine, directory, verbosity, and
// approval options applied.
func compileWorkflowsForUpdate(
	ctx context.Context,
	workflowFiles []string,
	workflowsDir string,
	engineOverride string,
	verbose bool,
	approve bool,
) error {
	config := newUpdateCompileConfig(workflowFiles, workflowsDir, engineOverride, verbose, approve)

	updateCompileLog.Printf("Compiling %d workflow file(s) for update (dir=%q, engineOverride=%q)", len(workflowFiles), workflowsDir, engineOverride)
	if _, err := CompileWorkflows(ctx, config); err != nil {
		updateCompileLog.Printf("Compilation failed: %v", err)
		return &updateCompilationError{err: fmt.Errorf("compile workflows: %w", err)}
	}
	return nil
}

func newUpdateCompileConfig(
	workflowFiles []string,
	workflowsDir string,
	engineOverride string,
	verbose bool,
	approve bool,
) CompileConfig {
	return CompileConfig{
		MarkdownFiles:  workflowFiles,
		Verbose:        verbose,
		EngineOverride: engineOverride,
		WorkflowDir:    workflowsDir,
		Approve:        approve,
	}
}
