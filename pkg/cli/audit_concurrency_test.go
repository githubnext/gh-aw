package cli

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/sync/errgroup"
)

func TestCollectAuditAnalysisResultsReturnsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collectAuditAnalysisResults(ctx, WorkflowRun{}, t.TempDir(), false, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestRunAuditAnalysisSoftFailuresRemainNonFatal(t *testing.T) {
	g, gctx := errgroup.WithContext(context.Background())
	called := false

	runAuditAnalysis(g, gctx, false, "test", "test warning", func(v int) {
		called = true
	}, func() (int, error) {
		return 0, errors.New("soft failure")
	})

	if err := g.Wait(); err != nil {
		t.Fatalf("expected nil errgroup error for soft failure, got %v", err)
	}
	if called {
		t.Fatal("expected setter not to be called on soft failure")
	}
}
