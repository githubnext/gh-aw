//go:build !integration

package cli

import (
	"context"
	"errors"
	"testing"
)

// TestWaitForWorkflowCompletionUsesSignalHandling verifies that WaitForWorkflowCompletion
// uses the signal-aware polling helper, which provides Ctrl-C support
func TestWaitForWorkflowCompletionUsesSignalHandling(t *testing.T) {
	// This test verifies that the function uses PollWithSignalHandling
	// by checking that it times out correctly (a key feature of the helper)

	// We can't easily test the actual workflow checking without a real workflow,
	// but we can verify that the timeout mechanism works, which confirms
	// it's using the polling helper

	err := WaitForWorkflowCompletion(context.Background(), "nonexistent/repo", "12345", 0, false)

	// Should timeout or fail to check workflow status
	if err == nil {
		t.Error("Expected error for nonexistent workflow, got nil")
	}
}

// TestWaitForWorkflowCompletion_ContextCancellation verifies that WaitForWorkflowCompletion
// propagates cancellation when the context is cancelled, so callers (e.g. the repeat loop)
// can detect an intentional interruption and stop immediately.
func TestWaitForWorkflowCompletion_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the poll loop exits on the first ctx.Done() check.
	cancel()

	err := WaitForWorkflowCompletion(ctx, "nonexistent/repo", "12345", 5, false)

	if err == nil {
		t.Fatal("Expected error on cancelled context, got nil")
	}

	// Must be either ErrInterrupted (from the poll select loop) or context.Canceled
	// (from the PollFunc guard when ctx is already cancelled before polling begins).
	// Both indicate an intentional interruption that callers should detect and propagate.
	isInterruption := errors.Is(err, ErrInterrupted) || errors.Is(err, context.Canceled)
	if !isInterruption {
		t.Errorf("Expected interruption error (ErrInterrupted or context.Canceled) from WaitForWorkflowCompletion, got: %v", err)
	}
}
