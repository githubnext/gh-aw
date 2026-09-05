//go:build !integration

package cli

import (
	"context"
	"errors"
	"testing"
)

func TestHandleBatchToolErrorPreservesFatalFindingInNonStrictMode(t *testing.T) {
	t.Parallel()
	fatal := &fatalFindingError{err: errors.New("high severity finding")}

	err := handleBatchToolError("zizmor", fatal, false, false)
	if err == nil {
		t.Fatal("expected fatalFindingError to propagate even in non-strict mode, got nil")
	}
	var got *fatalFindingError
	if !errors.As(err, &got) {
		t.Fatalf("expected wrapped fatalFindingError, got %v", err)
	}
}

func TestHandleBatchToolErrorSuppressesRegularErrorInNonStrictMode(t *testing.T) {
	t.Parallel()
	err := handleBatchToolError("zizmor", errors.New("plain warning"), false, false)
	if err != nil {
		t.Fatalf("expected non-strict mode to suppress plain errors, got %v", err)
	}
}

func TestHandleBatchToolErrorPropagatesInStrictMode(t *testing.T) {
	t.Parallel()
	err := handleBatchToolError("zizmor", errors.New("plain warning"), true, false)
	if err == nil {
		t.Fatal("expected strict mode to propagate errors, got nil")
	}
}

func TestRunBatchExternalToolsExecutesSequentialToolsWithoutEarlyAborting(t *testing.T) {
	t.Parallel()

	// Given a context and empty lock file options
	ctx := context.Background()
	config := CompileConfig{
		Actionlint:  true,
		Zizmor:      true,
		Poutine:     true,
		RunnerGuard: true,
		Syft:        true,
		Grype:       true,
		Yamllint:    true,
	}

	opts := batchToolsOptions{
		workflowDir:            t.TempDir(),
		lockFilesForActionlint: []string{},
		lockFilesForZizmor:     []string{},
		lockFilesForDirTools:   []string{},
		lockFilesForSyft:       []string{},
		lockFilesForGrype:      []string{},
		lockFilesForYamllint:   []string{},
	}

	stats := &CompilationStats{}
	var validationResults []ValidationResult

	strictGrantErr, batchToolErr := runBatchExternalTools(ctx, config, opts, stats, &validationResults)
	if strictGrantErr != nil {
		t.Fatalf("expected no strictGrantErr, got %v", strictGrantErr)
	}
	if batchToolErr != nil {
		t.Fatalf("expected no batchToolErr for empty lock files, got %v", batchToolErr)
	}
}
